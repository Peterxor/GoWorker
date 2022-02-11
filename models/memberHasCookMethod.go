package models

type MemberHasCookMethod struct {
	CookMethodID int64  `gorm:"column:cook_method_id" json:"cook_method_id"`
	MemberID     string `gorm:"column:member_id" json:"member_id"`
}

// TableName sets the insert table name for this struct type
func (m *MemberHasCookMethod) TableName() string {
	return "member_has_cook_method"
}
