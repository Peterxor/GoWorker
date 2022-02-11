package models

import "time"

type DishHasIngredient struct {
	CreatedAt    *time.Time `gorm:"column:created_at" json:"created_at"`
	DishID       int64      `gorm:"column:dish_id" json:"dish_id"`
	IngredientID int64      `gorm:"column:ingredient_id" json:"ingredient_id"`
	Quantity     float64    `gorm:"column:quantity" json:"quantity"`
	UpdatedAt    *time.Time `gorm:"column:updated_at" json:"updated_at"`
}

// TableName sets the insert table name for this struct type
func (d *DishHasIngredient) TableName() string {
	return "dish_has_ingredients"
}
