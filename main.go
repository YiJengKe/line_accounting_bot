package main

import (
	"log"
	"net/http"
	"os"

	"github.com/line/line-bot-sdk-go/v7/linebot"
)

func main() {
	http.HandleFunc("/callback", func(w http.ResponseWriter, req *http.Request) {
		bot, err := linebot.New(
			os.Getenv("LINE_CHANNEL_SECRET"),
			os.Getenv("LINE_CHANNEL_ACCESS_TOKEN"),
		)
		if err != nil {
			log.Println("LINE bot init error:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		events, err := bot.ParseRequest(req)
		if err != nil {
			if err == linebot.ErrInvalidSignature {
				log.Println("Invalid signature")
				w.WriteHeader(http.StatusBadRequest)
			} else {
				log.Println("ParseRequest error:", err)
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}

		// 關鍵：LINE 驗證 webhook 時，會送空 events 陣列
		if len(events) == 0 {
			log.Println("Webhook verification ping received. Returning 200.")
			w.WriteHeader(http.StatusOK)
			return
		}

		for _, event := range events {
			if event.Type == linebot.EventTypeMessage {
				if message, ok := event.Message.(*linebot.TextMessage); ok {
					reply := "收到指令：" + message.Text
					if _, err := bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(reply)).Do(); err != nil {
						log.Println("ReplyMessage error:", err)
					}
				}
			}
		}

		w.WriteHeader(http.StatusOK) // 一定要回 200
	})

	log.Println("Server started at :8080")
	http.ListenAndServe(":8080", nil)
}
