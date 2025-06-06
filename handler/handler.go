package handler

import (
	"accountingbot/logger"
	"accountingbot/model"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// WebhookHandler handles incoming web requests
func WebhookHandler(w http.ResponseWriter, r *http.Request) {
	ctx, span := logger.StartSpan(r.Context(), "WebhookHandler")
	defer span.End()

	userID := "demo_user"
	logger.Info(ctx, "Received web request", "user_id", userID)

	r.ParseForm()
	text := r.FormValue("message")
	response := HandleMessage(ctx, userID, text)
	fmt.Fprint(w, response)
}

// HandleMessage handles user input messages
func HandleMessage(ctx context.Context, userID, text string) string {
	ctx, span := logger.StartSpan(ctx, "HandleMessage")
	defer span.End()

	logger.Info(ctx, "Processing message", "user_id", userID, "message", text)

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

	logger.Info(ctx, "Unrecognized command", "command", tokens[0])
	return "❓ 指令不正確，請重新輸入。"
}

func handleAddCategory(ctx context.Context, userID, typeName, name string) string {
	ctx, span := logger.StartSpan(ctx, "handleAddCategory")
	defer span.End()

	logger.Info(ctx, "Add category", "type", typeName, "name", name)

	// Check if category name already exists
	exists, err := model.CheckCategoryExists(ctx, userID, name, typeName)
	if err != nil {
		logger.Error(ctx, "Failed to check category existence", "error", err.Error())
		return "❌ 類別檢查失敗，請稍後再試。"
	}

	if exists {
		logger.Warn(ctx, "Category already exists", "name", name)
		return fmt.Sprintf("❌ 類別 %s 已存在，請使用其他名稱。", name)
	}

	// Add category using model.AddCategory
	err = model.AddCategory(ctx, userID, name, typeName)
	if err != nil {
		logger.Error(ctx, "Failed to add category", "error", err.Error())
		return "❌ 新增類別失敗，請稍後再試。"
	}

	logger.Info(ctx, "Category added successfully", "name", name, "type", typeName)
	return fmt.Sprintf("✅ 類別 %s 已新增！", name)
}

// handleUpdateCategory handles the command to update a category
func handleUpdateCategory(ctx context.Context, userID, oldName, newName string) string {
	ctx, span := logger.StartSpan(ctx, "handleUpdateCategory")
	defer span.End()

	logger.Info(ctx, "Update category", "old_name", oldName, "new_name", newName)

	// Update category using model.UpdateCategory
	updated, err := model.UpdateCategory(ctx, userID, oldName, newName)
	if err != nil {
		logger.Error(ctx, "Failed to update category", "error", err.Error())
		return "❌ 修改失敗，請稍後再試。"
	}

	if !updated {
		logger.Warn(ctx, "Category to update not found", "name", oldName)
		return "❌ 類別不存在。"
	}

	logger.Info(ctx, "Category updated successfully", "old_name", oldName, "new_name", newName)
	return fmt.Sprintf("✏️ 類別已修改為：%s", newName)
}

// handleDeleteCategory handles the command to delete a category
func handleDeleteCategory(ctx context.Context, userID, name string) string {
	ctx, span := logger.StartSpan(ctx, "handleDeleteCategory")
	defer span.End()

	logger.Info(ctx, "Delete category", "name", name)

	// Delete category using model.DeleteCategory
	deleted, err := model.DeleteCategory(ctx, userID, name)
	if err != nil {
		logger.Error(ctx, "Failed to delete category", "error", err.Error())
		return "❌ 刪除失敗，請稍後再試。"
	}

	if !deleted {
		logger.Warn(ctx, "Category to delete not found", "name", name)
		return "❌ 類別不存在。"
	}

	logger.Info(ctx, "Category deleted successfully", "name", name)
	return fmt.Sprintf("🗑️ 類別 %s 已刪除", name)
}

// handleListCategories handles the command to list categories
func handleListCategories(ctx context.Context, userID string) string {
	ctx, span := logger.StartSpan(ctx, "handleListCategories")
	defer span.End()

	logger.Info(ctx, "List categories")

	// Get categories using model.GetCategoriesByType
	categoriesByType, err := model.GetCategoriesByType(ctx, userID)
	if err != nil {
		logger.Error(ctx, "Failed to query categories", "error", err.Error())
		return "❌ 類別查詢失敗，請稍後再試。"
	}

	incomeList := categoriesByType["收入"]
	expenseList := categoriesByType["支出"]

	if len(incomeList) == 0 && len(expenseList) == 0 {
		logger.Warn(ctx, "No categories yet")
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

	logger.Info(ctx, "Got category list",
		"income_count", len(incomeList),
		"expense_count", len(expenseList))
	return response
}

// handleQuickTransaction handles the command for quick transaction recording
func handleQuickTransaction(ctx context.Context, userID, categoryName, amountStr string) string {
	ctx, span := logger.StartSpan(ctx, "handleQuickTransaction")
	defer span.End()

	logger.Info(ctx, "Quick transaction", "category", categoryName, "amount", amountStr)

	amount, err := strconv.Atoi(amountStr)
	if err != nil {
		logger.Warn(ctx, "Amount format error", "amount", amountStr)
		return "金額格式錯誤"
	}

	// Get category ID and Type
	categoryID, categoryType, err := model.GetCategoryIdAndType(ctx, userID, categoryName)
	if err != nil {
		logger.Warn(ctx, "Category does not exist", "category", categoryName)
		return "❌ 類別不存在，請先新增。"
	}

	// Add transaction record
	transaction, err := model.AddTransaction(ctx, userID, categoryID, categoryType, amount)
	if err != nil {
		logger.Error(ctx, "Failed to record transaction", "error", err.Error())
		return "記錄失敗，請稍後再試。"
	}

	logger.Info(ctx, "Transaction recorded successfully",
		"transaction_id", transaction.ID,
		"type", categoryType,
		"amount", amount,
		"category", categoryName)
	return fmt.Sprintf("✅ %s $%d 類別：%s 已記錄！", categoryType, amount, categoryName)
}

// handleUpdateTransaction handles the command to update a transaction
func handleUpdateTransaction(ctx context.Context, userID, category, oldAmountStr, newAmountStr string) string {
	ctx, span := logger.StartSpan(ctx, "handleUpdateTransaction")
	defer span.End()

	logger.Info(ctx, "Update transaction",
		"category", category,
		"old_amount", oldAmountStr,
		"new_amount", newAmountStr)

	oldAmount, err1 := strconv.Atoi(oldAmountStr)
	newAmount, err2 := strconv.Atoi(newAmountStr)
	if err1 != nil || err2 != nil {
		logger.Warn(ctx, "Amount format error",
			"old_amount", oldAmountStr,
			"new_amount", newAmountStr)
		return "金額格式錯誤，請輸入數字。"
	}

	// Find transaction record
	transactionID, err := model.FindTransactionID(ctx, userID, category, oldAmount)
	if err != nil {
		logger.Warn(ctx, "No matching transaction record found",
			"category", category,
			"amount", oldAmount)
		return "❌ 找不到符合條件的紀錄。"
	}

	// Update transaction
	err = model.UpdateTransaction(ctx, transactionID, newAmount)
	if err != nil {
		logger.Error(ctx, "Failed to update transaction", "error", err.Error())
		return "❌ 修改失敗，請稍後再試。"
	}

	logger.Info(ctx, "Transaction updated successfully",
		"transaction_id", transactionID,
		"category", category,
		"old_amount", oldAmount,
		"new_amount", newAmount)
	return fmt.Sprintf("✅ 已將 %s 的金額從 $%d 修改為 $%d。", category, oldAmount, newAmount)
}

// handleDeleteTransaction handles the command to delete a transaction
func handleDeleteTransaction(ctx context.Context, userID, category, amountStr string) string {
	ctx, span := logger.StartSpan(ctx, "handleDeleteTransaction")
	defer span.End()

	logger.Info(ctx, "Delete transaction", "category", category, "amount", amountStr)

	amount, err := strconv.Atoi(amountStr)
	if err != nil {
		logger.Warn(ctx, "Amount format error", "amount", amountStr)
		return "金額格式錯誤，請輸入數字。"
	}

	// Find transaction record
	transactionID, err := model.FindTransactionID(ctx, userID, category, amount)
	if err != nil {
		logger.Warn(ctx, "No matching transaction record found",
			"category", category,
			"amount", amount)
		return "❌ 找不到符合條件的紀錄。"
	}

	// Delete transaction
	err = model.DeleteTransaction(ctx, transactionID)
	if err != nil {
		logger.Error(ctx, "Failed to delete transaction", "error", err.Error())
		return "❌ 刪除失敗，請稍後再試。"
	}

	logger.Info(ctx, "Transaction deleted successfully",
		"transaction_id", transactionID,
		"category", category,
		"amount", amount)
	return fmt.Sprintf("🗑️ 已刪除 %s $%d 的紀錄。", category, amount)
}

// handleMonthlySummary handles the command for monthly summary
func handleMonthlySummary(ctx context.Context, userID string, tokens []string) string {
	ctx, span := logger.StartSpan(ctx, "handleMonthlySummary")
	defer span.End()

	var targetMonth time.Time
	var monthSpec string

	if len(tokens) == 3 {
		// Try to parse format: "結算 2025年 5月"
		yearStr := strings.TrimSuffix(tokens[1], "年")
		monthStr := strings.TrimSuffix(tokens[2], "月")
		monthSpec = yearStr + "年" + monthStr + "月"

		logger.Info(ctx, "Specified month summary", "year", yearStr, "month", monthStr)
		year, yErr := strconv.Atoi(yearStr)
		month, mErr := strconv.Atoi(monthStr)

		if yErr != nil || mErr != nil || month < 1 || month > 12 {
			logger.Warn(ctx, "Summary format error", "year", yearStr, "month", monthStr)
			return "⚠️ 結算格式錯誤，請使用：結算 或 結算 2025年 5月"
		}

		// Create the corresponding month's start time (UTC)
		targetMonth = time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	} else {
		// Default to current month
		targetMonth = time.Now().UTC()
		monthSpec = "當月"
		logger.Info(ctx, "Current month summary")
	}

	// Get monthly summary using model.GetMonthlySummary
	summary, err := model.GetMonthlySummary(ctx, userID, targetMonth)
	if err != nil {
		logger.Error(ctx, "Failed to get summary", "error", err.Error())
		return "取得報表失敗，請稍後再試。"
	}

	// Create basic report header
	result := fmt.Sprintf("📊 %d年%d月\n收入：$%d\n支出：$%d\n\n",
		targetMonth.Year(), targetMonth.Month(), summary.IncomeTotal, summary.ExpenseTotal)

	// Organize income and expense categories separately
	incomeCategories := make(map[string]int)
	expenseCategories := make(map[string]int)

	// Get category info from models
	categoriesInfo, err := model.GetCategoriesInfo(ctx, userID)
	if err != nil {
		logger.Warn(ctx, "Failed to get category info", "error", err.Error())
		// Continue, since we at least have amount data
	}

	// Group by category type
	for cat, amt := range summary.CategoryTotals {
		// Check if we have type info for this category
		if catType, ok := categoriesInfo[cat]; ok {
			if catType == "收入" {
				incomeCategories[cat] = amt
			} else {
				expenseCategories[cat] = amt
			}
		} else {
			// If no type info, judge by amount (temporary solution)
			if amt > 0 {
				incomeCategories[cat] = amt
			} else {
				expenseCategories[cat] = amt
			}
		}
	}

	// Add income section
	if len(incomeCategories) > 0 {
		result += "💰 收入明細：\n"
		for cat, amt := range incomeCategories {
			result += fmt.Sprintf("・%s：$%d\n", cat, amt)
		}
		result += "\n"
	}

	// Add expense section
	if len(expenseCategories) > 0 {
		result += "💸 支出明細：\n"
		for cat, amt := range expenseCategories {
			result += fmt.Sprintf("・%s：$%d\n", cat, amt)
		}
		result += "\n"
	}

	// Add net income
	result += fmt.Sprintf("💰 淨收益：$%d", summary.IncomeTotal-summary.ExpenseTotal)

	logger.Info(ctx, "Summary completed",
		"month_spec", monthSpec,
		"income", summary.IncomeTotal,
		"expense", summary.ExpenseTotal,
		"income_categories", len(incomeCategories),
		"expense_categories", len(expenseCategories))

	return result
}

// getHelpText returns the help text for commands
func getHelpText(ctx context.Context) string {
	ctx, span := logger.StartSpan(ctx, "getHelpText")
	defer span.End()

	logger.Info(ctx, "Show help text")

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
