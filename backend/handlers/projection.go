package handlers

import (
	"math"
	"net/http"
	"sort"
	"time"

	"github.com/RomanWilkening/budget_projection/backend/database"
	"github.com/RomanWilkening/budget_projection/backend/models"
	"github.com/gin-gonic/gin"
)

// ProjectionDataPoint represents one data point in the projection time series.
type ProjectionDataPoint struct {
	Date    string  `json:"date"`
	Balance float64 `json:"balance"`
}

// AccountProjection contains the projected balance over time for one account.
type AccountProjection struct {
	ID         uint                  `json:"id"`
	Name       string                `json:"name"`
	Currency   string                `json:"currency"`
	DataPoints []ProjectionDataPoint `json:"dataPoints"`
}

// ProjectionResponse is the API response for the projection endpoint.
type ProjectionResponse struct {
	Accounts []AccountProjection   `json:"accounts"`
	Totals   []ProjectionDataPoint `json:"totals"`
}

// GetProjection computes projected account balances over a time period.
// Query parameters:
//   - months: number of months to project (default: 6)
//   - startDate: projection start date in YYYY-MM-DD (default: today)
//   - granularity: "daily", "weekly", or "monthly" (default: auto based on months)
func GetProjection(c *gin.Context) {
	// Parse parameters
	months := 6
	if m := c.Query("months"); m != "" {
		if _, err := parsePositiveInt(m); err == nil {
			months = int(mustParseInt(m))
		}
	}
	if months < 1 {
		months = 1
	}
	if months > 120 {
		months = 120
	}

	startDate := time.Now().Truncate(24 * time.Hour)
	if sd := c.Query("startDate"); sd != "" {
		if parsed, err := time.Parse("2006-01-02", sd); err == nil {
			startDate = parsed
		}
	}

	endDate := startDate.AddDate(0, months, 0)

	granularity := c.Query("granularity")
	if granularity == "" {
		// Auto-determine granularity
		if months <= 3 {
			granularity = "daily"
		} else if months <= 12 {
			granularity = "weekly"
		} else {
			granularity = "monthly"
		}
	}

	// Load accounts and positions
	var accounts []models.Account
	if err := database.DB.Find(&accounts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var positions []models.Position
	if err := database.DB.Find(&positions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Build balance map: account ID -> current balance
	balances := make(map[uint]float64)
	accountMap := make(map[uint]models.Account)
	for _, a := range accounts {
		balances[a.ID] = a.Balance
		accountMap[a.ID] = a
	}

	// Generate all position events within the projection period
	type balanceEvent struct {
		date      time.Time
		accountID uint
		amount    float64 // positive = credit, negative = debit
	}

	var events []balanceEvent

	for _, pos := range positions {
		occurrences := generateOccurrences(pos, startDate, endDate)
		for _, date := range occurrences {
			switch pos.Type {
			case models.PositionIncome:
				if pos.AccountID != nil {
					events = append(events, balanceEvent{
						date:      date,
						accountID: *pos.AccountID,
						amount:    pos.Amount,
					})
				}
				// Optional: debit source account
				if pos.SourceAccountID != nil {
					events = append(events, balanceEvent{
						date:      date,
						accountID: *pos.SourceAccountID,
						amount:    -pos.Amount,
					})
				}
			case models.PositionExpense:
				if pos.AccountID != nil {
					events = append(events, balanceEvent{
						date:      date,
						accountID: *pos.AccountID,
						amount:    -pos.Amount,
					})
				}
				// Optional: credit target account
				if pos.TargetAccountID != nil {
					events = append(events, balanceEvent{
						date:      date,
						accountID: *pos.TargetAccountID,
						amount:    pos.Amount,
					})
				}
			case models.PositionTransfer:
				if pos.SourceAccountID != nil {
					events = append(events, balanceEvent{
						date:      date,
						accountID: *pos.SourceAccountID,
						amount:    -pos.Amount,
					})
				}
				if pos.TargetAccountID != nil {
					events = append(events, balanceEvent{
						date:      date,
						accountID: *pos.TargetAccountID,
						amount:    pos.Amount,
					})
				}
			}
		}
	}

	// Sort events by date
	sort.Slice(events, func(i, j int) bool {
		return events[i].date.Before(events[j].date)
	})

	// Generate time series dates based on granularity
	dates := generateTimeSeriesDates(startDate, endDate, granularity)

	// For each date, compute the balance after applying all events up to that date
	// We'll walk through events and dates together
	eventIdx := 0

	// Build account projections
	accountProjections := make(map[uint][]ProjectionDataPoint)
	for _, a := range accounts {
		accountProjections[a.ID] = make([]ProjectionDataPoint, 0, len(dates))
	}

	totals := make([]ProjectionDataPoint, 0, len(dates))

	for _, d := range dates {
		endOfDay := d.Add(24*time.Hour - time.Second)

		// Apply all events up to and including this date
		for eventIdx < len(events) && !events[eventIdx].date.After(endOfDay) {
			e := events[eventIdx]
			balances[e.accountID] += e.amount
			eventIdx++
		}

		// Record data point for each account
		dateStr := d.Format("2006-01-02")
		total := 0.0
		for _, a := range accounts {
			bal := balances[a.ID]
			accountProjections[a.ID] = append(accountProjections[a.ID], ProjectionDataPoint{
				Date:    dateStr,
				Balance: roundToTwoDecimals(bal),
			})
			total += bal
		}
		totals = append(totals, ProjectionDataPoint{
			Date:    dateStr,
			Balance: roundToTwoDecimals(total),
		})
	}

	// Build response
	result := ProjectionResponse{
		Accounts: make([]AccountProjection, 0, len(accounts)),
		Totals:   totals,
	}
	for _, a := range accounts {
		result.Accounts = append(result.Accounts, AccountProjection{
			ID:         a.ID,
			Name:       a.Name,
			Currency:   a.Currency,
			DataPoints: accountProjections[a.ID],
		})
	}

	c.JSON(http.StatusOK, result)
}

// generateOccurrences returns all dates when a position occurs within [start, end).
func generateOccurrences(pos models.Position, start, end time.Time) []time.Time {
	var results []time.Time

	posStart := pos.StartDate.Time
	if posStart.IsZero() {
		return results
	}

	// If position hasn't started yet by end date, no occurrences
	if posStart.After(end) || posStart.Equal(end) {
		return results
	}

	// Determine effective end
	effectiveEnd := end
	if pos.EndDate != nil && !pos.EndDate.Time.IsZero() && pos.EndDate.Time.Before(end) {
		effectiveEnd = pos.EndDate.Time.Add(24 * time.Hour) // include the end date itself
	}

	interval := pos.Interval
	if interval < 1 {
		interval = 1
	}

	switch pos.FrequencyType {
	case models.FrequencyDaily:
		current := posStart
		for !current.After(effectiveEnd) && current.Before(effectiveEnd) {
			if !current.Before(start) {
				results = append(results, applyBusinessDayRule(current, pos.BusinessDayRule))
			}
			current = current.AddDate(0, 0, interval)
		}

	case models.FrequencyWeekly:
		current := posStart
		// Adjust to the desired day of week if specified
		if pos.DayOfWeek != nil {
			dow := time.Weekday(*pos.DayOfWeek)
			for current.Weekday() != dow {
				current = current.AddDate(0, 0, 1)
			}
		}
		for current.Before(effectiveEnd) {
			if !current.Before(start) {
				results = append(results, applyBusinessDayRule(current, pos.BusinessDayRule))
			}
			current = current.AddDate(0, 0, 7*interval)
		}

	case models.FrequencyBiweekly:
		current := posStart
		if pos.DayOfWeek != nil {
			dow := time.Weekday(*pos.DayOfWeek)
			for current.Weekday() != dow {
				current = current.AddDate(0, 0, 1)
			}
		}
		for current.Before(effectiveEnd) {
			if !current.Before(start) {
				results = append(results, applyBusinessDayRule(current, pos.BusinessDayRule))
			}
			current = current.AddDate(0, 0, 14*interval)
		}

	case models.FrequencyMonthly:
		day := 1
		if pos.DayOfMonth != nil {
			day = *pos.DayOfMonth
		}
		// Start from the position's start month
		current := time.Date(posStart.Year(), posStart.Month(), 1, 0, 0, 0, 0, time.UTC)
		for current.Before(effectiveEnd) {
			dateInMonth := clampDayToMonth(current.Year(), current.Month(), day)
			if !dateInMonth.Before(posStart) && !dateInMonth.Before(start) && dateInMonth.Before(effectiveEnd) {
				results = append(results, applyBusinessDayRule(dateInMonth, pos.BusinessDayRule))
			}
			current = current.AddDate(0, interval, 0)
		}

	case models.FrequencyQuarterly:
		day := 1
		if pos.DayOfMonth != nil {
			day = *pos.DayOfMonth
		}
		current := time.Date(posStart.Year(), posStart.Month(), 1, 0, 0, 0, 0, time.UTC)
		for current.Before(effectiveEnd) {
			dateInMonth := clampDayToMonth(current.Year(), current.Month(), day)
			if !dateInMonth.Before(posStart) && !dateInMonth.Before(start) && dateInMonth.Before(effectiveEnd) {
				results = append(results, applyBusinessDayRule(dateInMonth, pos.BusinessDayRule))
			}
			current = current.AddDate(0, 3*interval, 0)
		}

	case models.FrequencySemiAnnually:
		day := 1
		if pos.DayOfMonth != nil {
			day = *pos.DayOfMonth
		}
		current := time.Date(posStart.Year(), posStart.Month(), 1, 0, 0, 0, 0, time.UTC)
		for current.Before(effectiveEnd) {
			dateInMonth := clampDayToMonth(current.Year(), current.Month(), day)
			if !dateInMonth.Before(posStart) && !dateInMonth.Before(start) && dateInMonth.Before(effectiveEnd) {
				results = append(results, applyBusinessDayRule(dateInMonth, pos.BusinessDayRule))
			}
			current = current.AddDate(0, 6*interval, 0)
		}

	case models.FrequencyAnnually:
		day := 1
		if pos.DayOfMonth != nil {
			day = *pos.DayOfMonth
		}
		month := posStart.Month()
		if pos.MonthOfYear != nil {
			month = time.Month(*pos.MonthOfYear)
		}
		current := time.Date(posStart.Year(), month, 1, 0, 0, 0, 0, time.UTC)
		for current.Before(effectiveEnd) {
			dateInMonth := clampDayToMonth(current.Year(), current.Month(), day)
			if !dateInMonth.Before(posStart) && !dateInMonth.Before(start) && dateInMonth.Before(effectiveEnd) {
				results = append(results, applyBusinessDayRule(dateInMonth, pos.BusinessDayRule))
			}
			current = current.AddDate(interval, 0, 0)
		}
	}

	return results
}

// clampDayToMonth returns the given day in the given month, clamping to the last day of the month
// if the day exceeds the month's length.
func clampDayToMonth(year int, month time.Month, day int) time.Time {
	// Get the last day of the month
	lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if day > lastDay {
		day = lastDay
	}
	if day < 1 {
		day = 1
	}
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// applyBusinessDayRule adjusts a date according to the business day rule.
func applyBusinessDayRule(date time.Time, rule models.BusinessDayRule) time.Time {
	switch rule {
	case models.BusinessDayLastBefore:
		for isWeekend(date) {
			date = date.AddDate(0, 0, -1)
		}
	case models.BusinessDayFirstAfter:
		for isWeekend(date) {
			date = date.AddDate(0, 0, 1)
		}
	case models.BusinessDayLastOfMonth:
		// Go to the last day of the month, then back to a business day
		lastDay := time.Date(date.Year(), date.Month()+1, 0, 0, 0, 0, 0, time.UTC)
		for isWeekend(lastDay) {
			lastDay = lastDay.AddDate(0, 0, -1)
		}
		return lastDay
	default:
		// "exact" - no adjustment
	}
	return date
}

// isWeekend returns true if the date is a Saturday or Sunday.
func isWeekend(date time.Time) bool {
	return date.Weekday() == time.Saturday || date.Weekday() == time.Sunday
}

// generateTimeSeriesDates generates the dates for the time series based on granularity.
func generateTimeSeriesDates(start, end time.Time, granularity string) []time.Time {
	var dates []time.Time
	current := start

	switch granularity {
	case "daily":
		for !current.After(end) {
			dates = append(dates, current)
			current = current.AddDate(0, 0, 1)
		}
	case "weekly":
		for !current.After(end) {
			dates = append(dates, current)
			current = current.AddDate(0, 0, 7)
		}
	case "monthly":
		for !current.After(end) {
			dates = append(dates, current)
			current = current.AddDate(0, 1, 0)
		}
	default:
		// Default to weekly
		for !current.After(end) {
			dates = append(dates, current)
			current = current.AddDate(0, 0, 7)
		}
	}

	return dates
}

func roundToTwoDecimals(v float64) float64 {
	return math.Round(v*100) / 100
}

func parsePositiveInt(s string) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, errorf("invalid number: %s", s)
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

func mustParseInt(s string) int {
	n, _ := parsePositiveInt(s)
	return n
}
