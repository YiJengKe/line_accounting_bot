package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"accountingbot/db"
)

func WebhookHandler(w http.ResponseWriter, r *http.Request) {
	userID := "demo_user" // 模擬單一用戶
	r.ParseForm()
	text := r.FormValue("message")

	response := HandleMessage(userID, text)
	fmt.Fprint(w, response)
}

func HandleMessage(userID, text string) string {
	tokens := strings.Fields(text)
	if len(tokens) == 0 {
		return "請輸入有效的指令。"
	}

	if tokens[0] == "新增類別" && len(tokens) >= 3 {
		type_ := tokens[1]
		name := tokens[2]
		_, err := db.DB.Exec(`INSERT INTO categories (user_id, name, type) VALUES (?, ?, ?)`, userID, name, type_)
		if err != nil {
			return "新增類別失敗"
		}
		return fmt.Sprintf("✅ 已新增%s類別：%s", type_, name)
	}

	if len(tokens) == 2 {
		category := tokens[0]
		amount, err := strconv.Atoi(tokens[1])
		if err != nil {
			return "金額格式錯誤"
		}
		row := db.DB.QueryRow(`SELECT type FROM categories WHERE user_id = ? AND name = ?`, userID, category)
		var type_ string
		err = row.Scan(&type_)
		if err != nil {
			return "❌ 類別不存在，請先新增。"
		}
		db.DB.Exec(`INSERT INTO transactions (user_id, type, amount, category) VALUES (?, ?, ?, ?)`, userID, type_, amount, category)
		return fmt.Sprintf("✅ %s $%d 類別：%s 已記錄！", type_, amount, category)
	}

	return "請確認輸入格式。"
}
