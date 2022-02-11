package models

import "time"

type Dish struct {
	Analyzed            int64      `gorm:"column:analyzed" json:"analyzed"`
	AuditColumns        string     `gorm:"column:audit_columns" json:"audit_columns"`
	AuditStatus         string     `gorm:"column:audit_status" json:"audit_status"`
	AuditTarget         int64      `gorm:"column:audit_target" json:"audit_target"`
	AuditType           string     `gorm:"column:audit_type" json:"audit_type"`
	BelongsTo           int64      `gorm:"column:belongs_to" json:"belongs_to"`
	CreatedAt           *time.Time `gorm:"column:created_at" json:"created_at"`
	CreatedBy           int64      `gorm:"column:created_by" json:"created_by"`
	CrudeFat            float64    `gorm:"column:crude_fat" json:"crude_fat"`
	CrudeProtein        float64    `gorm:"column:crude_protein" json:"crude_protein"`
	CookMethodId        int64      `gorm:"column:cook_method_id" json:"cook_method_id"'`
	DeletedAt           *time.Time `gorm:"column:deleted_at" json:"deleted_at"`
	Description         string     `gorm:"column:description" json:"description"`
	DietaryFiber        float64    `gorm:"column:dietary_fiber" json:"dietary_fiber"`
	DishUnit            string     `gorm:"column:dish_unit" json:"dish_unit"`
	DrRecommand         int64      `gorm:"column:dr_recommand" json:"dr_recommand"`
	FixedKcal           int64      `gorm:"column:fixed_kcal" json:"fixed_kcal"`
	GroupID             int64      `gorm:"column:group_id" json:"group_id"`
	ID                  int64      `gorm:"column:id;primary_key" json:"id"`
	IsAppVisible        int64      `gorm:"column:is_app_visible" json:"is_app_visible"`
	Kcal                int64      `gorm:"column:kcal" json:"kcal"`
	LastUpdatedAt       *time.Time `gorm:"column:last_updated_at" json:"last_updated_at"`
	Name                string     `gorm:"column:name" json:"name"`
	Price               int64      `gorm:"column:price" json:"price"`
	PublishedAt         *time.Time `gorm:"column:published_at" json:"published_at"`
	RestaurantRecommand int64      `gorm:"column:restaurant_recommand" json:"restaurant_recommand"`
	SaleStatus          string     `gorm:"column:sale_status" json:"sale_status"`
	TotalCarbohydrate   float64    `gorm:"column:total_carbohydrate" json:"total_carbohydrate"`
	UpdatedAt           *time.Time `gorm:"column:updated_at" json:"updated_at"`
	UpdatedBy           int64      `gorm:"column:updated_by" json:"updated_by"`
	WeightTotal         float64    `gorm:"column:weight_total" json:"weight_total"`
	WeightUnit          string     `gorm:"column:weight_unit" json:"weight_unit"`
}

// TableName sets the insert table name for this struct type
func (d *Dish) TableName() string {
	return "dishes"
}
