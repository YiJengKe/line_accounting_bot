package models

import (
	"accountingbot/db"
	"accountingbot/logger"
	"context"
	"time"
)

type Transaction struct {
	ID         int       `json:"id" gorm:"column:id;primaryKey"`
	UserID     string    `json:"user_id" gorm:"column:user_id"`
	Type       string    `json:"type" gorm:"column:type"`
	Amount     int       `json:"amount" gorm:"column:amount"`
	CategoryID int       `json:"category_id" gorm:"column:category_id"`
	CreatedAt  time.Time `json:"created_at" gorm:"column:created_at;default:CURRENT_TIMESTAMP"`
}

type Summary struct {
	IncomeTotal    int
	ExpenseTotal   int
	CategoryTotals map[string]int
}

// GetMonthlySummary 現在接受 context 參數
func GetMonthlySummary(ctx context.Context, userID string, month time.Time) (Summary, error) {
	// 建立追蹤 span
	ctx, span := logger.StartSpan(ctx, "models.GetMonthlySummary")
	defer span.End()

	logger.Info(ctx, "獲取月結報表",
		"user_id", userID,
		"year", month.Year(),
		"month", month.Month())

	start := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	// 使用包含 trace context 的查詢
	rows, err := db.QueryContext(ctx, `
        SELECT t.type, c.name, SUM(t.amount)
        FROM transactions t
        JOIN categories c ON t.category_id = c.id
        WHERE t.user_id = $1 AND t.created_at >= $2 AND t.created_at < $3
        GROUP BY t.type, c.name
    `, userID, start, end)

	if err != nil {
		logger.Error(ctx, "月結報表查詢失敗", "error", err.Error())
		return Summary{}, err
	}
	defer rows.Close()

	summary := Summary{
		CategoryTotals: make(map[string]int),
	}

	var categories int
	for rows.Next() {
		var ttype, categoryName string
		var total int
		if err := rows.Scan(&ttype, &categoryName, &total); err != nil {
			logger.Error(ctx, "月結報表資料解析失敗", "error", err.Error())
			return summary, err
		}

		summary.CategoryTotals[categoryName] = total
		if ttype == "收入" {
			summary.IncomeTotal += total
		} else {
			summary.ExpenseTotal += total
		}
		categories++
	}

	logger.Info(ctx, "月結報表生成完成",
		"income_total", summary.IncomeTotal,
		"expense_total", summary.ExpenseTotal,
		"categories_count", categories)

	return summary, nil
}

// AddTransaction 新增交易紀錄
func AddTransaction(ctx context.Context, userID string, categoryID int, transType string, amount int) (*Transaction, error) {
	ctx, span := logger.StartSpan(ctx, "models.AddTransaction")
	defer span.End()

	logger.Info(ctx, "新增交易紀錄",
		"user_id", userID,
		"category_id", categoryID,
		"type", transType,
		"amount", amount)

	transaction := &Transaction{
		UserID:     userID,
		CategoryID: categoryID,
		Type:       transType,
		Amount:     amount,
		CreatedAt:  time.Now(),
	}

	result, err := db.ExecContext(ctx, `
        INSERT INTO transactions (user_id, category_id, type, amount, created_at)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `, transaction.UserID, transaction.CategoryID, transaction.Type, transaction.Amount, transaction.CreatedAt)

	if err != nil {
		logger.Error(ctx, "新增交易紀錄失敗", "error", err.Error())
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		// PostgreSQL 可能不支援 LastInsertId，可以用另一種方式取得 ID
		logger.Warn(ctx, "無法取得新增交易 ID", "error", err.Error())
	} else {
		transaction.ID = int(id)
	}

	logger.Info(ctx, "交易紀錄新增成功", "transaction_id", transaction.ID)
	return transaction, nil
}

// GetTransactions 獲取用戶的交易紀錄
func GetTransactions(ctx context.Context, userID string, limit int) ([]*Transaction, error) {
	ctx, span := logger.StartSpan(ctx, "models.GetTransactions")
	defer span.End()

	logger.Info(ctx, "查詢用戶交易紀錄", "user_id", userID, "limit", limit)

	rows, err := db.QueryContext(ctx, `
        SELECT id, user_id, type, amount, category_id, created_at
        FROM transactions 
        WHERE user_id = $1
        ORDER BY created_at DESC
        LIMIT $2
    `, userID, limit)

	if err != nil {
		logger.Error(ctx, "查詢交易紀錄失敗", "error", err.Error())
		return nil, err
	}
	defer rows.Close()

	var transactions []*Transaction

	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.UserID, &t.Type, &t.Amount, &t.CategoryID, &t.CreatedAt); err != nil {
			logger.Error(ctx, "解析交易紀錄失敗", "error", err.Error())
			return nil, err
		}
		transactions = append(transactions, &t)
	}

	logger.Info(ctx, "交易紀錄查詢完成", "count", len(transactions))
	return transactions, nil
}

// UpdateTransaction 更新交易紀錄
func UpdateTransaction(ctx context.Context, id int, amount int) error {
	ctx, span := logger.StartSpan(ctx, "models.UpdateTransaction")
	defer span.End()

	logger.Info(ctx, "更新交易紀錄", "id", id, "new_amount", amount)

	result, err := db.ExecContext(ctx, `UPDATE transactions SET amount = $1 WHERE id = $2`, amount, id)
	if err != nil {
		logger.Error(ctx, "更新交易紀錄失敗", "error", err.Error())
		return err
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		logger.Warn(ctx, "找不到要更新的交易紀錄", "id", id)
	} else {
		logger.Info(ctx, "交易紀錄更新成功", "id", id)
	}

	return nil
}

// DeleteTransaction 刪除交易紀錄
func DeleteTransaction(ctx context.Context, id int) error {
	ctx, span := logger.StartSpan(ctx, "models.DeleteTransaction")
	defer span.End()

	logger.Info(ctx, "刪除交易紀錄", "id", id)

	result, err := db.ExecContext(ctx, `DELETE FROM transactions WHERE id = $1`, id)
	if err != nil {
		logger.Error(ctx, "刪除交易紀錄失敗", "error", err.Error())
		return err
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		logger.Warn(ctx, "找不到要刪除的交易紀錄", "id", id)
	} else {
		logger.Info(ctx, "交易紀錄刪除成功", "id", id)
	}

	return nil
}

// FindTransactionID 根據用戶ID、類別名和金額尋找交易記錄
func FindTransactionID(ctx context.Context, userID, categoryName string, amount int) (int, error) {
	ctx, span := logger.StartSpan(ctx, "models.FindTransactionID")
	defer span.End()

	logger.Info(ctx, "查詢交易記錄ID",
		"user_id", userID,
		"category", categoryName,
		"amount", amount)

	var transactionID int
	err := db.QueryRowContext(ctx, `
        SELECT t.id 
        FROM transactions t
        JOIN categories c ON t.category_id = c.id
        WHERE t.user_id = $1 AND c.name = $2 AND t.amount = $3
        LIMIT 1
    `, userID, categoryName, amount).Scan(&transactionID)

	if err != nil {
		logger.Warn(ctx, "找不到符合條件的交易記錄",
			"category", categoryName,
			"amount", amount,
			"error", err.Error())
		return 0, err
	}

	logger.Info(ctx, "找到交易記錄", "transaction_id", transactionID)
	return transactionID, nil
}
