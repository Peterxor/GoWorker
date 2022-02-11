package models

import "time"

type MemberReport struct {
	CreatedAt *time.Time `gorm:"column:created_at" json:"created_at"`
	Data      string     `gorm:"column:data" json:"data"`
	EndDate   time.Time  `gorm:"column:end_date" json:"end_date"`
	ID        int64      `gorm:"column:id;primary_key" json:"id"`
	MemberID  string     `gorm:"column:member_id" json:"member_id"`
	StartDate time.Time  `gorm:"column:start_date" json:"start_date"`
	UpdatedAt *time.Time `gorm:"column:updated_at" json:"updated_at"`
	Week      string     `gorm:"column:week" json:"week"`
	Year      string     `gorm:"column:year" json:"year"`
}

// TableName sets the insert table name for this struct type
func (m *MemberReport) TableName() string {
	return "member_reports"
}
