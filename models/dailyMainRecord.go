package models

import "time"

type DailyMainRecord struct {
	APIID         int        `gorm:"column:api_id" json:"api_id"`
	BreakfastCal  int        `gorm:"column:breakfast_cal" json:"breakfast_cal"`
	CreatedAt     *time.Time `gorm:"column:created_at" json:"created_at"`
	DiaryDate     *time.Time `gorm:"column:diary_date" json:"diary_date"`
	DinnerCal     int        `gorm:"column:dinner_cal" json:"dinner_cal"`
	ID            int64      `gorm:"column:id;primary_key" json:"id"`
	KcalGoal      int        `gorm:"column:kcal_goal" json:"kcal_goal"`
	KcalTotal     int        `gorm:"column:kcal_total" json:"kcal_total"`
	LunchCal      int        `gorm:"column:lunch_cal" json:"lunch_cal"`
	MemberID      string     `gorm:"column:member_id" json:"member_id"`
	MoistureGoal  int        `gorm:"column:moisture_goal" json:"moisture_goal"`
	MoistureTotal int        `gorm:"column:moisture_total" json:"moisture_total"`
	OtherCal      int        `gorm:"column:other_cal" json:"other_cal"`
	UpdatedAt     *time.Time `gorm:"column:updated_at" json:"updated_at"`
}

// TableName sets the insert table name for this struct type
func (d *DailyMainRecord) TableName() string {
	return "daily_main_records"
}
