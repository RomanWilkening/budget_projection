package bankingbridge

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// BridgeAccount represents an account from the Banking Bridge API.
type BridgeAccount struct {
	ID            int     `json:"id"`
	Name          string  `json:"name"`
	AccountNumber string  `json:"account_number"`
	IBAN          string  `json:"iban"`
	BIC           string  `json:"bic"`
	AccountType   string  `json:"account_type"`
	Bank          string  `json:"bank"`
	BankCode      string  `json:"bank_code"`
	Balance       float64 `json:"balance"`
	Currency      string  `json:"currency"`
	LastUpdate    string  `json:"last_update"`
}

// BridgeTransaction represents a transaction from the Banking Bridge API.
type BridgeTransaction struct {
	BookingDate string  `json:"booking_date"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	IBAN        string  `json:"iban"`
	BookingText string  `json:"booking_text"`
}

type accountsResponse struct {
	Success  bool            `json:"success"`
	Count    int             `json:"count"`
	Accounts []BridgeAccount `json:"accounts"`
	Message  string          `json:"message"`
}

type singleAccountResponse struct {
	Success bool          `json:"success"`
	Account BridgeAccount `json:"account"`
	Message string        `json:"message"`
}

type transactionsResponse struct {
	Success      bool                `json:"success"`
	Count        int                 `json:"count"`
	Transactions []BridgeTransaction `json:"transactions"`
	Message      string              `json:"message"`
}

// Client communicates with the Banking Bridge API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new Banking Bridge client.
// The URL can be set later via SetBaseURL or loaded from database settings.
func NewClient() *Client {
	baseURL := os.Getenv("BANKING_BRIDGE_URL")
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewClientWithURL creates a new Banking Bridge client with a specific base URL.
func NewClientWithURL(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetBaseURL returns the configured base URL.
func (c *Client) GetBaseURL() string {
	return c.baseURL
}

// SetBaseURL updates the base URL at runtime.
func (c *Client) SetBaseURL(url string) {
	c.baseURL = url
}

// IsConfigured returns true if the Banking Bridge URL has been set.
func (c *Client) IsConfigured() bool {
	return c.baseURL != ""
}

// GetAccounts fetches all accounts from the Banking Bridge.
func (c *Client) GetAccounts() ([]BridgeAccount, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/v1/accounts")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Banking Bridge: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result accountsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("Banking Bridge error: %s", result.Message)
	}

	return result.Accounts, nil
}

// GetAccount fetches a single account from the Banking Bridge.
func (c *Client) GetAccount(accountID int) (*BridgeAccount, error) {
	resp, err := c.httpClient.Get(fmt.Sprintf("%s/api/v1/accounts/%d", c.baseURL, accountID))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Banking Bridge: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result singleAccountResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("Banking Bridge error: %s", result.Message)
	}

	return &result.Account, nil
}

// GetTransactions fetches transactions for an account in a date range.
func (c *Client) GetTransactions(accountID int, from, to string) ([]BridgeTransaction, error) {
	url := fmt.Sprintf("%s/api/v1/accounts/%d/transactions?from=%s&to=%s", c.baseURL, accountID, from, to)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Banking Bridge: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result transactionsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("Banking Bridge error: %s", result.Message)
	}

	return result.Transactions, nil
}

// CheckConnection tests the connection to the Banking Bridge by fetching accounts.
func (c *Client) CheckConnection() error {
	_, err := c.GetAccounts()
	return err
}
