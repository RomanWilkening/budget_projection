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
