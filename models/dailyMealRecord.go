package models

import "time"

type DailyMealRecord struct {
	APIID             int        `gorm:"column:api_id" json:"api_id"`
	CreatedAt         *time.Time `gorm:"column:created_at" json:"created_at"`
	CrudeFat          float64    `gorm:"column:crude_fat" json:"crude_fat"`
	CrudeProtein      float64    `gorm:"column:crude_protein" json:"crude_protein"`
	DietaryFiber      float64    `gorm:"column:dietary_fiber" json:"dietary_fiber"`
	FixedKcal         int        `gorm:"column:fixed_kcal" json:"fixed_kcal"`
	ID                int64      `gorm:"column:id;primary_key" json:"id"`
	MainID            int64      `gorm:"column:main_id" json:"main_id"`
	MealType          string     `gorm:"column:meal_type" json:"meal_type"`
	MemberID          string     `gorm:"column:member_id" json:"member_id"`
	RestaurantID      int64      `gorm:"column:restaurant_id" json:"restaurant_id"`
	RestaurantModel   string     `gorm:"column:restaurant_model" json:"restaurant_model"`
	TotalCarbohydrate float64    `gorm:"column:total_carbohydrate" json:"total_carbohydrate"`
	UpdatedAt         *time.Time `gorm:"column:updated_at" json:"updated_at"`
}

// TableName sets the insert table name for this struct type
func (d *DailyMealRecord) TableName() string {
	return "daily_meal_records"
}
