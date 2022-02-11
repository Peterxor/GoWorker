package models

import "time"

type User struct {
	CreatedAt       *time.Time `gorm:"column:created_at" json:"created_at"`
	CreatedBy       string     `gorm:"column:created_by" json:"created_by"`
	Deletable       int        `gorm:"column:deletable" json:"deletable"`
	DeletedAt       *time.Time `gorm:"column:deleted_at" json:"deleted_at"`
	Email           string     `gorm:"column:email" json:"email"`
	EmailVerifiedAt *time.Time `gorm:"column:email_verified_at" json:"email_verified_at"`
	ID              int      `gorm:"column:id;primary_key" json:"id"`
	Name            string     `gorm:"column:name" json:"name"`
	Password        string     `gorm:"column:password" json:"password"`
	RememberToken   string     `gorm:"column:remember_token" json:"remember_token"`
	Status          int        `gorm:"column:status" json:"status"`
	UpdatedAt       *time.Time `gorm:"column:updated_at" json:"updated_at"`
}

// TableName sets the insert table name for this struct type
func (u *User) TableName() string {
	return "users"
}
