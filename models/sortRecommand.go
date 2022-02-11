package models

import "time"

type SortRecommend struct {
	CreatedAt        *time.Time `gorm:"column:created_at" json:"created_at"`
	ID               int64      `gorm:"column:id;primary_key" json:"id"`
	MemberID         int64      `gorm:"column:member_id" json:"member_id"`
	Recommend        float64    `gorm:"column:recommend" json:"recommend"`
	RestaurantID     int64      `gorm:"column:restaurant_id" json:"restaurant_id"`
	SuggestDishCount int        `gorm:"column:suggest_dish_count" json:"suggest_dish_count"`
	SuggestRatio     float64    `gorm:"column:suggest_ratio" json:"suggest_ratio"`
	TotalDishCount   int        `gorm:"column:total_dish_count" json:"total_dish_count"`
	UpdatedAt        *time.Time `gorm:"column:updated_at" json:"updated_at"`
}

// TableName sets the insert table name for this struct type
func (s *SortRecommend) TableName() string {
	return "sort_recommend"
}
