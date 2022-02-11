package models

type DishHasRestaurant struct {
	DishID       int64 `gorm:"column:dish_id" json:"dish_id"`
	ID           int64 `gorm:"column:id;primary_key" json:"id"`
	RestaurantID int64 `gorm:"column:restaurant_id" json:"restaurant_id"`
}

// TableName sets the insert table name for this struct type
func (d *DishHasRestaurant) TableName() string {
	return "dish_has_restaurants"
}
