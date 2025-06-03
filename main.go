package main

import (
	"log"
	"net/http"
	"os"

	"github.com/line/line-bot-sdk-go/v7/linebot"
)

func main() {
	bot, err := linebot.New(
		os.Getenv("LINE_CHANNEL_SECRET"),
		os.Getenv("LINE_CHANNEL_ACCESS_TOKEN"),
	)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/callback", func(w http.ResponseWriter, req *http.Request) {
		events, err := bot.ParseRequest(req)
		if err != nil {
			if err == linebot.ErrInvalidSignature {
				w.WriteHeader(400)
			} else {
				w.WriteHeader(500)
			}
			return
		}

		for _, event := range events {
			if event.Type == linebot.EventTypeMessage {
				if message, ok := event.Message.(*linebot.TextMessage); ok {
					// 在這裡處理使用者的訊息，例如記帳功能
					if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("收到: "+message.Text)).Do(); err != nil {
						log.Print(err)
					}
				}
			}
		}
	})

	log.Println("Server started at :8080")
	http.ListenAndServe(":8080", nil)
}
