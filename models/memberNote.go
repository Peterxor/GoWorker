package models

import "time"

type MemberNote struct {
	Content       string     `gorm:"column:content" json:"content"`
	CreatedAt     *time.Time `gorm:"column:created_at" json:"created_at"`
	DeletedAt     *time.Time `gorm:"column:deleted_at" json:"deleted_at"`
	ID            int64      `gorm:"column:id;primary_key" json:"id"`
	MemberID      string     `gorm:"column:member_id" json:"member_id"`
	ReportComment string     `gorm:"column:report_comment" json:"report_comment"`
	ReportedBy    int        `gorm:"column:reported_by" json:"reported_by"`
	UpdatedAt     *time.Time `gorm:"column:updated_at" json:"updated_at"`
	UserID        int64      `gorm:"column:user_id" json:"user_id"`
}

// TableName sets the insert table name for this struct type
func (m *MemberNote) TableName() string {
	return "member_note"
}
