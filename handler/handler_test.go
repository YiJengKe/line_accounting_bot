package handler

import (
	"accountingbot/db"
	"accountingbot/logger"
	"context"
	"strings"
	"testing"
	"time"
)

func TestHandleMessageDirectly(t *testing.T) {
	ctx := context.Background()

	shutdown := logger.Init()
	defer func() {
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		_ = shutdown(ctx)
	}()

	testDBName := db.SetupTestDB(ctx)
	defer db.CleanupTestDB(ctx, testDBName)

	commands := []struct {
		name     string
		input    string
		contains string // Expected substring in the response
	}{
		// Basic command tests
		{
			name:     "空輸入",
			input:    "",
			contains: "請輸入有效的指令。",
		},
		{
			name:     "無效指令",
			input:    "無效指令",
			contains: "❓ 指令不正確，請重新輸入。",
		},

		// Category management tests
		{
			name:     "新增收入類別",
			input:    "新增類別 收入 獎金",
			contains: "✅ 類別 獎金 已新增！",
		},
		{
			name:     "新增支出類別",
			input:    "新增類別 支出 午餐",
			contains: "✅ 類別 午餐 已新增！",
		},
		{
			name:     "新增支出類別",
			input:    "新增類別 支出 餐費",
			contains: "✅ 類別 餐費 已新增！",
		},
		{
			name:     "新增已存在類別",
			input:    "新增類別 收入 獎金",
			contains: "❌ 類別 獎金 已存在，請使用其他名稱。",
		},
		{
			name:     "查看類別列表",
			input:    "已設定類別",
			contains: "獎金",
		},
		{
			name:     "修改類別名稱",
			input:    "修改類別 餐費 伙食費",
			contains: "✏️ 類別已修改為：伙食費",
		},
		{
			name:     "刪除類別",
			input:    "刪除類別 伙食費",
			contains: "🗑️ 類別 伙食費 已刪除",
		},
		{
			name:     "刪除不存在類別",
			input:    "刪除類別 不存在類別",
			contains: "❌ 類別不存在",
		},

		// Transaction record tests
		{
			name:     "快速記帳-支出",
			input:    "午餐 150",
			contains: "✅ 支出 $150 類別：午餐 已記錄！",
		},
		{
			name:     "快速記帳-收入",
			input:    "獎金 5000",
			contains: "✅ 收入 $5000 類別：獎金 已記錄！",
		},
		{
			name:     "快速記帳-類別不存在",
			input:    "不存在類別 100",
			contains: "❌ 類別不存在，請先新增。",
		},
		{
			name:     "修改交易紀錄",
			input:    "修改 午餐 150 200",
			contains: "✅ 已將 午餐 的金額從 $150 修改為 $200。",
		},
		{
			name:     "修改不存在的交易紀錄",
			input:    "修改 午餐 999 200",
			contains: "❌ 找不到符合條件的紀錄。",
		},
		{
			name:     "刪除交易紀錄",
			input:    "刪除 午餐 200",
			contains: "🗑️ 已刪除 午餐 $200 的紀錄。",
		},
		{
			name:     "刪除不存在的交易紀錄",
			input:    "刪除 午餐 999",
			contains: "❌ 找不到符合條件的紀錄。",
		},

		// Monthly summary report tests
		{
			name:     "當月結算",
			input:    "結算",
			contains: "獎金：$5000",
		},
		{
			name:     "指定月份結算",
			input:    "結算 2025年 5月",
			contains: "支出：$0",
		},
		{
			name:     "無效月份格式",
			input:    "結算 無效 月份",
			contains: "⚠️ 結算格式錯誤，請使用：結算 或 結算 2025年 5月",
		},

		// documentation test
		{
			name:     "取得說明",
			input:    "指令大全",
			contains: "📖 指令大全",
		},
	}

	userID := "test_user"

	for _, cmd := range commands {
		t.Run(cmd.name, func(t *testing.T) {
			response := HandleMessage(ctx, userID, cmd.input)

			if !strings.Contains(response, cmd.contains) {
				t.Errorf("Response %q does not contain expected %q", response, cmd.contains)
			}
		})
	}
}
