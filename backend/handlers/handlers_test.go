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
		}
		positions := api.Group("/positions")
		{
			positions.GET("", handlers.ListPositions)
			positions.GET("/:id", handlers.GetPosition)
			positions.POST("", handlers.CreatePosition)
			positions.PUT("/:id", handlers.UpdatePosition)
			positions.DELETE("/:id", handlers.DeletePosition)
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
