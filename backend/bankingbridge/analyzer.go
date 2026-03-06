package bankingbridge

import (
	"math"
	"sort"
	"strings"
	"time"
)

// RecurringPattern represents a detected recurring transaction pattern.
type RecurringPattern struct {
	Name            string  `json:"name"`
	CounterpartIBAN string  `json:"counterpartIban"`
	Description     string  `json:"description"`
	AverageAmount   float64 `json:"averageAmount"`
	MedianAmount    float64 `json:"medianAmount"`
	LastAmount      float64 `json:"lastAmount"`
	MinAmount       float64 `json:"minAmount"`
	MaxAmount       float64 `json:"maxAmount"`
	IsExpense       bool    `json:"isExpense"`
	Frequency       string  `json:"frequency"`       // monthly, quarterly, annually, weekly
	DayOfMonth      *int    `json:"dayOfMonth"`      // most common day of month
	Occurrences     int     `json:"occurrences"`      // how many times detected
	Confidence      float64 `json:"confidence"`       // 0.0-1.0 confidence score
	BookingText     string  `json:"bookingText"`
}

// transactionGroup groups transactions by counterpart.
type transactionGroup struct {
	name        string
	iban        string
	description string
	bookingText string
	amounts     []float64
	days        []int    // day of month for each occurrence
	dates       []string // booking dates
}

// AnalyzeRecurringTransactions analyzes transactions to detect recurring patterns.
func AnalyzeRecurringTransactions(transactions []BridgeTransaction) []RecurringPattern {
	// Group transactions by counterpart (name + IBAN combination)
	groups := groupTransactions(transactions)

	var patterns []RecurringPattern
	for _, group := range groups {
		if len(group.amounts) < 2 {
			continue // Need at least 2 occurrences
		}

		pattern := analyzeGroup(group)
		if pattern != nil && pattern.Confidence >= 0.3 {
			patterns = append(patterns, *pattern)
		}
	}

	// Sort by confidence descending
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Confidence > patterns[j].Confidence
	})

	return patterns
}

// groupTransactions groups transactions by counterpart name and IBAN.
func groupTransactions(transactions []BridgeTransaction) map[string]*transactionGroup {
	groups := make(map[string]*transactionGroup)

	for _, tx := range transactions {
		// Create a key from normalized name + IBAN
		key := normalizeKey(tx.Name, tx.IBAN)

		group, exists := groups[key]
		if !exists {
			group = &transactionGroup{
				name:        tx.Name,
				iban:        tx.IBAN,
				description: tx.Description,
				bookingText: tx.BookingText,
			}
			groups[key] = group
		}

		group.amounts = append(group.amounts, tx.Amount)
		group.dates = append(group.dates, tx.BookingDate)

		// Extract day of month
		if len(tx.BookingDate) >= 10 {
			day := parseDayOfMonth(tx.BookingDate)
			if day > 0 {
				group.days = append(group.days, day)
			}
		}
	}

	return groups
}

func normalizeKey(name, iban string) string {
	normalized := strings.ToLower(strings.TrimSpace(name))
	if iban != "" {
		return normalized + "|" + strings.ToLower(strings.TrimSpace(iban))
	}
	return normalized + "|"
}

func parseDayOfMonth(dateStr string) int {
	// Parse YYYY-MM-DD
	if len(dateStr) < 10 {
		return 0
	}
	day := 0
	for _, c := range dateStr[8:10] {
		day = day*10 + int(c-'0')
	}
	return day
}

func analyzeGroup(group *transactionGroup) *RecurringPattern {
	amounts := group.amounts
	n := len(amounts)
	if n < 2 {
		return nil
	}

	// Check if all amounts have the same sign (all positive or all negative)
	allSameSign := true
	isExpense := amounts[0] < 0
	for _, a := range amounts {
		if (a < 0) != isExpense {
			allSameSign = false
			break
		}
	}

	if !allSameSign {
		return nil // Mixed positive/negative amounts — not a clear recurring pattern
	}

	// Calculate amount statistics using absolute values
	absAmounts := make([]float64, n)
	for i, a := range amounts {
		absAmounts[i] = math.Abs(a)
	}
	sort.Float64s(absAmounts)

	avgAmount := average(absAmounts)
	medianAmount := median(absAmounts)
	minAmount := absAmounts[0]
	maxAmount := absAmounts[n-1]

	// Check amount consistency (coefficient of variation)
	amountCV := coefficientOfVariation(absAmounts)
	amountConsistency := 1.0
	if amountCV > 0.5 {
		amountConsistency = 0.3
	} else if amountCV > 0.2 {
		amountConsistency = 0.6
	} else if amountCV > 0.1 {
		amountConsistency = 0.8
	}

	// Detect frequency from dates
	frequency, freqConfidence := detectFrequency(group.dates, n)

	// Calculate day-of-month consistency
	dayConfidence := 0.0
	var dominantDay *int
	if len(group.days) > 0 {
		day, conf := mostCommonDay(group.days)
		dayConfidence = conf
		dominantDay = &day
	}

	// Find the last (most recent) transaction amount
	lastAmount := math.Abs(amounts[n-1]) // default to last in slice
	if len(group.dates) == n {
		lastIdx := 0
		lastDate := group.dates[0]
		for i, d := range group.dates {
			if d > lastDate {
				lastDate = d
				lastIdx = i
			}
		}
		lastAmount = math.Abs(amounts[lastIdx])
	}

	// Overall confidence
	confidence := freqConfidence*0.5 + amountConsistency*0.3 + dayConfidence*0.2

	// Require minimum occurrences for high confidence
	if n < 3 {
		confidence *= 0.6
	}

	return &RecurringPattern{
		Name:            group.name,
		CounterpartIBAN: group.iban,
		Description:     group.description,
		AverageAmount:   math.Round(avgAmount*100) / 100,
		MedianAmount:    math.Round(medianAmount*100) / 100,
		LastAmount:      math.Round(lastAmount*100) / 100,
		MinAmount:       math.Round(minAmount*100) / 100,
		MaxAmount:       math.Round(maxAmount*100) / 100,
		IsExpense:       isExpense,
		Frequency:       frequency,
		DayOfMonth:      dominantDay,
		Occurrences:     n,
		Confidence:      math.Round(confidence*100) / 100,
		BookingText:     group.bookingText,
	}
}

func detectFrequency(dates []string, occurrences int) (string, float64) {
	if len(dates) < 2 {
		return "monthly", 0.3
	}

	// Sort dates
	sortedDates := make([]string, len(dates))
	copy(sortedDates, dates)
	sort.Strings(sortedDates)

	// Calculate intervals in days between consecutive transactions
	var intervals []int
	for i := 1; i < len(sortedDates); i++ {
		days := daysBetween(sortedDates[i-1], sortedDates[i])
		if days > 0 {
			intervals = append(intervals, days)
		}
	}

	if len(intervals) == 0 {
		return "monthly", 0.3
	}

	avgInterval := averageInt(intervals)

	// Determine frequency based on average interval
	type freqMatch struct {
		name     string
		days     float64
		tolerance float64
	}
	freqOptions := []freqMatch{
		{"weekly", 7, 3},
		{"biweekly", 14, 4},
		{"monthly", 30.44, 8},    // average month length: 365.25/12
		{"quarterly", 91.31, 20}, // average quarter length: 365.25/4
		{"semi_annually", 182.6, 30},
		{"annually", 365.25, 45},
	}

	bestFreq := "monthly"
	bestConf := 0.3

	for _, opt := range freqOptions {
		diff := math.Abs(avgInterval - opt.days)
		if diff <= opt.tolerance {
			// Check how consistently the intervals match
			matchCount := 0
			for _, interval := range intervals {
				if math.Abs(float64(interval)-opt.days) <= opt.tolerance {
					matchCount++
				}
			}
			conf := float64(matchCount) / float64(len(intervals))
			if conf > bestConf {
				bestConf = conf
				bestFreq = opt.name
			}
		}
	}

	return bestFreq, bestConf
}

func daysBetween(date1, date2 string) int {
	t1, err1 := time.Parse("2006-01-02", date1)
	t2, err2 := time.Parse("2006-01-02", date2)
	if err1 != nil || err2 != nil {
		return 0
	}
	diff := t2.Sub(t1)
	days := int(diff.Hours() / 24)
	if days < 0 {
		days = -days
	}
	return days
}

func mostCommonDay(days []int) (int, float64) {
	counts := make(map[int]int)
	for _, d := range days {
		counts[d]++
	}

	maxCount := 0
	maxDay := 0
	for day, count := range counts {
		if count > maxCount {
			maxCount = count
			maxDay = day
		}
	}

	// Also count days that are within ±2 of the dominant day
	nearCount := 0
	for _, d := range days {
		diff := d - maxDay
		if diff < 0 {
			diff = -diff
		}
		if diff <= 2 {
			nearCount++
		}
	}

	return maxDay, float64(nearCount) / float64(len(days))
}

func average(values []float64) float64 {
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func averageInt(values []int) float64 {
	sum := 0
	for _, v := range values {
		sum += v
	}
	return float64(sum) / float64(len(values))
}

func median(sorted []float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}

func coefficientOfVariation(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	avg := average(values)
	if avg == 0 {
		return 0
	}

	sumSqDiff := 0.0
	for _, v := range values {
		diff := v - avg
		sumSqDiff += diff * diff
	}
	stdDev := math.Sqrt(sumSqDiff / float64(len(values)))
	return stdDev / avg
}
