package models

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// FlexDate is a time.Time that accepts both "2006-01-02" and RFC3339 formats during JSON
// unmarshaling, allowing date inputs from HTML <input type="date"> elements.
type FlexDate struct {
	time.Time
}

func (d *FlexDate) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	if s == "null" || s == "" {
		d.Time = time.Time{}
		return nil
	}
	// Try date-only format first
	if t, err := time.Parse("2006-01-02", s); err == nil {
		d.Time = t
		return nil
	}
	// Fall back to RFC3339
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return fmt.Errorf("cannot parse %q as date", s)
	}
	d.Time = t
	return nil
}

func (d FlexDate) MarshalJSON() ([]byte, error) {
	if d.Time.IsZero() {
		return []byte("null"), nil
	}
	return []byte(`"` + d.Time.Format(time.RFC3339) + `"`), nil
}

// Value implements driver.Valuer so GORM can store FlexDate in the database.
func (d FlexDate) Value() (driver.Value, error) {
	return d.Time, nil
}

// Scan implements sql.Scanner so GORM can read FlexDate from the database.
func (d *FlexDate) Scan(value interface{}) error {
	switch v := value.(type) {
	case time.Time:
		d.Time = v
	case string:
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			t, err = time.Parse("2006-01-02", v)
			if err != nil {
				return fmt.Errorf("cannot scan %q as FlexDate", v)
			}
		}
		d.Time = t
	case []byte:
		return d.Scan(string(v))
	case nil:
		d.Time = time.Time{}
	default:
		return fmt.Errorf("unsupported type for FlexDate: %T", value)
	}
	return nil
}

// Setting stores application configuration as key-value pairs.
type Setting struct {
	Key   string `json:"key" gorm:"primaryKey"`
	Value string `json:"value"`
}

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
	ID                      uint           `json:"id" gorm:"primaryKey"`
	Name                    string         `json:"name" gorm:"not null"`
	Balance                 float64        `json:"balance" gorm:"default:0"`
	Currency                string         `json:"currency" gorm:"default:EUR"`
	BankingBridgeAccountID  *int           `json:"bankingBridgeAccountId,omitempty" gorm:"default:null"`
	Owners                  []Person       `json:"owners,omitempty" gorm:"many2many:person_accounts;"`
	CreatedAt               time.Time      `json:"createdAt"`
	UpdatedAt               time.Time      `json:"updatedAt"`
	DeletedAt               gorm.DeletedAt `json:"-" gorm:"index"`
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

	StartDate       FlexDate        `json:"startDate" gorm:"not null"`
	EndDate         *FlexDate       `json:"endDate"`                         // nil = indefinite

	SortOrder       int             `json:"sortOrder" gorm:"default:0"`      // user-defined display order

	CreatedAt       time.Time       `json:"createdAt"`
	UpdatedAt       time.Time       `json:"updatedAt"`
	DeletedAt       gorm.DeletedAt  `json:"-" gorm:"index"`
}

// PositionSeparator represents a user-defined visual separator between position groups.
type PositionSeparator struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Name      string         `json:"name" gorm:"not null"`
	SortOrder int            `json:"sortOrder" gorm:"default:0"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}
