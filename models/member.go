package models

import (
	"database/sql/driver"
	"time"
)

type status string

const (
	active status = "active"
	suspend status = "suspend"
	deleted status = "deleted"
)

func (s *status) Scan(value interface{}) error {
	*s = status(value.([]byte))
	return nil
}

func (s status) Value() (driver.Value, error) {
	return string(s), nil
}

type Member struct {
	ActivationCode     string     `gorm:"column:activation_code" json:"activation_code"`
	Activity           int        `gorm:"column:activity" json:"activity"`
	BackendID          int64      `gorm:"column:backend_id;primary_key" json:"backend_id"`
	Birthday           time.Time  `gorm:"column:birthday" json:"birthday"`
	CreatedAt          *time.Time `gorm:"column:created_at" json:"created_at"`
	DeletedAt          *time.Time `gorm:"column:deleted_at" json:"deleted_at"`
	DishJob            int        `gorm:"column:dish_job" json:"dish_job"`
	Email              string     `gorm:"column:email" json:"email"`
	ExpiredAt          *time.Time `gorm:"column:expired_at" json:"expired_at"`
	ForceDishJob       int        `gorm:"column:force_dish_job" json:"force_dish_job"`
	Gender             int        `gorm:"column:gender" json:"gender"`
	Height             string     `gorm:"column:height" json:"height"`
	ID                 string     `gorm:"column:id" json:"id"`
	IngredientJob      int        `gorm:"column:ingredient_job" json:"ingredient_job"`
	LastVisitedAt      *time.Time `gorm:"column:last_visited_at" json:"last_visited_at"`
	Name               string     `gorm:"column:name" json:"name"`
	Nickname           string     `gorm:"column:nickname" json:"nickname"`
	Phone              string     `gorm:"column:phone" json:"phone"`
	PushToken          string     `gorm:"column:push_token" json:"push_token"`
	RegistrationSource string     `gorm:"column:registration_source" json:"registration_source"`
	SuggestKcal        int        `gorm:"column:suggest_kcal" json:"suggest_kcal"`
	UpdatedAt          *time.Time `gorm:"column:updated_at" json:"updated_at"`
	Weight             string     `gorm:"column:weight" json:"weight"`
	Status			   status     `sql:"status" json:"status"`
}

// TableName sets the insert table name for this struct type
// db2struct --host localhost -d dishrank_nu2 -t daily_meal_records --package models --struct DailyMealRecord -p jtMpPWhEmK7AYZ --user root --guregu --gorm --json
func (m *Member) TableName() string {
	return "members"
}
