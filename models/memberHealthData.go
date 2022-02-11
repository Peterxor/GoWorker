package models

import "time"

type MemberHealthData struct {
	Allergens        string     `gorm:"column:allergens" json:"allergens"`
	CreatedAt        *time.Time `gorm:"column:created_at" json:"created_at"`
	Diet             string     `gorm:"column:diet" json:"diet"`
	DietingPlans     string     `gorm:"column:dieting_plans" json:"dieting_plans"`
	ID               int64      `gorm:"column:id;primary_key" json:"id"`
	Ingredients      string     `gorm:"column:ingredients" json:"ingredients"`
	MemberID         string     `gorm:"column:member_id" json:"member_id"`
	PhysicalContents string     `gorm:"column:physical_contents" json:"physical_contents"`
	Physiology       string     `gorm:"column:physiology" json:"physiology"`
	SuggestionPlans  string     `gorm:"column:suggestion_plans" json:"suggestion_plans"`
	UpdatedAt        *time.Time `gorm:"column:updated_at" json:"updated_at"`
}

// TableName sets the insert table name for this struct type
func (m *MemberHealthData) TableName() string {
	return "member_health_data"
}
