package models

type MemberHasDish struct {
	DishID       int64  `gorm:"column:dish_id" json:"dish_id"`
	MemberID     int64  `gorm:"column:member_id" json:"member_id"`
	RestaurantID int64  `gorm:"column:restaurant_id" json:"restaurant_id"`
	Type         string `gorm:"column:type" json:"type"`
	Version      int64  `gorm:"column:version" json:"version"`
	Status       string `gorm:"column:status" json:"status"`
	Points       int    `gorm:"column:points" json:"points"`
}

// TableName sets the insert table name for this struct type
func (m *MemberHasDish) TableName() string {
	return "member_has_dish"
}
