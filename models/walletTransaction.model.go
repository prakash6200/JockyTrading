package models

import (
	"time"

	"gorm.io/gorm"
)

// TransactionType defines the type of wallet transaction
type TransactionType string

const (
	TransactionTypeDeposit      TransactionType = "DEPOSIT"
	TransactionTypeWithdrawal   TransactionType = "WITHDRAWAL"
	TransactionTypeSubscription TransactionType = "SUBSCRIPTION"
	TransactionTypeRefund       TransactionType = "REFUND"
	TransactionTypeAdminCredit  TransactionType = "ADMIN_CREDIT"
	TransactionTypeAdminDebit   TransactionType = "ADMIN_DEBIT"
)

// TransactionStatus defines the status of a transaction
type TransactionStatus string

const (
	TransactionStatusPending   TransactionStatus = "PENDING"
	TransactionStatusCompleted TransactionStatus = "COMPLETED"
	TransactionStatusFailed    TransactionStatus = "FAILED"
	TransactionStatusRefunded  TransactionStatus = "REFUNDED"
)

// WalletTransaction tracks all wallet transactions for a user
type WalletTransaction struct {
	gorm.Model
	UserID          uint              `gorm:"not null;index" json:"userId"`
	TransactionType TransactionType   `gorm:"type:varchar(50);not null" json:"transactionType"`
	Amount          float64           `gorm:"not null" json:"amount"`
	BalanceBefore   float64           `gorm:"not null" json:"balanceBefore"`
	BalanceAfter    float64           `gorm:"not null" json:"balanceAfter"`
	Status          TransactionStatus `gorm:"type:varchar(20);default:'COMPLETED'" json:"status"`
	Description     string            `gorm:"type:text" json:"description"`

	// Payment gateway details (for deposits)
	PaymentGateway     string `gorm:"type:varchar(50)" json:"paymentGateway"`    // razorpay, phonepe, etc.
	PaymentOrderID     string `gorm:"type:varchar(100)" json:"paymentOrderId"`   // Order ID from gateway
	PaymentID          string `gorm:"type:varchar(100);index" json:"paymentId"`  // Transaction ID from gateway
	PaymentSignature   string `gorm:"type:varchar(255)" json:"paymentSignature"` // Signature for verification
	PaymentMethod      string `gorm:"type:varchar(50)" json:"paymentMethod"`     // UPI, card, netbanking
	PaymentStatus      string `gorm:"type:varchar(50)" json:"paymentStatus"`     // success, failed
	PaymentResponseRaw string `gorm:"type:text" json:"paymentResponseRaw"`       // Full response JSON

	// Reference details (for subscriptions)
	ReferenceType string `gorm:"type:varchar(50)" json:"referenceType"`  // basket, course, etc.
	ReferenceID   uint   `gorm:"default:0" json:"referenceId"`           // basket_id, course_id
	ReferenceName string `gorm:"type:varchar(255)" json:"referenceName"` // basket name, course name

	// Admin details (for manual credits/debits)
	AdminID uint   `gorm:"default:0" json:"adminId"`
	Reason  string `gorm:"type:text" json:"reason"`

	TransactionDate time.Time `gorm:"not null" json:"transactionDate"`
	IsDeleted       bool      `gorm:"default:false" json:"isDeleted"`

	// Relations - omit in JSON by default (only load when needed)
	User User `gorm:"foreignKey:UserID" json:"-"`
}

func (WalletTransaction) TableName() string {
	return "wallet_transactions"
}
