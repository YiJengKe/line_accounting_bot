package db

import (
	"context"
	"database/sql"
	"time"

	"accountingbot/config"
	"accountingbot/logger"

	_ "github.com/lib/pq"
)

var DB *sql.DB

// Init 初始化資料庫連線
func Init(ctx context.Context) {
	// 建立追蹤 span
	ctx, span := logger.StartSpan(ctx, "db.Init")
	defer span.End()

	// 取得資料庫連線設定
	cfg := config.Get()
	logger.Info(ctx, "正在連線到資料庫", "db_url_masked", maskDatabaseURL(cfg.Db.PsqlUrl))

	// 建立資料庫連線
	var err error
	DB, err = sql.Open("postgres", cfg.Db.PsqlUrl)
	if err != nil {
		logger.Fatal(ctx, "無法建立資料庫連線", "error", err.Error())
	}

	// 設定連線池
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(5)
	DB.SetConnMaxLifetime(5 * time.Minute)

	// 嘗試連線
	retries := 5
	for i := 0; i < retries; i++ {
		err = DB.PingContext(ctx)
		if err == nil {
			break
		}

		logger.Warn(ctx, "資料庫連線測試失敗，稍後重試",
			"attempt", i+1,
			"max_attempts", retries,
			"error", err.Error(),
		)

		// 等待後重試
		time.Sleep(3 * time.Second)
	}

	// 檢查最終連線結果
	if err != nil {
		logger.Fatal(ctx, "無法連線至資料庫", "error", err.Error())
	}

	logger.Info(ctx, "資料庫連線成功")
	createTables(ctx)
}

// createTables 建立必要的資料表
func createTables(ctx context.Context) {
	ctx, span := logger.StartSpan(ctx, "db.createTables")
	defer span.End()

	logger.Info(ctx, "正在檢查並建立資料表")

	// 建立資料表的 SQL 語句
	query := `
        CREATE TABLE IF NOT EXISTS categories (
            id SERIAL PRIMARY KEY,
            user_id TEXT NOT NULL,
            name TEXT NOT NULL,
            type TEXT NOT NULL,
            UNIQUE(user_id, name)
        );

        CREATE TABLE IF NOT EXISTS transactions (
            id SERIAL PRIMARY KEY,
            user_id TEXT NOT NULL,
            type TEXT NOT NULL,
            amount INTEGER NOT NULL,
            category_id INTEGER NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );
    `

	// 執行 SQL
	_, err := DB.ExecContext(ctx, query)
	if err != nil {
		logger.Fatal(ctx, "建立資料表失敗", "error", err.Error())
	}

	logger.Info(ctx, "資料表檢查/建立完成")
}

// 包裝資料庫操作函數，自動加入追蹤

// QueryContext 執行查詢並返回 rows
func QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	ctx, span := logger.StartSpan(ctx, "db.query")
	defer span.End()

	rows, err := DB.QueryContext(ctx, query, args...)
	if err != nil {
		logger.Error(ctx, "查詢失敗", "query", query, "error", err.Error())
	}
	return rows, err
}

// ExecContext 執行命令並返回結果
func ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	ctx, span := logger.StartSpan(ctx, "db.exec")
	defer span.End()

	result, err := DB.ExecContext(ctx, query, args...)
	if err != nil {
		logger.Error(ctx, "執行失敗", "query", query, "error", err.Error())
	}
	return result, err
}

// QueryRowContext 執行查詢並返回單一行
func QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	ctx, span := logger.StartSpan(ctx, "db.queryRow")
	defer span.End()

	return DB.QueryRowContext(ctx, query, args...)
}

// maskDatabaseURL 遮蔽資料庫連接字串中的敏感資訊
func maskDatabaseURL(url string) string {
	if len(url) < 20 {
		return "[遮蔽]"
	}
	return url[:10] + "..." + url[len(url)-10:]
}
