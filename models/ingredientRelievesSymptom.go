package models

type IngredientRelievesSymptom struct {
	IngredientID int64 `gorm:"column:ingredient_id" json:"ingredient_id"`
	SymptomID    int64 `gorm:"column:symptom_id" json:"symptom_id"`
}

// TableName sets the insert table name for this struct type
func (i *IngredientRelievesSymptom) TableName() string {
	return "ingredient_relieves_symptoms"
}