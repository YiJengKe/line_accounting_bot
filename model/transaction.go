package model

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

// GetMonthlySummary now accepts a context parameter
func GetMonthlySummary(ctx context.Context, userID string, month time.Time) (Summary, error) {
	ctx, span := logger.StartSpan(ctx, "models.GetMonthlySummary")
	defer span.End()

	logger.Info(ctx, "Get monthly summary report",
		"user_id", userID,
		"year", month.Year(),
		"month", month.Month())

	start := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	rows, err := db.QueryContext(ctx, `
        SELECT t.type, c.name, SUM(t.amount)
        FROM transactions t
        JOIN categories c ON t.category_id = c.id
        WHERE t.user_id = $1 AND t.created_at >= $2 AND t.created_at < $3
        GROUP BY t.type, c.name
    `, userID, start, end)

	if err != nil {
		logger.Error(ctx, "Failed to query monthly summary", "error", err.Error())
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
			logger.Error(ctx, "Failed to parse monthly summary data", "error", err.Error())
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

	logger.Info(ctx, "Monthly summary generated",
		"income_total", summary.IncomeTotal,
		"expense_total", summary.ExpenseTotal,
		"categories_count", categories)

	return summary, nil
}

// AddTransaction adds a new transaction record
func AddTransaction(ctx context.Context, userID string, categoryID int, transType string, amount int) (*Transaction, error) {
	ctx, span := logger.StartSpan(ctx, "models.AddTransaction")
	defer span.End()

	logger.Info(ctx, "Add transaction record",
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
		logger.Error(ctx, "Failed to add transaction record", "error", err.Error())
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		logger.Warn(ctx, "Cannot get new transaction ID", "error", err.Error())
	} else {
		transaction.ID = int(id)
	}

	logger.Info(ctx, "Transaction record added successfully", "transaction_id", transaction.ID)
	return transaction, nil
}

// GetTransactions gets user's transaction records
func GetTransactions(ctx context.Context, userID string, limit int) ([]*Transaction, error) {
	ctx, span := logger.StartSpan(ctx, "models.GetTransactions")
	defer span.End()

	logger.Info(ctx, "Query user transactions", "user_id", userID, "limit", limit)

	rows, err := db.QueryContext(ctx, `
        SELECT id, user_id, type, amount, category_id, created_at
        FROM transactions 
        WHERE user_id = $1
        ORDER BY created_at DESC
        LIMIT $2
    `, userID, limit)

	if err != nil {
		logger.Error(ctx, "Failed to query transactions", "error", err.Error())
		return nil, err
	}
	defer rows.Close()

	var transactions []*Transaction

	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.UserID, &t.Type, &t.Amount, &t.CategoryID, &t.CreatedAt); err != nil {
			logger.Error(ctx, "Failed to parse transaction record", "error", err.Error())
			return nil, err
		}
		transactions = append(transactions, &t)
	}

	logger.Info(ctx, "Transaction query completed", "count", len(transactions))
	return transactions, nil
}

// UpdateTransaction updates a transaction record
func UpdateTransaction(ctx context.Context, id int, amount int) error {
	ctx, span := logger.StartSpan(ctx, "models.UpdateTransaction")
	defer span.End()

	logger.Info(ctx, "Update transaction record", "id", id, "new_amount", amount)

	result, err := db.ExecContext(ctx, `UPDATE transactions SET amount = $1 WHERE id = $2`, amount, id)
	if err != nil {
		logger.Error(ctx, "Failed to update transaction record", "error", err.Error())
		return err
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		logger.Warn(ctx, "No transaction record found to update", "id", id)
	} else {
		logger.Info(ctx, "Transaction record updated successfully", "id", id)
	}

	return nil
}

// DeleteTransaction deletes a transaction record
func DeleteTransaction(ctx context.Context, id int) error {
	ctx, span := logger.StartSpan(ctx, "models.DeleteTransaction")
	defer span.End()

	logger.Info(ctx, "Delete transaction record", "id", id)

	result, err := db.ExecContext(ctx, `DELETE FROM transactions WHERE id = $1`, id)
	if err != nil {
		logger.Error(ctx, "Failed to delete transaction record", "error", err.Error())
		return err
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		logger.Warn(ctx, "No transaction record found to delete", "id", id)
	} else {
		logger.Info(ctx, "Transaction record deleted successfully", "id", id)
	}

	return nil
}

// FindTransactionID finds a transaction record by user ID, category name, and amount
func FindTransactionID(ctx context.Context, userID, categoryName string, amount int) (int, error) {
	ctx, span := logger.StartSpan(ctx, "models.FindTransactionID")
	defer span.End()

	logger.Info(ctx, "Query transaction record ID",
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
		logger.Warn(ctx, "No matching transaction record found",
			"category", categoryName,
			"amount", amount,
			"error", err.Error())
		return 0, err
	}

	logger.Info(ctx, "Transaction record found", "transaction_id", transactionID)
	return transactionID, nil
}
