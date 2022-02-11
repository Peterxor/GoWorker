package models

import "time"

type Restaurant struct {
	Address          string     `gorm:"column:address" json:"address"`
	Announcement     string     `gorm:"column:announcement" json:"announcement"`
	Area             string     `gorm:"column:area" json:"area"`
	AuditColumns     string     `gorm:"column:audit_columns" json:"audit_columns"`
	AuditStatus      string     `gorm:"column:audit_status" json:"audit_status"`
	AuditTarget      int        `gorm:"column:audit_target" json:"audit_target"`
	AuditType        string     `gorm:"column:audit_type" json:"audit_type"`
	BelongsTo        int        `gorm:"column:belongs_to" json:"belongs_to"`
	BusinessStatus   int        `gorm:"column:business_status" json:"business_status"`
	Category         string     `gorm:"column:category" json:"category"`
	City             string     `gorm:"column:city" json:"city"`
	ContactEmail     string     `gorm:"column:contact_email" json:"contact_email"`
	ContactName      string     `gorm:"column:contact_name" json:"contact_name"`
	ContactPhone     string     `gorm:"column:contact_phone" json:"contact_phone"`
	ContractEndAt    time.Time  `gorm:"column:contract_end_at" json:"contract_end_at"`
	ContractStartAt  time.Time  `gorm:"column:contract_start_at" json:"contract_start_at"`
	Country          string     `gorm:"column:country" json:"country"`
	CreatedAt        time.Time  `gorm:"column:created_at" json:"created_at"`
	CreatedBy        int64      `gorm:"column:created_by" json:"created_by"`
	DeletedAt        *time.Time `gorm:"column:deleted_at" json:"deleted_at"`
	Description      string     `gorm:"column:description" json:"description"`
	DrRecommand      int        `gorm:"column:dr_recommand" json:"dr_recommand"`
	ID               int64      `gorm:"column:id;primary_key" json:"id"`
	IsAppVisible     int        `gorm:"column:is_app_visible" json:"is_app_visible"`
	LastUpdatedAt    time.Time  `gorm:"column:last_updated_at" json:"last_updated_at"`
	Latitude         float64    `gorm:"column:latitude" json:"latitude"`
	Longitude        float64    `gorm:"column:longitude" json:"longitude"`
	Name             string     `gorm:"column:name" json:"name"`
	Note             string     `gorm:"column:note" json:"note"`
	OrganizationType string     `gorm:"column:organization_type" json:"organization_type"`
	Phone            string     `gorm:"column:phone" json:"phone"`
	PublishedAt      time.Time  `gorm:"column:published_at" json:"published_at"`
	ParentId         *int64      `gorm:"parent_id" json:"parent_id"`
	Street           string     `gorm:"column:street" json:"street"`
	Type             int        `gorm:"column:type" json:"type"`
	UpdatedAt        *time.Time `gorm:"column:updated_at" json:"updated_at"`
	UpdatedBy        int64      `gorm:"column:updated_by" json:"updated_by"`
	Zipcode          string     `gorm:"column:zipcode" json:"zipcode"`
}

// TableName sets the insert table name for this struct type
func (r *Restaurant) TableName() string {
	return "restaurants"
}
