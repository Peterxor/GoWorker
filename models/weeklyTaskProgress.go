package models

import "time"

type WeeklyTaskProgress struct {
	ClickedBy     int        `gorm:"column:clicked_by" json:"clicked_by"`
	CommentChange string     `gorm:"column:comment_change" json:"comment_change"`
	CreatedAt     *time.Time `gorm:"column:created_at" json:"created_at"`
	HealthUpdate  int        `gorm:"column:health_update" json:"health_update"`
	ID            int64      `gorm:"column:id;primary_key" json:"id"`
	MealUpdate    int        `gorm:"column:meal_update" json:"meal_update"`
	MemberID      string     `gorm:"column:member_id" json:"member_id"`
	Status        int        `gorm:"column:status" json:"status"`
	UpdatedAt     *time.Time `gorm:"column:updated_at" json:"updated_at"`
	Week          string     `gorm:"column:week" json:"week"`
	Year          string     `gorm:"column:year" json:"year"`
}

// TableName sets the insert table name for this struct type
func (w *WeeklyTaskProgress) TableName() string {
	return "weekly_task_progress"
}
