package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/RomanWilkening/budget_projection/backend/database"
	"github.com/RomanWilkening/budget_projection/backend/handlers"
	"github.com/RomanWilkening/budget_projection/backend/models"
	"github.com/gin-gonic/gin"
)

func setupTestDB(t *testing.T) {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "test-budget-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp db: %v", err)
	}
	tmpFile.Close()
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })
	os.Setenv("DB_PATH", tmpFile.Name())
	database.Init()
}

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	api := r.Group("/api")
	{
		persons := api.Group("/persons")
		{
			persons.GET("", handlers.ListPersons)
			persons.GET("/:id", handlers.GetPerson)
			persons.POST("", handlers.CreatePerson)
			persons.PUT("/:id", handlers.UpdatePerson)
			persons.DELETE("/:id", handlers.DeletePerson)
		}
		accounts := api.Group("/accounts")
		{
			accounts.GET("", handlers.ListAccounts)
			accounts.GET("/:id", handlers.GetAccount)
			accounts.POST("", handlers.CreateAccount)
			accounts.PUT("/:id", handlers.UpdateAccount)
			accounts.DELETE("/:id", handlers.DeleteAccount)
			accounts.POST("/:id/owners", handlers.AddAccountOwner)
			accounts.DELETE("/:id/owners/:personId", handlers.RemoveAccountOwner)
			accounts.POST("/:id/link-banking-bridge", handlers.LinkBankingBridgeAccount)
			accounts.POST("/:id/sync-balance", handlers.SyncAccountBalance)
			accounts.GET("/:id/recurring-transactions", handlers.AnalyzeRecurringTransactions)
		}
		positions := api.Group("/positions")
		{
			positions.GET("", handlers.ListPositions)
			positions.GET("/:id", handlers.GetPosition)
			positions.POST("", handlers.CreatePosition)
			positions.PUT("/:id", handlers.UpdatePosition)
			positions.DELETE("/:id", handlers.DeletePosition)
			positions.PUT("", handlers.ReorderPositions)
		}
		separators := api.Group("/position-separators")
		{
			separators.GET("", handlers.ListSeparators)
			separators.POST("", handlers.CreateSeparator)
			separators.PUT("/:id", handlers.UpdateSeparator)
			separators.DELETE("/:id", handlers.DeleteSeparator)
		}
		api.GET("/projection", handlers.GetProjection)

		bridge := api.Group("/banking-bridge")
		{
			bridge.GET("/status", handlers.GetBankingBridgeStatus)
			bridge.GET("/accounts", handlers.ListBankingBridgeAccounts)
			bridge.POST("/sync-all-balances", handlers.SyncAllBalances)
		}
		settings := api.Group("/settings")
		{
			settings.GET("", handlers.GetSettings)
			settings.PUT("", handlers.UpdateSettings)
		}
	}
	return r
}

func TestPersonCRUD(t *testing.T) {
	setupTestDB(t)
	router := setupRouter()

	// Create person
	body, _ := json.Marshal(map[string]string{"name": "Alice"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/persons", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var person models.Person
	json.Unmarshal(w.Body.Bytes(), &person)
	if person.Name != "Alice" {
		t.Fatalf("Expected name 'Alice', got '%s'", person.Name)
	}
	if person.ID == 0 {
		t.Fatal("Expected non-zero ID")
	}

	// List persons
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/persons", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var persons []models.Person
	json.Unmarshal(w.Body.Bytes(), &persons)
	if len(persons) != 1 {
		t.Fatalf("Expected 1 person, got %d", len(persons))
	}

	// Get person
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/persons/1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	// Update person
	body, _ = json.Marshal(map[string]string{"name": "Alice Updated"})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", "/api/persons/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	json.Unmarshal(w.Body.Bytes(), &person)
	if person.Name != "Alice Updated" {
		t.Fatalf("Expected name 'Alice Updated', got '%s'", person.Name)
	}

	// Delete person
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/api/persons/1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	// Verify deleted
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/persons/1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected 404, got %d", w.Code)
	}
}

func TestAccountCRUD(t *testing.T) {
	setupTestDB(t)
	router := setupRouter()

	// Create a person first
	body, _ := json.Marshal(map[string]string{"name": "Bob"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/persons", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d", w.Code)
	}

	// Create account with owner
	accountData := map[string]interface{}{
		"name":     "Girokonto",
		"balance":  1500.50,
		"currency": "EUR",
		"ownerIds": []uint{1},
	}
	body, _ = json.Marshal(accountData)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/accounts", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var account models.Account
	json.Unmarshal(w.Body.Bytes(), &account)
	if account.Name != "Girokonto" {
		t.Fatalf("Expected name 'Girokonto', got '%s'", account.Name)
	}
	if len(account.Owners) != 1 {
		t.Fatalf("Expected 1 owner, got %d", len(account.Owners))
	}

	// List accounts
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/accounts", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	// Update account
	updateData := map[string]interface{}{
		"name":     "Girokonto Updated",
		"balance":  2000.00,
		"currency": "EUR",
		"ownerIds": []uint{1},
	}
	body, _ = json.Marshal(updateData)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", "/api/accounts/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Delete account
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/api/accounts/1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
}

func TestAccountOwnership(t *testing.T) {
	setupTestDB(t)
	router := setupRouter()

	// Create two persons
	for _, name := range []string{"Alice", "Bob"} {
		body, _ := json.Marshal(map[string]string{"name": name})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/persons", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("Expected 201, got %d", w.Code)
		}
	}

	// Create account owned by Alice
	accountData := map[string]interface{}{
		"name":     "Shared Account",
		"balance":  1000.0,
		"currency": "EUR",
		"ownerIds": []uint{1},
	}
	body, _ := json.Marshal(accountData)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/accounts", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d", w.Code)
	}

	// Add Bob as co-owner
	body, _ = json.Marshal(map[string]uint{"personId": 2})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/accounts/1/owners", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var account models.Account
	json.Unmarshal(w.Body.Bytes(), &account)
	if len(account.Owners) != 2 {
		t.Fatalf("Expected 2 owners, got %d", len(account.Owners))
	}

	// Remove Alice as owner
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/api/accounts/1/owners/1", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	json.Unmarshal(w.Body.Bytes(), &account)
	if len(account.Owners) != 1 {
		t.Fatalf("Expected 1 owner, got %d", len(account.Owners))
	}
}

func TestPositionCRUD(t *testing.T) {
	setupTestDB(t)
	router := setupRouter()

	// Create a person and account first
	body, _ := json.Marshal(map[string]string{"name": "Charlie"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/persons", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	accountData := map[string]interface{}{
		"name":     "Gehaltskonto",
		"balance":  0,
		"currency": "EUR",
		"ownerIds": []uint{1},
	}
	body, _ = json.Marshal(accountData)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/accounts", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	// Create income position
	dayOfMonth := 1
	accountID := uint(1)
	positionData := map[string]interface{}{
		"name":            "Gehalt",
		"type":            "income",
		"amount":          3500.00,
		"accountId":       accountID,
		"frequencyType":   "monthly",
		"interval":        1,
		"dayOfMonth":      dayOfMonth,
		"businessDayRule": "last_business_day_before",
		"startDate":       "2024-01-01T00:00:00Z",
	}
	body, _ = json.Marshal(positionData)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/positions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var position models.Position
	json.Unmarshal(w.Body.Bytes(), &position)
	if position.Name != "Gehalt" {
		t.Fatalf("Expected name 'Gehalt', got '%s'", position.Name)
	}
	if position.FrequencyType != "monthly" {
		t.Fatalf("Expected frequencyType 'monthly', got '%s'", position.FrequencyType)
	}

	// List positions
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/positions", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var positions []models.Position
	json.Unmarshal(w.Body.Bytes(), &positions)
	if len(positions) != 1 {
		t.Fatalf("Expected 1 position, got %d", len(positions))
	}

	// Delete position
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/api/positions/1", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
}

func TestPositionDateOnlyFormat(t *testing.T) {
	setupTestDB(t)
	router := setupRouter()

	// Set up person and account
	body, _ := json.Marshal(map[string]string{"name": "Dave"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/persons", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	accountData := map[string]interface{}{
		"name":     "Sparkonto",
		"balance":  0,
		"currency": "EUR",
		"ownerIds": []uint{1},
	}
	body, _ = json.Marshal(accountData)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/accounts", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	// Create position with date-only startDate (as sent by the frontend <input type="date">)
	positionData := map[string]interface{}{
		"name":            "Miete",
		"type":            "expense",
		"amount":          800.00,
		"accountId":       uint(1),
		"frequencyType":   "monthly",
		"interval":        1,
		"businessDayRule": "exact",
		"startDate":       "2026-03-06",
	}
	body, _ = json.Marshal(positionData)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/positions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201 for date-only startDate, got %d: %s", w.Code, w.Body.String())
	}

	var position models.Position
	json.Unmarshal(w.Body.Bytes(), &position)
	if position.Name != "Miete" {
		t.Fatalf("Expected name 'Miete', got '%s'", position.Name)
	}
	if position.StartDate.IsZero() {
		t.Fatal("Expected non-zero StartDate")
	}
}

func TestPositionValidation(t *testing.T) {
	setupTestDB(t)
	router := setupRouter()

	// Try to create position without required fields
	positionData := map[string]interface{}{
		"name": "Bad Position",
	}
	body, _ := json.Marshal(positionData)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/positions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 for invalid position, got %d: %s", w.Code, w.Body.String())
	}
}

func TestProjectionBasic(t *testing.T) {
	setupTestDB(t)
	router := setupRouter()

	// Create person
	body, _ := json.Marshal(map[string]string{"name": "Eve"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/persons", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d", w.Code)
	}

	// Create account with balance 1000
	accountData := map[string]interface{}{
		"name":     "Girokonto",
		"balance":  1000.0,
		"currency": "EUR",
		"ownerIds": []uint{1},
	}
	body, _ = json.Marshal(accountData)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/accounts", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Create monthly income of 3000 starting 2026-01-01
	dayOfMonth := 1
	accountID := uint(1)
	posData := map[string]interface{}{
		"name":            "Gehalt",
		"type":            "income",
		"amount":          3000.00,
		"accountId":       accountID,
		"frequencyType":   "monthly",
		"interval":        1,
		"dayOfMonth":      dayOfMonth,
		"businessDayRule": "exact",
		"startDate":       "2026-01-01",
	}
	body, _ = json.Marshal(posData)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/positions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Create monthly expense of 800 starting 2026-01-01
	posData2 := map[string]interface{}{
		"name":            "Miete",
		"type":            "expense",
		"amount":          800.00,
		"accountId":       accountID,
		"frequencyType":   "monthly",
		"interval":        1,
		"dayOfMonth":      dayOfMonth,
		"businessDayRule": "exact",
		"startDate":       "2026-01-01",
	}
	body, _ = json.Marshal(posData2)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/positions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Get projection for 3 months starting 2026-01-01
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/projection?months=3&startDate=2026-01-01&granularity=monthly", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result struct {
		Accounts []struct {
			ID         uint   `json:"id"`
			Name       string `json:"name"`
			DataPoints []struct {
				Date    string  `json:"date"`
				Balance float64 `json:"balance"`
			} `json:"dataPoints"`
		} `json:"accounts"`
		Totals []struct {
			Date    string  `json:"date"`
			Balance float64 `json:"balance"`
		} `json:"totals"`
	}
	json.Unmarshal(w.Body.Bytes(), &result)

	if len(result.Accounts) != 1 {
		t.Fatalf("Expected 1 account, got %d", len(result.Accounts))
	}

	if len(result.Totals) < 2 {
		t.Fatalf("Expected at least 2 data points, got %d", len(result.Totals))
	}

	// Initial balance is 1000. On 2026-01-01: +3000 -800 = 1000+2200 = 3200
	firstPoint := result.Accounts[0].DataPoints[0]
	if firstPoint.Date != "2026-01-01" {
		t.Fatalf("Expected first date '2026-01-01', got '%s'", firstPoint.Date)
	}
	if firstPoint.Balance != 3200.0 {
		t.Fatalf("Expected first balance 3200.00, got %.2f", firstPoint.Balance)
	}

	// Second month (2026-02-01): 3200 + 3000 - 800 = 5400
	secondPoint := result.Accounts[0].DataPoints[1]
	if secondPoint.Balance != 5400.0 {
		t.Fatalf("Expected second balance 5400.00, got %.2f", secondPoint.Balance)
	}
}

func TestProjectionEmpty(t *testing.T) {
	setupTestDB(t)
	router := setupRouter()

	// Get projection with no data
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/projection?months=1&startDate=2026-01-01&granularity=monthly", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result struct {
		Accounts []interface{} `json:"accounts"`
		Totals   []interface{} `json:"totals"`
	}
	json.Unmarshal(w.Body.Bytes(), &result)

	if len(result.Accounts) != 0 {
		t.Fatalf("Expected 0 accounts, got %d", len(result.Accounts))
	}
}

func TestCreatePersonValidation(t *testing.T) {
	setupTestDB(t)
	router := setupRouter()

	// Try to create person without name
	body, _ := json.Marshal(map[string]string{})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/persons", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 for missing name, got %d", w.Code)
	}
}

func TestSettingsCRUD(t *testing.T) {
	setupTestDB(t)
	router := setupRouter()

	// GET settings (initially empty)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/settings", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var settings map[string]string
	json.Unmarshal(w.Body.Bytes(), &settings)
	if len(settings) != 0 {
		t.Fatalf("Expected 0 settings, got %d", len(settings))
	}

	// PUT settings
	body, _ := json.Marshal(map[string]string{
		"banking_bridge_url": "http://bridge.local:9090",
	})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", "/api/settings", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	json.Unmarshal(w.Body.Bytes(), &settings)
	if settings["banking_bridge_url"] != "http://bridge.local:9090" {
		t.Fatalf("Expected banking_bridge_url 'http://bridge.local:9090', got '%s'", settings["banking_bridge_url"])
	}

	// GET settings again to verify persistence
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/settings", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	json.Unmarshal(w.Body.Bytes(), &settings)
	if settings["banking_bridge_url"] != "http://bridge.local:9090" {
		t.Fatalf("Expected persisted banking_bridge_url, got '%s'", settings["banking_bridge_url"])
	}

	// Update setting to new value
	body, _ = json.Marshal(map[string]string{
		"banking_bridge_url": "http://new-bridge:8080",
	})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", "/api/settings", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	json.Unmarshal(w.Body.Bytes(), &settings)
	if settings["banking_bridge_url"] != "http://new-bridge:8080" {
		t.Fatalf("Expected updated URL, got '%s'", settings["banking_bridge_url"])
	}
}

func TestExpenseWithTargetAccount(t *testing.T) {
	setupTestDB(t)
	router := setupRouter()

	// Create person
	body, _ := json.Marshal(map[string]string{"name": "Frank"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/persons", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d", w.Code)
	}

	// Create two accounts: Private (1000) and Household (500)
	for _, acc := range []map[string]interface{}{
		{"name": "Privatkonto", "balance": 1000.0, "currency": "EUR", "ownerIds": []uint{1}},
		{"name": "Haushaltskonto", "balance": 500.0, "currency": "EUR", "ownerIds": []uint{1}},
	} {
		body, _ = json.Marshal(acc)
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "/api/accounts", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("Expected 201, got %d: %s", w.Code, w.Body.String())
		}
	}

	// Create expense from Privatkonto (accountId=1) with target Haushaltskonto (targetAccountId=2)
	privateID := uint(1)
	householdID := uint(2)
	dayOfMonth := 15
	posData := map[string]interface{}{
		"name":            "Haushaltsgeld",
		"type":            "expense",
		"amount":          200.00,
		"accountId":       privateID,
		"targetAccountId": householdID,
		"frequencyType":   "monthly",
		"interval":        1,
		"dayOfMonth":      dayOfMonth,
		"businessDayRule": "exact",
		"startDate":       "2026-01-01",
	}
	body, _ = json.Marshal(posData)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/positions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Verify the position was created with targetAccountId
	var position models.Position
	json.Unmarshal(w.Body.Bytes(), &position)
	if position.TargetAccountID == nil || *position.TargetAccountID != householdID {
		t.Fatalf("Expected targetAccountId %d, got %v", householdID, position.TargetAccountID)
	}

	// Get projection for 2 months
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/projection?months=2&startDate=2026-01-01&granularity=monthly", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result struct {
		Accounts []struct {
			ID         uint   `json:"id"`
			Name       string `json:"name"`
			DataPoints []struct {
				Date    string  `json:"date"`
				Balance float64 `json:"balance"`
			} `json:"dataPoints"`
		} `json:"accounts"`
		Totals []struct {
			Date    string  `json:"date"`
			Balance float64 `json:"balance"`
		} `json:"totals"`
	}
	json.Unmarshal(w.Body.Bytes(), &result)

	if len(result.Accounts) != 2 {
		t.Fatalf("Expected 2 accounts, got %d", len(result.Accounts))
	}

	// On 2026-01-15: Privatkonto 1000-200=800, Haushaltskonto 500+200=700
	// On 2026-02-15: Privatkonto 800-200=600, Haushaltskonto 700+200=900
	// At 2026-02-01 (monthly granularity): only first occurrence applied
	// Privatkonto: 800, Haushaltskonto: 700
	privateAcc := result.Accounts[0]
	householdAcc := result.Accounts[1]

	// After first month (first occurrence on Jan 15)
	if privateAcc.DataPoints[1].Balance != 800.0 {
		t.Fatalf("Expected Privatkonto balance 800.00 after first month, got %.2f", privateAcc.DataPoints[1].Balance)
	}
	if householdAcc.DataPoints[1].Balance != 700.0 {
		t.Fatalf("Expected Haushaltskonto balance 700.00 after first month, got %.2f", householdAcc.DataPoints[1].Balance)
	}

	// Total should remain constant (no money is created or destroyed)
	if result.Totals[0].Balance != 1500.0 {
		t.Fatalf("Expected total balance 1500.00, got %.2f", result.Totals[0].Balance)
	}
	if result.Totals[1].Balance != 1500.0 {
		t.Fatalf("Expected total balance 1500.00 after first month, got %.2f", result.Totals[1].Balance)
	}
}

func TestIncomeWithSourceAccount(t *testing.T) {
	setupTestDB(t)
	router := setupRouter()

	// Create person
	body, _ := json.Marshal(map[string]string{"name": "Grace"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/persons", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d", w.Code)
	}

	// Create two accounts: Savings (2000) and Checking (500)
	for _, acc := range []map[string]interface{}{
		{"name": "Sparkonto", "balance": 2000.0, "currency": "EUR", "ownerIds": []uint{1}},
		{"name": "Girokonto", "balance": 500.0, "currency": "EUR", "ownerIds": []uint{1}},
	} {
		body, _ = json.Marshal(acc)
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "/api/accounts", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("Expected 201, got %d: %s", w.Code, w.Body.String())
		}
	}

	// Create income to Girokonto (accountId=2) from Sparkonto (sourceAccountId=1)
	savingsID := uint(1)
	checkingID := uint(2)
	dayOfMonth := 1
	posData := map[string]interface{}{
		"name":            "Sparkonto Entnahme",
		"type":            "income",
		"amount":          300.00,
		"accountId":       checkingID,
		"sourceAccountId": savingsID,
		"frequencyType":   "monthly",
		"interval":        1,
		"dayOfMonth":      dayOfMonth,
		"businessDayRule": "exact",
		"startDate":       "2026-01-01",
	}
	body, _ = json.Marshal(posData)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/positions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Get projection
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/projection?months=2&startDate=2026-01-01&granularity=monthly", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result struct {
		Accounts []struct {
			ID         uint   `json:"id"`
			Name       string `json:"name"`
			DataPoints []struct {
				Date    string  `json:"date"`
				Balance float64 `json:"balance"`
			} `json:"dataPoints"`
		} `json:"accounts"`
		Totals []struct {
			Date    string  `json:"date"`
			Balance float64 `json:"balance"`
		} `json:"totals"`
	}
	json.Unmarshal(w.Body.Bytes(), &result)

	// On 2026-01-01: Sparkonto 2000-300=1700, Girokonto 500+300=800
	savingsAcc := result.Accounts[0]
	checkingAcc := result.Accounts[1]

	if savingsAcc.DataPoints[0].Balance != 1700.0 {
		t.Fatalf("Expected Sparkonto balance 1700.00, got %.2f", savingsAcc.DataPoints[0].Balance)
	}
	if checkingAcc.DataPoints[0].Balance != 800.0 {
		t.Fatalf("Expected Girokonto balance 800.00, got %.2f", checkingAcc.DataPoints[0].Balance)
	}

	// Total should remain constant
	if result.Totals[0].Balance != 2500.0 {
		t.Fatalf("Expected total balance 2500.00, got %.2f", result.Totals[0].Balance)
	}
}

func TestSeparatorCRUD(t *testing.T) {
	setupTestDB(t)
	router := setupRouter()

	// Create separator
	body, _ := json.Marshal(map[string]string{"name": "Fixkosten"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/position-separators", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var sep models.PositionSeparator
	json.Unmarshal(w.Body.Bytes(), &sep)
	if sep.Name != "Fixkosten" {
		t.Fatalf("Expected name 'Fixkosten', got '%s'", sep.Name)
	}
	if sep.ID == 0 {
		t.Fatal("Expected non-zero ID")
	}

	// List separators
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/position-separators", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
	var seps []models.PositionSeparator
	json.Unmarshal(w.Body.Bytes(), &seps)
	if len(seps) != 1 {
		t.Fatalf("Expected 1 separator, got %d", len(seps))
	}

	// Update separator
	body, _ = json.Marshal(map[string]string{"name": "Variable Kosten"})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", "/api/position-separators/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}
	json.Unmarshal(w.Body.Bytes(), &sep)
	if sep.Name != "Variable Kosten" {
		t.Fatalf("Expected name 'Variable Kosten', got '%s'", sep.Name)
	}

	// Delete separator
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/api/position-separators/1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	// Verify deleted
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/position-separators", nil)
	router.ServeHTTP(w, req)
	json.Unmarshal(w.Body.Bytes(), &seps)
	if len(seps) != 0 {
		t.Fatalf("Expected 0 separators after delete, got %d", len(seps))
	}
}

func TestReorderPositions(t *testing.T) {
	setupTestDB(t)
	router := setupRouter()

	// Create account first
	body, _ := json.Marshal(map[string]interface{}{
		"name": "Girokonto", "balance": 1000, "currency": "EUR",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/accounts", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	// Create two positions
	for _, name := range []string{"Miete", "Strom"} {
		body, _ = json.Marshal(map[string]interface{}{
			"name": name, "type": "expense", "amount": 100,
			"accountId": 1, "frequencyType": "monthly",
			"startDate": "2025-01-01",
		})
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "/api/positions", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("Expected 201, got %d: %s", w.Code, w.Body.String())
		}
	}

	// Create a separator
	body, _ = json.Marshal(map[string]string{"name": "Fixkosten"})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/position-separators", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Reorder: separator first, then Strom (id=2), then Miete (id=1)
	reorder := []map[string]interface{}{
		{"type": "separator", "id": 1},
		{"type": "position", "id": 2},
		{"type": "position", "id": 1},
	}
	body, _ = json.Marshal(reorder)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", "/api/positions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify order: positions should come back in order Strom, Miete
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/positions", nil)
	router.ServeHTTP(w, req)

	var positions []models.Position
	json.Unmarshal(w.Body.Bytes(), &positions)
	if len(positions) != 2 {
		t.Fatalf("Expected 2 positions, got %d", len(positions))
	}
	if positions[0].Name != "Strom" {
		t.Fatalf("Expected first position 'Strom', got '%s'", positions[0].Name)
	}
	if positions[1].Name != "Miete" {
		t.Fatalf("Expected second position 'Miete', got '%s'", positions[1].Name)
	}

	// Verify separator order
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/position-separators", nil)
	router.ServeHTTP(w, req)

	var seps []models.PositionSeparator
	json.Unmarshal(w.Body.Bytes(), &seps)
	if len(seps) != 1 {
		t.Fatalf("Expected 1 separator, got %d", len(seps))
	}
	if seps[0].SortOrder != 0 {
		t.Fatalf("Expected separator sort order 0, got %d", seps[0].SortOrder)
	}
}
