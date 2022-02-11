package models

type DishHasCookMethod struct {
	CookMethodID     int64  `gorm:"column:cook_method_id" json:"cook_method_id"`
	DishID           int64  `gorm:"column:dish_id" json:"dish_id"`
	OtherDescription string `gorm:"column:other_description" json:"other_description"`
}

// TableName sets the insert table name for this struct type
func (d *DishHasCookMethod) TableName() string {
	return "dish_has_cook_method"
}
