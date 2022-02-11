package models

import "time"

type ActivityLog struct {
	CauserID    int        `gorm:"column:causer_id" json:"causer_id"`
	CauserType  string     `gorm:"column:causer_type" json:"causer_type"`
	CreatedAt   *time.Time `gorm:"column:created_at" json:"created_at"`
	Description string     `gorm:"column:description" json:"description"`
	ID          int64      `gorm:"column:id;primary_key" json:"id"`
	LogName     string     `gorm:"column:log_name" json:"log_name"`
	Properties  string     `gorm:"column:properties" json:"properties"`
	SubjectID   int        `gorm:"column:subject_id" json:"subject_id"`
	SubjectType string     `gorm:"column:subject_type" json:"subject_type"`
	UpdatedAt   *time.Time `gorm:"column:updated_at" json:"updated_at"`
}

// TableName sets the insert table name for this struct type
func (a *ActivityLog) TableName() string {
	return "activity_log"
}
