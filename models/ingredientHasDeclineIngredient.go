package models

type IngredientHasDeclineIngredient struct {
	DeclineIngredientID int64 `gorm:"column:decline_ingredient_id" json:"decline_ingredient_id"`
	IngredientID        int64 `gorm:"column:ingredient_id" json:"ingredient_id"`
}

// TableName sets the insert table name for this struct type
func (i *IngredientHasDeclineIngredient) TableName() string {
	return "ingredient_has_decline_ingredients"
}