package model

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

// AddCategory adds a new category
func AddCategory(ctx context.Context, userID, name, typeName string) error {
	ctx, span := logger.StartSpan(ctx, "models.AddCategory")
	defer span.End()

	logger.Info(ctx, "Add category", "user_id", userID, "name", name, "type", typeName)

	_, err := db.ExecContext(ctx, `
        INSERT INTO categories (user_id, name, type) VALUES ($1, $2, $3)
    `, userID, name, typeName)

	if err != nil {
		logger.Error(ctx, "Failed to add category", "error", err.Error())
		return err
	}

	logger.Info(ctx, "Category added successfully", "name", name, "type", typeName)
	return nil
}

// UpdateCategory updates a category
func UpdateCategory(ctx context.Context, userID, oldName, newName string) (bool, error) {
	ctx, span := logger.StartSpan(ctx, "models.UpdateCategory")
	defer span.End()

	logger.Info(ctx, "Update category", "user_id", userID, "old_name", oldName, "new_name", newName)

	result, err := db.ExecContext(ctx, `
        UPDATE categories SET name = $1 WHERE user_id = $2 AND name = $3
    `, newName, userID, oldName)

	if err != nil {
		logger.Error(ctx, "Failed to update category", "error", err.Error())
		return false, err
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		logger.Warn(ctx, "Category to update not found", "name", oldName)
		return false, nil
	}

	logger.Info(ctx, "Category updated successfully", "old_name", oldName, "new_name", newName)
	return true, nil
}

// DeleteCategory deletes a category
func DeleteCategory(ctx context.Context, userID, name string) (bool, error) {
	ctx, span := logger.StartSpan(ctx, "models.DeleteCategory")
	defer span.End()

	logger.Info(ctx, "Delete category", "user_id", userID, "name", name)

	result, err := db.ExecContext(ctx, `DELETE FROM categories WHERE user_id = $1 AND name = $2`, userID, name)
	if err != nil {
		logger.Error(ctx, "Failed to delete category", "error", err.Error())
		return false, err
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		logger.Warn(ctx, "Category to delete not found", "name", name)
		return false, nil
	}

	logger.Info(ctx, "Category deleted successfully", "name", name)
	return true, nil
}

// CheckCategoryExists checks if a category already exists
func CheckCategoryExists(ctx context.Context, userID, name, typeName string) (bool, error) {
	ctx, span := logger.StartSpan(ctx, "models.CheckCategoryExists")
	defer span.End()

	logger.Info(ctx, "Check if category exists", "user_id", userID, "name", name, "type", typeName)

	var exists bool
	err := db.QueryRowContext(ctx, `
        SELECT EXISTS (
            SELECT 1 FROM categories WHERE user_id = $1 AND name = $2 AND type = $3
        )
    `, userID, name, typeName).Scan(&exists)

	if err != nil {
		logger.Error(ctx, "Failed to check category", "error", err.Error())
		return false, err
	}

	return exists, nil
}

// GetCategoriesByType gets categories by type
func GetCategoriesByType(ctx context.Context, userID string) (map[string][]string, error) {
	ctx, span := logger.StartSpan(ctx, "models.GetCategoriesByType")
	defer span.End()

	logger.Info(ctx, "Get categories by type", "user_id", userID)

	rows, err := db.QueryContext(ctx, `
        SELECT type, name FROM categories WHERE user_id = $1 ORDER BY type, name
    `, userID)
	if err != nil {
		logger.Error(ctx, "Failed to query categories", "error", err.Error())
		return nil, err
	}
	defer rows.Close()

	categoriesByType := make(map[string][]string)

	for rows.Next() {
		var typeName, name string
		if err := rows.Scan(&typeName, &name); err != nil {
			logger.Error(ctx, "Failed to parse category data", "error", err.Error())
			return nil, err
		}

		categoriesByType[typeName] = append(categoriesByType[typeName], name)
	}

	logger.Info(ctx, "Categories fetched", "types_count", len(categoriesByType))
	return categoriesByType, nil
}

// GetCategoryIdAndType gets the category ID and type
func GetCategoryIdAndType(ctx context.Context, userID, name string) (int, string, error) {
	ctx, span := logger.StartSpan(ctx, "models.GetCategoryIdAndType")
	defer span.End()

	logger.Info(ctx, "Get category ID and type", "user_id", userID, "name", name)

	var id int
	var typeName string

	err := db.QueryRowContext(ctx, `
        SELECT id, type FROM categories WHERE user_id = $1 AND name = $2
    `, userID, name).Scan(&id, &typeName)

	if err != nil {
		logger.Warn(ctx, "Category does not exist", "name", name, "error", err.Error())
		return 0, "", err
	}

	logger.Info(ctx, "Category fetched", "id", id, "type", typeName)
	return id, typeName, nil
}

// GetCategoriesInfo gets all category info for a user, returns map[category_name]type
func GetCategoriesInfo(ctx context.Context, userID string) (map[string]string, error) {
	ctx, span := logger.StartSpan(ctx, "models.GetCategoriesInfo")
	defer span.End()

	logger.Info(ctx, "Get categories info", "user_id", userID)

	rows, err := db.QueryContext(ctx, `
        SELECT name, type FROM categories WHERE user_id = $1
    `, userID)
	if err != nil {
		logger.Error(ctx, "Failed to get categories info", "error", err.Error())
		return nil, err
	}
	defer rows.Close()

	categoriesInfo := make(map[string]string)

	for rows.Next() {
		var name, typeName string
		if err := rows.Scan(&name, &typeName); err != nil {
			logger.Error(ctx, "Failed to parse category info", "error", err.Error())
			return nil, err
		}
		categoriesInfo[name] = typeName
	}

	logger.Info(ctx, "Categories info fetched", "count", len(categoriesInfo))
	return categoriesInfo, nil
}
