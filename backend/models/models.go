package models

import (
	"time"

	"gorm.io/gorm"
)

// Person represents an individual who can own accounts.
type Person struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Name      string         `json:"name" gorm:"not null"`
	Accounts  []Account      `json:"accounts,omitempty" gorm:"many2many:person_accounts;"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// Account represents a financial account that can be owned by one or more persons.
type Account struct {
	ID             uint           `json:"id" gorm:"primaryKey"`
	Name           string         `json:"name" gorm:"not null"`
	Balance        float64        `json:"balance" gorm:"default:0"`
	Currency       string         `json:"currency" gorm:"default:EUR"`
	Owners         []Person       `json:"owners,omitempty" gorm:"many2many:person_accounts;"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`
}

// FrequencyType defines how often a position recurs.
type FrequencyType string

const (
	FrequencyDaily        FrequencyType = "daily"
	FrequencyWeekly       FrequencyType = "weekly"
	FrequencyBiweekly     FrequencyType = "biweekly"
	FrequencyMonthly      FrequencyType = "monthly"
	FrequencyQuarterly    FrequencyType = "quarterly"
	FrequencySemiAnnually FrequencyType = "semi_annually"
	FrequencyAnnually     FrequencyType = "annually"
)

// BusinessDayRule defines how to adjust when a payment date falls on a non-business day.
type BusinessDayRule string

const (
	BusinessDayExact              BusinessDayRule = "exact"
	BusinessDayLastBefore         BusinessDayRule = "last_business_day_before"
	BusinessDayFirstAfter         BusinessDayRule = "first_business_day_after"
	BusinessDayLastOfMonth        BusinessDayRule = "last_business_day_of_month"
)

// PositionType defines whether a position is income, expense, or a transfer.
type PositionType string

const (
	PositionIncome   PositionType = "income"
	PositionExpense  PositionType = "expense"
	PositionTransfer PositionType = "transfer"
)

// Position represents a recurring income, expense, or transfer between accounts.
type Position struct {
	ID              uint            `json:"id" gorm:"primaryKey"`
	Name            string          `json:"name" gorm:"not null"`
	Type            PositionType    `json:"type" gorm:"not null"`            // income, expense, transfer
	Amount          float64         `json:"amount" gorm:"not null"`          // positive amount per occurrence
	AccountID       *uint           `json:"accountId"`                       // target account for income/expense
	Account         *Account        `json:"account,omitempty" gorm:"foreignKey:AccountID"`
	SourceAccountID *uint           `json:"sourceAccountId"`                 // source account for transfers
	SourceAccount   *Account        `json:"sourceAccount,omitempty" gorm:"foreignKey:SourceAccountID"`
	TargetAccountID *uint           `json:"targetAccountId"`                 // target account for transfers
	TargetAccount   *Account        `json:"targetAccount,omitempty" gorm:"foreignKey:TargetAccountID"`

	// Schedule configuration
	FrequencyType   FrequencyType   `json:"frequencyType" gorm:"not null"`   // daily, weekly, monthly, etc.
	Interval        int             `json:"interval" gorm:"default:1"`       // every N periods
	DayOfMonth      *int            `json:"dayOfMonth"`                      // 1-31, which day of month
	MonthOfYear     *int            `json:"monthOfYear"`                     // 1-12, for annual frequencies
	DayOfWeek       *int            `json:"dayOfWeek"`                       // 0=Sun..6=Sat, for weekly
	BusinessDayRule BusinessDayRule `json:"businessDayRule" gorm:"default:exact"` // how to handle non-business days

	StartDate       time.Time       `json:"startDate" gorm:"not null"`
	EndDate         *time.Time      `json:"endDate"`                         // nil = indefinite

	CreatedAt       time.Time       `json:"createdAt"`
	UpdatedAt       time.Time       `json:"updatedAt"`
	DeletedAt       gorm.DeletedAt  `json:"-" gorm:"index"`
}
