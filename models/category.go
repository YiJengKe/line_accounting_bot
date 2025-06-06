package models

import (
	"accountingbot/db"
	"accountingbot/logger"
	"context"
)

type Category struct {
	ID     int    `json:"id"`
	UserID string `json:"user_id"`
	Name   string `json:"name"`
	Type   string `json:"type"`
}

// AddCategory 新增類別
func AddCategory(ctx context.Context, userID, name, typeName string) error {
	ctx, span := logger.StartSpan(ctx, "models.AddCategory")
	defer span.End()

	logger.Info(ctx, "新增類別", "user_id", userID, "name", name, "type", typeName)

	_, err := db.ExecContext(ctx, `
        INSERT INTO categories (user_id, name, type) VALUES ($1, $2, $3)
    `, userID, name, typeName)

	if err != nil {
		logger.Error(ctx, "新增類別失敗", "error", err.Error())
		return err
	}

	logger.Info(ctx, "類別新增成功", "name", name, "type", typeName)
	return nil
}

// UpdateCategory 修改類別
func UpdateCategory(ctx context.Context, userID, oldName, newName string) (bool, error) {
	ctx, span := logger.StartSpan(ctx, "models.UpdateCategory")
	defer span.End()

	logger.Info(ctx, "修改類別", "user_id", userID, "old_name", oldName, "new_name", newName)

	result, err := db.ExecContext(ctx, `
        UPDATE categories SET name = $1 WHERE user_id = $2 AND name = $3
    `, newName, userID, oldName)

	if err != nil {
		logger.Error(ctx, "修改類別失敗", "error", err.Error())
		return false, err
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		logger.Warn(ctx, "找不到要修改的類別", "name", oldName)
		return false, nil
	}

	logger.Info(ctx, "類別修改成功", "old_name", oldName, "new_name", newName)
	return true, nil
}

// DeleteCategory 刪除類別
func DeleteCategory(ctx context.Context, userID, name string) (bool, error) {
	ctx, span := logger.StartSpan(ctx, "models.DeleteCategory")
	defer span.End()

	logger.Info(ctx, "刪除類別", "user_id", userID, "name", name)

	result, err := db.ExecContext(ctx, `DELETE FROM categories WHERE user_id = $1 AND name = $2`, userID, name)
	if err != nil {
		logger.Error(ctx, "刪除類別失敗", "error", err.Error())
		return false, err
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		logger.Warn(ctx, "找不到要刪除的類別", "name", name)
		return false, nil
	}

	logger.Info(ctx, "類別刪除成功", "name", name)
	return true, nil
}

// CheckCategoryExists 檢查類別是否已存在
func CheckCategoryExists(ctx context.Context, userID, name, typeName string) (bool, error) {
	ctx, span := logger.StartSpan(ctx, "models.CheckCategoryExists")
	defer span.End()

	logger.Info(ctx, "檢查類別是否存在", "user_id", userID, "name", name, "type", typeName)

	var exists bool
	err := db.QueryRowContext(ctx, `
        SELECT EXISTS (
            SELECT 1 FROM categories WHERE user_id = $1 AND name = $2 AND type = $3
        )
    `, userID, name, typeName).Scan(&exists)

	if err != nil {
		logger.Error(ctx, "檢查類別失敗", "error", err.Error())
		return false, err
	}

	return exists, nil
}

// GetCategoriesByType 按類型取得類別
func GetCategoriesByType(ctx context.Context, userID string) (map[string][]string, error) {
	ctx, span := logger.StartSpan(ctx, "models.GetCategoriesByType")
	defer span.End()

	logger.Info(ctx, "取得分類類別", "user_id", userID)

	rows, err := db.QueryContext(ctx, `
        SELECT type, name FROM categories WHERE user_id = $1 ORDER BY type, name
    `, userID)
	if err != nil {
		logger.Error(ctx, "查詢類別失敗", "error", err.Error())
		return nil, err
	}
	defer rows.Close()

	categoriesByType := make(map[string][]string)

	for rows.Next() {
		var typeName, name string
		if err := rows.Scan(&typeName, &name); err != nil {
			logger.Error(ctx, "解析類別資料失敗", "error", err.Error())
			return nil, err
		}

		categoriesByType[typeName] = append(categoriesByType[typeName], name)
	}

	logger.Info(ctx, "取得類別完成", "types_count", len(categoriesByType))
	return categoriesByType, nil
}

// GetCategoryIdAndType 取得類別 ID 和類型
func GetCategoryIdAndType(ctx context.Context, userID, name string) (int, string, error) {
	ctx, span := logger.StartSpan(ctx, "models.GetCategoryIdAndType")
	defer span.End()

	logger.Info(ctx, "取得類別 ID 和類型", "user_id", userID, "name", name)

	var id int
	var typeName string

	err := db.QueryRowContext(ctx, `
        SELECT id, type FROM categories WHERE user_id = $1 AND name = $2
    `, userID, name).Scan(&id, &typeName)

	if err != nil {
		logger.Warn(ctx, "類別不存在", "name", name, "error", err.Error())
		return 0, "", err
	}

	logger.Info(ctx, "取得類別成功", "id", id, "type", typeName)
	return id, typeName, nil
}

// GetCategoriesInfo 獲取用戶的所有類別資訊，返回 map[類別名稱]類型
func GetCategoriesInfo(ctx context.Context, userID string) (map[string]string, error) {
	ctx, span := logger.StartSpan(ctx, "models.GetCategoriesInfo")
	defer span.End()

	logger.Info(ctx, "獲取類別資訊", "user_id", userID)

	rows, err := db.QueryContext(ctx, `
        SELECT name, type FROM categories WHERE user_id = $1
    `, userID)
	if err != nil {
		logger.Error(ctx, "獲取類別資訊失敗", "error", err.Error())
		return nil, err
	}
	defer rows.Close()

	categoriesInfo := make(map[string]string)

	for rows.Next() {
		var name, typeName string
		if err := rows.Scan(&name, &typeName); err != nil {
			logger.Error(ctx, "解析類別資訊失敗", "error", err.Error())
			return nil, err
		}
		categoriesInfo[name] = typeName
	}

	logger.Info(ctx, "類別資訊獲取成功", "count", len(categoriesInfo))
	return categoriesInfo, nil
}
