package models

type Category struct {
	ID     int
	UserID string
	Name   string
	Type   string // "收入" 或 "支出"
}
