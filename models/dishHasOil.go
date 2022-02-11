package models

type DishHasOil struct {
	DishID int64 `gorm:"column:dish_id" json:"dish_id"`
	OilID  int64 `gorm:"column:oil_id" json:"oil_id"`
}

// TableName sets the insert table name for this struct type
func (d *DishHasOil) TableName() string {
	return "dish_has_oil"
}