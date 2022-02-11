package models

import "time"

type DailyMealDish struct {
	MemberId          string     `json:"member_id" gorm:"column:member_id"`
	MainId            int64      `json:"main_id" gorm:"column:main_id"`
	MealId            int64      `json:"meal_id" gorm:"column:meal_id"`
	MealType          string     `json:"meal_type" gorm:"column:meal_type"`
	DishId            int64      `json:"dish_id" gorm:"column:dish_id"`
	Source            int        `json:"source" gorm:"column:source"`
	AvatarPath        string     `json:"avatar_path" gorm:"column:avatar_path"`
	Quantity          int        `json:"quantity" gorm:"column:quantity"`
	FixedKcal         int        `json:"fixed_kcal" gorm:"column:fixed_kcal"`
	CrudeProtein      float64    `json:"crude_protein" gorm:"column:crude_protein"`
	CrudeFat          float64    `json:"crude_fat" gorm:"column:crude_fat"`
	TotalCarbohydrate float64    `json:"total_carbohydrate" gorm:"column:total_carbohydrate"`
	DietaryFiber      float64    `json:"dietary_fiber" gorm:"column:dietary_fiber"`
	Analyzed          int        `json:"analyzed" gorm:"column:analyzed"`
	AnalyzedJson      string     `json:"analyzed_json" gorm:"column:analyzed_json"`
	MealTime          string     `json:"meal_time" gorm:"column:meal_time"`
	Name              string     `json:"name" gorm:"column:name"`
	ApiId             int64      `json:"api_id" gorm:"column:api_id"`
	AnalyzeTarget     string     `json:"analyze_target" gorm:"column:analyze_target"`
	UnitQuantities    float64    `json:"unit_quantities" gorm:"column:unit_quantities"`
	RestaurantId      int64      `json:"restaurant_id" gorm:"column:restaurant_id"`
	RestaurantName    string     `json:"restaurant_name" gorm:"column:restaurant_name"`
	Rating            float64    `json:"rating" gorm:"column:rating"`
	CreatedAt         *time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt         *time.Time `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt         *time.Time `gorm:"column:deleted_at" json:"deleted_at"`
}

// TableName sets the insert table name for this struct type
func (d *DailyMealDish) TableName() string {
	return "daily_meal_dish"
}
