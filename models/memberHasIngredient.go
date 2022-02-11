package models

type MemberHasIngredient struct {
	IngredientID  int64  `gorm:"column:ingredient_id" json:"ingredient_id"`
	MemberID      int64 `gorm:"column:member_id" json:"member_id"`
	Status        string `gorm:"column:status" json:"status"`
	Type          string `gorm:"column:type" json:"type"`
	OperationType string `gorm:"column:operation_type" json:"operation_type"`
	Version       int64  `gorm:"column:version" json:"version"`
}

// TableName sets the insert table name for this struct type
func (m *MemberHasIngredient) TableName() string {
	return "member_has_ingredients"
}
