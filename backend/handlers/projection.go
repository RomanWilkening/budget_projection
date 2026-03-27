package handlers

import (
	"math"
	"net/http"
	"sort"
	"strconv"
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
	ID              uint                  `json:"id"`
	Name            string                `json:"name"`
	Currency        string                `json:"currency"`
	DataPoints      []ProjectionDataPoint `json:"dataPoints"`
	MonthlyNetFlow  float64               `json:"monthlyNetFlow"`
}

// DepotProjection contains the projected value over time for one depot.
type DepotProjection struct {
	ID           uint                  `json:"id"`
	Name         string                `json:"name"`
	InterestRate float64               `json:"interestRate"`
	DataPoints   []ProjectionDataPoint `json:"dataPoints"`
}

// ProjectionResponse is the API response for the projection endpoint.
type ProjectionResponse struct {
	Accounts               []AccountProjection   `json:"accounts"`
	Depots                 []DepotProjection     `json:"depots"`
	Totals                 []ProjectionDataPoint `json:"totals"`
	InflationAdjustedTotals []ProjectionDataPoint `json:"inflationAdjustedTotals,omitempty"`
}

// ScenarioRequest contains temporary position modifications for scenario projections.
// These modifications are applied in-memory only and do not change stored data.
type ScenarioRequest struct {
	ModifiedPositions  []models.Position `json:"modifiedPositions"`  // existing positions with changed fields
	RemovedPositionIDs []uint            `json:"removedPositionIds"` // IDs of positions to exclude
	NewPositions       []models.Position `json:"newPositions"`       // virtual positions to add
}

// GetProjection computes projected account balances over a time period.
// Query parameters:
//   - months: number of months to project (default: 6)
//   - startDate: projection start date in YYYY-MM-DD (default: today)
//   - granularity: "daily", "weekly", or "monthly" (default: auto based on months)
func GetProjection(c *gin.Context) {
	projectionHandler(c, nil)
}

// PostProjectionScenario computes projected account balances with scenario modifications.
// Accepts the same query parameters as GetProjection, plus a JSON body with ScenarioRequest.
func PostProjectionScenario(c *gin.Context) {
	var scenario ScenarioRequest
	if err := c.ShouldBindJSON(&scenario); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	projectionHandler(c, &scenario)
}

// projectionHandler is the shared implementation for both GET and POST projection endpoints.
func projectionHandler(c *gin.Context, scenario *ScenarioRequest) {
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
	if months > 600 {
		months = 600
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

	// Parse inflation rate (annual percent, e.g. 2.0 = 2%)
	inflationRate := 0.0
	if ir := c.Query("inflationRate"); ir != "" {
		if parsed, err := strconv.ParseFloat(ir, 64); err == nil && parsed >= 0 && parsed <= 100 {
			inflationRate = parsed
		}
	}

	// Load accounts, positions, and depots
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

	var depots []models.Depot
	if err := database.DB.Preload("Accounts").Find(&depots).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Apply scenario modifications if provided
	if scenario != nil {
		positions = applyScenario(positions, scenario)
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
			// Apply per-position growth rate: scale amount by (1 + rate/100)^yearsElapsed
			amount := pos.Amount
			if pos.GrowthRate != 0 {
				yearsElapsed := date.Sub(pos.StartDate.Time).Hours() / (24.0 * 365.25)
				if yearsElapsed > 0 {
					amount = pos.Amount * math.Pow(1+pos.GrowthRate/100.0, yearsElapsed)
				}
			}
			switch pos.Type {
			case models.PositionIncome:
				if pos.AccountID != nil {
					events = append(events, balanceEvent{
						date:      date,
						accountID: *pos.AccountID,
						amount:    amount,
					})
				}
				// Optional: debit source account
				if pos.SourceAccountID != nil {
					events = append(events, balanceEvent{
						date:      date,
						accountID: *pos.SourceAccountID,
						amount:    -amount,
					})
				}
			case models.PositionExpense:
				if pos.AccountID != nil {
					events = append(events, balanceEvent{
						date:      date,
						accountID: *pos.AccountID,
						amount:    -amount,
					})
				}
				// Optional: credit target account
				if pos.TargetAccountID != nil {
					events = append(events, balanceEvent{
						date:      date,
						accountID: *pos.TargetAccountID,
						amount:    amount,
					})
				}
			case models.PositionTransfer:
				if pos.SourceAccountID != nil {
					events = append(events, balanceEvent{
						date:      date,
						accountID: *pos.SourceAccountID,
						amount:    -amount,
					})
				}
				if pos.TargetAccountID != nil {
					events = append(events, balanceEvent{
						date:      date,
						accountID: *pos.TargetAccountID,
						amount:    amount,
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
		Depots:   make([]DepotProjection, 0, len(depots)),
		Totals:   totals,
	}

	// Compute inflation-adjusted totals if inflation rate is set
	if inflationRate > 0 && len(totals) > 0 {
		adjustedTotals := make([]ProjectionDataPoint, len(totals))
		for i, tp := range totals {
			yearsFromStart := dates[i].Sub(startDate).Hours() / (24.0 * 365.25)
			discountFactor := math.Pow(1+inflationRate/100.0, yearsFromStart)
			adjustedTotals[i] = ProjectionDataPoint{
				Date:    tp.Date,
				Balance: roundToTwoDecimals(tp.Balance / discountFactor),
			}
		}
		result.InflationAdjustedTotals = adjustedTotals
	}
	for _, a := range accounts {
		pts := accountProjections[a.ID]
		var monthlyNet float64
		if len(pts) >= 1 && months > 0 {
			monthlyNet = roundToTwoDecimals((pts[len(pts)-1].Balance - a.Balance) / float64(months))
		}
		result.Accounts = append(result.Accounts, AccountProjection{
			ID:             a.ID,
			Name:           a.Name,
			Currency:       a.Currency,
			DataPoints:     pts,
			MonthlyNetFlow: monthlyNet,
		})
	}

	// Build depot projections: aggregate linked account balances and apply compound interest
	for _, depot := range depots {
		dp := DepotProjection{
			ID:           depot.ID,
			Name:         depot.Name,
			InterestRate: depot.InterestRate,
			DataPoints:   make([]ProjectionDataPoint, 0, len(dates)),
		}

		// Collect linked account IDs
		linkedAccountIDs := make(map[uint]bool)
		for _, a := range depot.Accounts {
			linkedAccountIDs[a.ID] = true
		}

		// Daily interest rate from annual rate
		dailyRate := depot.InterestRate / 100.0 / 365.0

		// Track accumulated interest separately
		accumulatedInterest := 0.0
		var prevDate time.Time

		for i, d := range dates {
			// Sum up account balances for this depot at this date
			accountSum := 0.0
			for aID := range linkedAccountIDs {
				if pts, ok := accountProjections[aID]; ok && i < len(pts) {
					accountSum += pts[i].Balance
				}
			}

			// Apply compound interest: grow based on days elapsed since last data point
			if i > 0 && dailyRate != 0 {
				daysElapsed := d.Sub(prevDate).Hours() / 24.0
				// Compound on the previous total depot value (accounts + interest)
				lastValue := dp.DataPoints[i-1].Balance
				interestForPeriod := lastValue * dailyRate * daysElapsed
				accumulatedInterest += interestForPeriod
			}

			depotValue := accountSum + accumulatedInterest
			dp.DataPoints = append(dp.DataPoints, ProjectionDataPoint{
				Date:    d.Format("2006-01-02"),
				Balance: roundToTwoDecimals(depotValue),
			})
			prevDate = d
		}

		result.Depots = append(result.Depots, dp)
	}

	c.JSON(http.StatusOK, result)
}

// applyScenario applies scenario modifications to the positions list without modifying the originals.
func applyScenario(positions []models.Position, scenario *ScenarioRequest) []models.Position {
	// Build set of removed IDs
	removedIDs := make(map[uint]bool, len(scenario.RemovedPositionIDs))
	for _, id := range scenario.RemovedPositionIDs {
		removedIDs[id] = true
	}

	// Build map of modified positions (by ID)
	modifiedMap := make(map[uint]models.Position, len(scenario.ModifiedPositions))
	for _, p := range scenario.ModifiedPositions {
		modifiedMap[p.ID] = p
	}

	// Build result: filter out removed, replace modified
	result := make([]models.Position, 0, len(positions)+len(scenario.NewPositions))
	for _, pos := range positions {
		if removedIDs[pos.ID] {
			continue
		}
		if mod, ok := modifiedMap[pos.ID]; ok {
			result = append(result, mod)
		} else {
			result = append(result, pos)
		}
	}

	// Append new virtual positions
	result = append(result, scenario.NewPositions...)

	return result
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
			current = current.AddDate(0, 3*interval, 0)
		}

	case models.FrequencySemiAnnually:
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
