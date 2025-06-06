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
	config.Init()
	cfg := config.Get()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	shutdown := logger.Init()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdown(shutdownCtx)
	}()

	db.Init(ctx)

	// Set up HTTP handler functions
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		rCtx, span := logger.StartSpan(r.Context(), "callback")
		defer span.End()

		if r.Method != "POST" {
			logger.Warn(rCtx, "Received non-standard LINE callback request", "method", r.Method, "path", r.URL.Path)
		}

		bot, err := linebot.New(
			cfg.Line.ChannelSecret,
			cfg.Line.ChannelAccessToken,
		)
		if err != nil {
			logger.Error(rCtx, "Failed to initialize LINE Bot", "error", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Parse LINE request
		events, err := bot.ParseRequest(r)
		if err != nil {
			if err == linebot.ErrInvalidSignature {
				logger.Warn(rCtx, "Invalid LINE signature")
				w.WriteHeader(http.StatusBadRequest)
			} else {
				logger.Error(rCtx, "Failed to parse LINE request", "error", err.Error())
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}

		// Handle webhook verification
		if len(events) == 0 {
			logger.Info(ctx, "Server started", "port", cfg.Port)
			w.WriteHeader(http.StatusOK)
			return
		}

		// Handle messages
		for _, event := range events {
			if event.Type == linebot.EventTypeMessage {
				if message, ok := event.Message.(*linebot.TextMessage); ok {
					logger.Info(rCtx, "Received message",
						"user_id", event.Source.UserID,
						"message", message.Text,
					)

					reply := handler.HandleMessage(rCtx, event.Source.UserID, message.Text)

					if _, err := bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(reply)).Do(); err != nil {
						logger.Error(rCtx, "Failed to reply message", "error", err.Error())
					}
				}
			}
		}

		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Start server
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: http.DefaultServeMux,
	}

	// Start server asynchronously
	go func() {
		logger.Info(ctx, "Server started", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal(ctx, "Server failed to start", "error", err.Error())
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()

	logger.Info(ctx, "Shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error(ctx, "Server shutdown failed", "error", err.Error())
	}

	logger.Info(ctx, "Server stopped")
}
