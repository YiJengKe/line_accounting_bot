package handler

import (
	"accountingbot/logger"
	"accountingbot/models"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func WebhookHandler(w http.ResponseWriter, r *http.Request) {
	// 建立追蹤 span
	ctx, span := logger.StartSpan(r.Context(), "WebhookHandler")
	defer span.End()

	userID := "demo_user" // 單一用戶模擬
	logger.Info(ctx, "接收網頁請求", "user_id", userID)

	r.ParseForm()
	text := r.FormValue("message")
	response := HandleMessage(ctx, userID, text)
	fmt.Fprint(w, response)
}

// HandleMessage 處理用戶輸入的訊息
func HandleMessage(ctx context.Context, userID, text string) string {
	// 建立追蹤 span
	ctx, span := logger.StartSpan(ctx, "HandleMessage")
	defer span.End()

	// 記錄輸入訊息
	logger.Info(ctx, "處理訊息", "user_id", userID, "message", text)

	tokens := strings.Fields(text)
	if len(tokens) == 0 {
		return "請輸入有效的指令。"
	}

	switch {
	case tokens[0] == "新增類別" && len(tokens) >= 3:
		return handleAddCategory(ctx, userID, tokens[1], tokens[2])

	case tokens[0] == "修改類別" && len(tokens) == 3:
		return handleUpdateCategory(ctx, userID, tokens[1], tokens[2])

	case tokens[0] == "刪除類別" && len(tokens) == 2:
		return handleDeleteCategory(ctx, userID, tokens[1])

	case tokens[0] == "已設定類別":
		return handleListCategories(ctx, userID)

	case len(tokens) == 2:
		return handleQuickTransaction(ctx, userID, tokens[0], tokens[1])

	case tokens[0] == "修改" && len(tokens) == 4:
		return handleUpdateTransaction(ctx, userID, tokens[1], tokens[2], tokens[3])

	case tokens[0] == "刪除" && len(tokens) == 3:
		return handleDeleteTransaction(ctx, userID, tokens[1], tokens[2])

	case tokens[0] == "結算":
		return handleMonthlySummary(ctx, userID, tokens)

	case tokens[0] == "指令大全":
		return getHelpText(ctx)
	}

	logger.Info(ctx, "未識別的指令", "command", tokens[0])
	return "❓ 指令不正確，請重新輸入。"
}

func handleAddCategory(ctx context.Context, userID, typeName, name string) string {
	ctx, span := logger.StartSpan(ctx, "handleAddCategory")
	defer span.End()

	logger.Info(ctx, "新增類別", "type", typeName, "name", name)

	// 檢查類別名稱是否已存在
	exists, err := models.CheckCategoryExists(ctx, userID, name, typeName)
	if err != nil {
		logger.Error(ctx, "檢查類別存在性失敗", "error", err.Error())
		return "❌ 類別檢查失敗，請稍後再試。"
	}

	if exists {
		logger.Warn(ctx, "類別已存在", "name", name)
		return fmt.Sprintf("❌ 類別 %s 已存在，請使用其他名稱。", name)
	}

	// 使用 models.AddCategory 新增類別
	err = models.AddCategory(ctx, userID, name, typeName)
	if err != nil {
		logger.Error(ctx, "新增類別失敗", "error", err.Error())
		return "❌ 新增類別失敗，請稍後再試。"
	}

	logger.Info(ctx, "類別新增成功", "name", name, "type", typeName)
	return fmt.Sprintf("✅ 類別 %s 已新增！", name)
}

// handleUpdateCategory 處理修改類別的指令
func handleUpdateCategory(ctx context.Context, userID, oldName, newName string) string {
	ctx, span := logger.StartSpan(ctx, "handleUpdateCategory")
	defer span.End()

	logger.Info(ctx, "修改類別", "old_name", oldName, "new_name", newName)

	// 使用 models.UpdateCategory 修改類別
	updated, err := models.UpdateCategory(ctx, userID, oldName, newName)
	if err != nil {
		logger.Error(ctx, "修改類別失敗", "error", err.Error())
		return "❌ 修改失敗，請稍後再試。"
	}

	if !updated {
		logger.Warn(ctx, "找不到要修改的類別", "name", oldName)
		return "❌ 類別不存在。"
	}

	logger.Info(ctx, "類別修改成功", "old_name", oldName, "new_name", newName)
	return fmt.Sprintf("✏️ 類別已修改為：%s", newName)
}

// handleDeleteCategory 處理刪除類別的指令
func handleDeleteCategory(ctx context.Context, userID, name string) string {
	ctx, span := logger.StartSpan(ctx, "handleDeleteCategory")
	defer span.End()

	logger.Info(ctx, "刪除類別", "name", name)

	// 使用 models.DeleteCategory 刪除類別
	deleted, err := models.DeleteCategory(ctx, userID, name)
	if err != nil {
		logger.Error(ctx, "刪除類別失敗", "error", err.Error())
		return "❌ 刪除失敗，請稍後再試。"
	}

	if !deleted {
		logger.Warn(ctx, "找不到要刪除的類別", "name", name)
		return "❌ 類別不存在。"
	}

	logger.Info(ctx, "類別刪除成功", "name", name)
	return fmt.Sprintf("🗑️ 類別 %s 已刪除", name)
}

// handleListCategories 處理列出類別的指令
func handleListCategories(ctx context.Context, userID string) string {
	ctx, span := logger.StartSpan(ctx, "handleListCategories")
	defer span.End()

	logger.Info(ctx, "列出類別")

	// 使用 models.GetCategoriesByType 取得類別
	categoriesByType, err := models.GetCategoriesByType(ctx, userID)
	if err != nil {
		logger.Error(ctx, "類別查詢失敗", "error", err.Error())
		return "❌ 類別查詢失敗，請稍後再試。"
	}

	incomeList := categoriesByType["收入"]
	expenseList := categoriesByType["支出"]

	if len(incomeList) == 0 && len(expenseList) == 0 {
		logger.Warn(ctx, "尚未有任何類別")
		return "⚠️ 你尚未新增任何類別。"
	}

	response := "📂 你的可用類別：\n"
	if len(incomeList) > 0 {
		response += "💰 收入類別：\n"
		for _, name := range incomeList {
			response += fmt.Sprintf("・%s\n", name)
		}
	}
	if len(expenseList) > 0 {
		response += "💸 支出類別：\n"
		for _, name := range expenseList {
			response += fmt.Sprintf("・%s\n", name)
		}
	}

	logger.Info(ctx, "取得類別列表",
		"income_count", len(incomeList),
		"expense_count", len(expenseList))
	return response
}

// handleQuickTransaction 處理快速記帳的指令
func handleQuickTransaction(ctx context.Context, userID, categoryName, amountStr string) string {
	ctx, span := logger.StartSpan(ctx, "handleQuickTransaction")
	defer span.End()

	logger.Info(ctx, "快速記帳", "category", categoryName, "amount", amountStr)

	amount, err := strconv.Atoi(amountStr)
	if err != nil {
		logger.Warn(ctx, "金額格式錯誤", "amount", amountStr)
		return "金額格式錯誤"
	}

	// 取得類別的 ID 和 Type
	categoryID, categoryType, err := models.GetCategoryIdAndType(ctx, userID, categoryName)
	if err != nil {
		logger.Warn(ctx, "類別不存在", "category", categoryName)
		return "❌ 類別不存在，請先新增。"
	}

	// 新增交易紀錄
	transaction, err := models.AddTransaction(ctx, userID, categoryID, categoryType, amount)
	if err != nil {
		logger.Error(ctx, "記錄交易失敗", "error", err.Error())
		return "記錄失敗，請稍後再試。"
	}

	logger.Info(ctx, "交易記錄成功",
		"transaction_id", transaction.ID,
		"type", categoryType,
		"amount", amount,
		"category", categoryName)
	return fmt.Sprintf("✅ %s $%d 類別：%s 已記錄！", categoryType, amount, categoryName)
}

// handleUpdateTransaction 處理修改交易的指令
func handleUpdateTransaction(ctx context.Context, userID, category, oldAmountStr, newAmountStr string) string {
	ctx, span := logger.StartSpan(ctx, "handleUpdateTransaction")
	defer span.End()

	logger.Info(ctx, "修改交易",
		"category", category,
		"old_amount", oldAmountStr,
		"new_amount", newAmountStr)

	oldAmount, err1 := strconv.Atoi(oldAmountStr)
	newAmount, err2 := strconv.Atoi(newAmountStr)
	if err1 != nil || err2 != nil {
		logger.Warn(ctx, "金額格式錯誤",
			"old_amount", oldAmountStr,
			"new_amount", newAmountStr)
		return "金額格式錯誤，請輸入數字。"
	}

	// 尋找交易記錄
	transactionID, err := models.FindTransactionID(ctx, userID, category, oldAmount)
	if err != nil {
		logger.Warn(ctx, "找不到符合條件的交易記錄",
			"category", category,
			"amount", oldAmount)
		return "❌ 找不到符合條件的紀錄。"
	}

	// 更新交易
	err = models.UpdateTransaction(ctx, transactionID, newAmount)
	if err != nil {
		logger.Error(ctx, "修改交易失敗", "error", err.Error())
		return "❌ 修改失敗，請稍後再試。"
	}

	logger.Info(ctx, "交易修改成功",
		"transaction_id", transactionID,
		"category", category,
		"old_amount", oldAmount,
		"new_amount", newAmount)
	return fmt.Sprintf("✅ 已將 %s 的金額從 $%d 修改為 $%d。", category, oldAmount, newAmount)
}

// handleDeleteTransaction 處理刪除交易的指令
func handleDeleteTransaction(ctx context.Context, userID, category, amountStr string) string {
	ctx, span := logger.StartSpan(ctx, "handleDeleteTransaction")
	defer span.End()

	logger.Info(ctx, "刪除交易", "category", category, "amount", amountStr)

	amount, err := strconv.Atoi(amountStr)
	if err != nil {
		logger.Warn(ctx, "金額格式錯誤", "amount", amountStr)
		return "金額格式錯誤，請輸入數字。"
	}

	// 尋找交易記錄
	transactionID, err := models.FindTransactionID(ctx, userID, category, amount)
	if err != nil {
		logger.Warn(ctx, "找不到符合條件的交易記錄",
			"category", category,
			"amount", amount)
		return "❌ 找不到符合條件的紀錄。"
	}

	// 刪除交易
	err = models.DeleteTransaction(ctx, transactionID)
	if err != nil {
		logger.Error(ctx, "刪除交易失敗", "error", err.Error())
		return "❌ 刪除失敗，請稍後再試。"
	}

	logger.Info(ctx, "交易刪除成功",
		"transaction_id", transactionID,
		"category", category,
		"amount", amount)
	return fmt.Sprintf("🗑️ 已刪除 %s $%d 的紀錄。", category, amount)
}

// handleMonthlySummary 處理月結算的指令
func handleMonthlySummary(ctx context.Context, userID string, tokens []string) string {
	ctx, span := logger.StartSpan(ctx, "handleMonthlySummary")
	defer span.End()

	var targetMonth time.Time
	var monthSpec string

	if len(tokens) == 3 {
		// 嘗試解析格式：「結算 2025年 5月」
		yearStr := strings.TrimSuffix(tokens[1], "年")
		monthStr := strings.TrimSuffix(tokens[2], "月")
		monthSpec = yearStr + "年" + monthStr + "月"

		logger.Info(ctx, "指定月份結算", "year", yearStr, "month", monthStr)
		year, yErr := strconv.Atoi(yearStr)
		month, mErr := strconv.Atoi(monthStr)

		if yErr != nil || mErr != nil || month < 1 || month > 12 {
			logger.Warn(ctx, "結算格式錯誤", "year", yearStr, "month", monthStr)
			return "⚠️ 結算格式錯誤，請使用：結算 或 結算 2025年 5月"
		}

		// 建立對應月份起始時間（UTC）
		targetMonth = time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	} else {
		// 預設為當月
		targetMonth = time.Now().UTC()
		monthSpec = "當月"
		logger.Info(ctx, "當月結算")
	}

	// 使用 models.GetMonthlySummary 獲取月報表
	summary, err := models.GetMonthlySummary(ctx, userID, targetMonth)
	if err != nil {
		logger.Error(ctx, "取得報表失敗", "error", err.Error())
		return "取得報表失敗，請稍後再試。"
	}

	// 創建基本報表頭
	result := fmt.Sprintf("📊 %d年%d月\n收入：$%d\n支出：$%d\n\n",
		targetMonth.Year(), targetMonth.Month(), summary.IncomeTotal, summary.ExpenseTotal)

	// 分別整理收入和支出類別
	incomeCategories := make(map[string]int)
	expenseCategories := make(map[string]int)

	// 從 models 中獲取類別及其類型
	categoriesInfo, err := models.GetCategoriesInfo(ctx, userID)
	if err != nil {
		logger.Warn(ctx, "無法獲取類別資訊", "error", err.Error())
		// 繼續執行，因為我們至少有金額數據
	}

	// 根據類別類型分組
	for cat, amt := range summary.CategoryTotals {
		// 檢查我們是否有此類別的類型資訊
		if catType, ok := categoriesInfo[cat]; ok {
			if catType == "收入" {
				incomeCategories[cat] = amt
			} else {
				expenseCategories[cat] = amt
			}
		} else {
			// 如果沒有類型資訊，根據金額判斷（暫時解決方案）
			if amt > 0 {
				incomeCategories[cat] = amt
			} else {
				expenseCategories[cat] = amt
			}
		}
	}

	// 添加收入區塊
	if len(incomeCategories) > 0 {
		result += "💰 收入明細：\n"
		for cat, amt := range incomeCategories {
			result += fmt.Sprintf("・%s：$%d\n", cat, amt)
		}
		result += "\n"
	}

	// 添加支出區塊
	if len(expenseCategories) > 0 {
		result += "💸 支出明細：\n"
		for cat, amt := range expenseCategories {
			result += fmt.Sprintf("・%s：$%d\n", cat, amt)
		}
		result += "\n"
	}

	// 添加淨收益
	result += fmt.Sprintf("💰 淨收益：$%d", summary.IncomeTotal-summary.ExpenseTotal)

	logger.Info(ctx, "結算完成",
		"month_spec", monthSpec,
		"income", summary.IncomeTotal,
		"expense", summary.ExpenseTotal,
		"income_categories", len(incomeCategories),
		"expense_categories", len(expenseCategories))

	return result
}

// getHelpText 取得指令說明文字
func getHelpText(ctx context.Context) string {
	ctx, span := logger.StartSpan(ctx, "getHelpText")
	defer span.End()

	logger.Info(ctx, "顯示指令說明")

	return `📖 指令大全：

📂 類別管理
- 新增類別 支出/收入 類別名稱
- 修改類別 舊名稱 新名稱
- 刪除類別 名稱
- 已設定類別（查看目前所有可用類別）

📝 記帳與查詢
- 類別名稱 金額（快速記帳）
- 修改 類別名稱 原金額 新金額
- 刪除 類別名稱 金額

📊 月結報表
- 結算 2025年 5月 (指定年月)`
}
