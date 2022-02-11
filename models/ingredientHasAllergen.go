package models

type IngredientHasAllergen struct {
	AllergenID   int64 `gorm:"column:allergen_id" json:"allergen_id"`
	IngredientID int64 `gorm:"column:ingredient_id" json:"ingredient_id"`
}

// TableName sets the insert table name for this struct type
func (i *IngredientHasAllergen) TableName() string {
	return "ingredient_has_allergens"
}