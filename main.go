package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"accountingbot/config"
	"accountingbot/db"
	"accountingbot/handler"
	"accountingbot/logger"

	"github.com/line/line-bot-sdk-go/v7/linebot"
)

func main() {
	// 設定 context 和信號處理
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 初始化日誌和追蹤系統
	shutdown := logger.Init()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdown(shutdownCtx)
	}()

	// 初始化設定
	config.Init()
	cfg := config.Get()

	// 初始化資料庫
	db.Init(ctx)

	// 建立 HTTP 處理函數
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		// 為 HTTP 請求建立 span
		rCtx, span := logger.StartSpan(r.Context(), "callback")
		defer span.End()

		// 紀錄請求資訊
		if r.Method != "POST" {
			logger.Warn(rCtx, "收到非標準 LINE 回調請求", "method", r.Method, "path", r.URL.Path)
		}

		// 建立 LINE bot client
		bot, err := linebot.New(
			cfg.Line.ChannelSecret,
			cfg.Line.ChannelAccessToken,
		)
		if err != nil {
			logger.Error(rCtx, "LINE Bot 初始化失敗", "error", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// 解析 LINE 請求
		events, err := bot.ParseRequest(r)
		if err != nil {
			if err == linebot.ErrInvalidSignature {
				logger.Warn(rCtx, "無效的 LINE 簽名")
				w.WriteHeader(http.StatusBadRequest)
			} else {
				logger.Error(rCtx, "解析 LINE 請求失敗", "error", err.Error())
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}

		// 處理 webhook 驗證
		if len(events) == 0 {
			logger.Info(ctx, "伺服器已啟動", "port", cfg.Port)
			w.WriteHeader(http.StatusOK)
			return
		}

		// 處理訊息
		for _, event := range events {
			if event.Type == linebot.EventTypeMessage {
				if message, ok := event.Message.(*linebot.TextMessage); ok {
					logger.Info(rCtx, "收到訊息", // 使用上層的 rCtx
						"user_id", event.Source.UserID,
						"message", message.Text,
					)

					reply := handler.HandleMessage(rCtx, event.Source.UserID, message.Text) // 使用上層的 rCtx

					if _, err := bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(reply)).Do(); err != nil {
						logger.Error(rCtx, "回覆訊息失敗", "error", err.Error())
					}
				}
			}
		}

		w.WriteHeader(http.StatusOK)
	})

	// 健康檢查端點
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// 啟動伺服器
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: http.DefaultServeMux,
	}

	// 非同步啟動伺服器
	go func() {
		logger.Info(ctx, "伺服器已啟動", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal(ctx, "伺服器啟動失敗", "error", err.Error())
		}
	}()

	// 等待關閉信號
	<-ctx.Done()

	// 優雅關閉
	logger.Info(ctx, "正在關閉伺服器...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error(ctx, "伺服器關閉失敗", "error", err.Error())
	}

	logger.Info(ctx, "伺服器已關閉")
}
