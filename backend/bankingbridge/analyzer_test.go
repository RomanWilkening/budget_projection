package bankingbridge

import (
	"testing"
)

func TestAnalyzeRecurringTransactions_Monthly(t *testing.T) {
	transactions := []BridgeTransaction{
		{BookingDate: "2024-01-15", Name: "Arbeitgeber GmbH", Description: "Gehalt Januar", Amount: 3200.00, IBAN: "DE89370400440532013001", BookingText: "Gehalt/Rente"},
		{BookingDate: "2024-02-15", Name: "Arbeitgeber GmbH", Description: "Gehalt Februar", Amount: 3200.00, IBAN: "DE89370400440532013001", BookingText: "Gehalt/Rente"},
		{BookingDate: "2024-03-15", Name: "Arbeitgeber GmbH", Description: "Gehalt März", Amount: 3200.00, IBAN: "DE89370400440532013001", BookingText: "Gehalt/Rente"},
		{BookingDate: "2024-04-15", Name: "Arbeitgeber GmbH", Description: "Gehalt April", Amount: 3200.00, IBAN: "DE89370400440532013001", BookingText: "Gehalt/Rente"},
		{BookingDate: "2024-05-15", Name: "Arbeitgeber GmbH", Description: "Gehalt Mai", Amount: 3200.00, IBAN: "DE89370400440532013001", BookingText: "Gehalt/Rente"},
		{BookingDate: "2024-06-15", Name: "Arbeitgeber GmbH", Description: "Gehalt Juni", Amount: 3200.00, IBAN: "DE89370400440532013001", BookingText: "Gehalt/Rente"},
	}

	patterns := AnalyzeRecurringTransactions(transactions)
	if len(patterns) != 1 {
		t.Fatalf("Expected 1 pattern, got %d", len(patterns))
	}

	p := patterns[0]
	if p.Name != "Arbeitgeber GmbH" {
		t.Errorf("Expected name 'Arbeitgeber GmbH', got '%s'", p.Name)
	}
	if p.AverageAmount != 3200.00 {
		t.Errorf("Expected average amount 3200.00, got %.2f", p.AverageAmount)
	}
	if p.Frequency != "monthly" {
		t.Errorf("Expected frequency 'monthly', got '%s'", p.Frequency)
	}
	if p.IsExpense {
		t.Error("Expected income (positive amounts), got expense")
	}
	if p.Occurrences != 6 {
		t.Errorf("Expected 6 occurrences, got %d", p.Occurrences)
	}
	if p.Confidence < 0.5 {
		t.Errorf("Expected confidence >= 0.5, got %.2f", p.Confidence)
	}
}

func TestAnalyzeRecurringTransactions_MixedTypes(t *testing.T) {
	transactions := []BridgeTransaction{
		// Monthly rent (expense)
		{BookingDate: "2024-01-01", Name: "Vermieter GmbH", Description: "Miete Januar", Amount: -850.00, IBAN: "DE27100777770209299700", BookingText: "SEPA-Überweisung"},
		{BookingDate: "2024-02-01", Name: "Vermieter GmbH", Description: "Miete Februar", Amount: -850.00, IBAN: "DE27100777770209299700", BookingText: "SEPA-Überweisung"},
		{BookingDate: "2024-03-01", Name: "Vermieter GmbH", Description: "Miete März", Amount: -850.00, IBAN: "DE27100777770209299700", BookingText: "SEPA-Überweisung"},
		{BookingDate: "2024-04-01", Name: "Vermieter GmbH", Description: "Miete April", Amount: -850.00, IBAN: "DE27100777770209299700", BookingText: "SEPA-Überweisung"},
		// Monthly salary (income)
		{BookingDate: "2024-01-28", Name: "Arbeitgeber GmbH", Description: "Gehalt", Amount: 3000.00, IBAN: "DE89370400440532013001", BookingText: "Gehalt/Rente"},
		{BookingDate: "2024-02-28", Name: "Arbeitgeber GmbH", Description: "Gehalt", Amount: 3000.00, IBAN: "DE89370400440532013001", BookingText: "Gehalt/Rente"},
		{BookingDate: "2024-03-28", Name: "Arbeitgeber GmbH", Description: "Gehalt", Amount: 3000.00, IBAN: "DE89370400440532013001", BookingText: "Gehalt/Rente"},
		// One-off transaction (should not be detected)
		{BookingDate: "2024-01-10", Name: "Amazon", Description: "Bestellung", Amount: -29.99, IBAN: "LU123456789", BookingText: "SEPA-Lastschrift"},
	}

	patterns := AnalyzeRecurringTransactions(transactions)
	if len(patterns) != 2 {
		t.Fatalf("Expected 2 patterns, got %d", len(patterns))
	}

	// Patterns should be sorted by confidence
	foundRent := false
	foundSalary := false
	for _, p := range patterns {
		if p.Name == "Vermieter GmbH" {
			foundRent = true
			if !p.IsExpense {
				t.Error("Rent should be an expense")
			}
			if p.AverageAmount != 850.00 {
				t.Errorf("Expected rent amount 850.00, got %.2f", p.AverageAmount)
			}
		}
		if p.Name == "Arbeitgeber GmbH" {
			foundSalary = true
			if p.IsExpense {
				t.Error("Salary should be income")
			}
		}
	}

	if !foundRent {
		t.Error("Expected to find rent pattern")
	}
	if !foundSalary {
		t.Error("Expected to find salary pattern")
	}
}

func TestAnalyzeRecurringTransactions_TooFewOccurrences(t *testing.T) {
	transactions := []BridgeTransaction{
		{BookingDate: "2024-01-15", Name: "One-off Payment", Description: "Test", Amount: -100.00, IBAN: "DE12345", BookingText: "SEPA"},
	}

	patterns := AnalyzeRecurringTransactions(transactions)
	if len(patterns) != 0 {
		t.Fatalf("Expected 0 patterns for single transaction, got %d", len(patterns))
	}
}

func TestAnalyzeRecurringTransactions_VariableAmounts(t *testing.T) {
	transactions := []BridgeTransaction{
		{BookingDate: "2024-01-05", Name: "Stadtwerke", Description: "Strom", Amount: -75.50, IBAN: "DE44500105175407324931", BookingText: "SEPA-Lastschrift"},
		{BookingDate: "2024-02-05", Name: "Stadtwerke", Description: "Strom", Amount: -78.00, IBAN: "DE44500105175407324931", BookingText: "SEPA-Lastschrift"},
		{BookingDate: "2024-03-05", Name: "Stadtwerke", Description: "Strom", Amount: -72.30, IBAN: "DE44500105175407324931", BookingText: "SEPA-Lastschrift"},
		{BookingDate: "2024-04-05", Name: "Stadtwerke", Description: "Strom", Amount: -76.20, IBAN: "DE44500105175407324931", BookingText: "SEPA-Lastschrift"},
	}

	patterns := AnalyzeRecurringTransactions(transactions)
	if len(patterns) != 1 {
		t.Fatalf("Expected 1 pattern, got %d", len(patterns))
	}

	p := patterns[0]
	if p.Name != "Stadtwerke" {
		t.Errorf("Expected name 'Stadtwerke', got '%s'", p.Name)
	}
	if !p.IsExpense {
		t.Error("Expected expense")
	}
	// Average of 75.50, 78.00, 72.30, 76.20 = 75.50
	if p.AverageAmount < 72.0 || p.AverageAmount > 78.5 {
		t.Errorf("Expected average amount around 75.50, got %.2f", p.AverageAmount)
	}
	if p.MinAmount != 72.30 {
		t.Errorf("Expected min amount 72.30, got %.2f", p.MinAmount)
	}
	if p.MaxAmount != 78.00 {
		t.Errorf("Expected max amount 78.00, got %.2f", p.MaxAmount)
	}
	// Last transaction by date is 2024-04-05 with amount -76.20
	if p.LastAmount != 76.20 {
		t.Errorf("Expected last amount 76.20, got %.2f", p.LastAmount)
	}
}

func TestAnalyzeRecurringTransactions_LastAmount(t *testing.T) {
	// Transactions are NOT sorted by date to verify LastAmount picks the most recent
	transactions := []BridgeTransaction{
		{BookingDate: "2024-03-10", Name: "Telekom", Description: "Mobilfunk", Amount: -39.99, IBAN: "DE123", BookingText: "SEPA"},
		{BookingDate: "2024-01-10", Name: "Telekom", Description: "Mobilfunk", Amount: -35.00, IBAN: "DE123", BookingText: "SEPA"},
		{BookingDate: "2024-04-10", Name: "Telekom", Description: "Mobilfunk", Amount: -42.50, IBAN: "DE123", BookingText: "SEPA"},
		{BookingDate: "2024-02-10", Name: "Telekom", Description: "Mobilfunk", Amount: -37.00, IBAN: "DE123", BookingText: "SEPA"},
	}

	patterns := AnalyzeRecurringTransactions(transactions)
	if len(patterns) != 1 {
		t.Fatalf("Expected 1 pattern, got %d", len(patterns))
	}

	p := patterns[0]
	// Most recent transaction is 2024-04-10 with amount -42.50
	if p.LastAmount != 42.50 {
		t.Errorf("Expected last amount 42.50 (most recent transaction), got %.2f", p.LastAmount)
	}
}

func TestAnalyzeRecurringTransactions_Empty(t *testing.T) {
	patterns := AnalyzeRecurringTransactions(nil)
	if len(patterns) != 0 {
		t.Fatalf("Expected 0 patterns for nil transactions, got %d", len(patterns))
	}

	patterns = AnalyzeRecurringTransactions([]BridgeTransaction{})
	if len(patterns) != 0 {
		t.Fatalf("Expected 0 patterns for empty transactions, got %d", len(patterns))
	}
}

func TestDaysBetween(t *testing.T) {
	// Same month: Jan 1 to Jan 31 = 30 days
	d := daysBetween("2024-01-01", "2024-01-31")
	if d != 30 {
		t.Errorf("Expected 30 days, got %d", d)
	}

	// One month apart: Jan 15 to Feb 15 = 31 days (January has 31 days)
	d = daysBetween("2024-01-15", "2024-02-15")
	if d != 31 {
		t.Errorf("Expected 31 days, got %d", d)
	}
}

func TestParseDayOfMonth(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"2024-01-15", 15},
		{"2024-12-01", 1},
		{"2024-03-31", 31},
		{"", 0},
	}

	for _, tt := range tests {
		result := parseDayOfMonth(tt.input)
		if result != tt.expected {
			t.Errorf("parseDayOfMonth(%q) = %d, expected %d", tt.input, result, tt.expected)
		}
	}
}
