package models

import "time"

type Quotation struct {
	ActivationCode        string     `gorm:"column:activation_code" json:"activation_code"`
	Active                int        `gorm:"column:active" json:"active"`
	BuyerEmail            string     `gorm:"column:buyer_email" json:"buyer_email"`
	BuyerIdentifier       string     `gorm:"column:buyer_identifier" json:"buyer_identifier"`
	BuyerName             string     `gorm:"column:buyer_name" json:"buyer_name"`
	BuyerPhone            string     `gorm:"column:buyer_phone" json:"buyer_phone"`
	CardInfo              string     `gorm:"column:card_info" json:"card_info"`
	CardKey               string     `gorm:"column:card_key" json:"card_key"`
	CardToken             string     `gorm:"column:card_token" json:"card_token"`
	CarrierID             string     `gorm:"column:carrier_id" json:"carrier_id"`
	CarrierType           string     `gorm:"column:carrier_type" json:"carrier_type"`
	CompanyName           string     `gorm:"column:company_name" json:"company_name"`
	Contents              string     `gorm:"column:contents" json:"contents"`
	CreatedAt             *time.Time `gorm:"column:created_at" json:"created_at"`
	CreatedBy             int64      `gorm:"column:created_by" json:"created_by"`
	DeductionStart        int        `gorm:"column:deduction_start" json:"deduction_start"`
	DeletedAt             *time.Time `gorm:"column:deleted_at" json:"deleted_at"`
	Expired               *time.Time `gorm:"column:expired" json:"expired"`
	ID                    int64      `gorm:"column:id;primary_key" json:"id"`
	MemberID              string     `gorm:"column:member_id" json:"member_id"`
	Name                  string     `gorm:"column:name" json:"name"`
	PayWay                int        `gorm:"column:pay_way" json:"pay_way"`
	PaymentStatus         int        `gorm:"column:payment_status" json:"payment_status"`
	PeriodType            string     `gorm:"column:period_type" json:"period_type"`
	PeriodValue           int        `gorm:"column:period_value" json:"period_value"`
	PlanID                int64      `gorm:"column:plan_id" json:"plan_id"`
	Price                 int        `gorm:"column:price" json:"price"`
	ProductionCode        string     `gorm:"column:production_code" json:"production_code"`
	ProductionDescription string     `gorm:"column:production_description" json:"production_description"`
	QuotationNo           string     `gorm:"column:quotation_no" json:"quotation_no"`
	QuotationSn           string     `gorm:"column:quotation_sn" json:"quotation_sn"`
	ReceiptRemark         string     `gorm:"column:receipt_remark" json:"receipt_remark"`
	UpdatedAt             *time.Time `gorm:"column:updated_at" json:"updated_at"`
}

// TableName sets the insert table name for this struct type
func (q *Quotation) TableName() string {
	return "quotations"
}
