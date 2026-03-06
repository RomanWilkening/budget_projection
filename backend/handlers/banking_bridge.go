package handlers

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/RomanWilkening/budget_projection/backend/bankingbridge"
	"github.com/RomanWilkening/budget_projection/backend/database"
	"github.com/RomanWilkening/budget_projection/backend/models"
	"github.com/gin-gonic/gin"
)

var bridgeClient = bankingbridge.NewClient()

// GetBankingBridgeStatus returns the Banking Bridge connection status.
func GetBankingBridgeStatus(c *gin.Context) {
	configured := bridgeClient.IsConfigured()
	status := gin.H{
		"configured": configured,
		"url":        bridgeClient.GetBaseURL(),
	}

	if configured {
		err := bridgeClient.CheckConnection()
		if err != nil {
			status["connected"] = false
			status["error"] = err.Error()
		} else {
			status["connected"] = true
		}
	} else {
		status["connected"] = false
	}

	c.JSON(http.StatusOK, status)
}

// ListBankingBridgeAccounts lists all accounts from the Banking Bridge.
func ListBankingBridgeAccounts(c *gin.Context) {
	accounts, err := bridgeClient.GetAccounts()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, accounts)
}

// LinkBankingBridgeAccount links a local account to a Banking Bridge account.
func LinkBankingBridgeAccount(c *gin.Context) {
	id := c.Param("id")
	var account models.Account
	if err := database.DB.First(&account, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return
	}

	var input struct {
		BankingBridgeAccountID *int `json:"bankingBridgeAccountId"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate that the Banking Bridge account exists (if linking, not unlinking)
	if input.BankingBridgeAccountID != nil {
		_, err := bridgeClient.GetAccount(*input.BankingBridgeAccountID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Banking Bridge account not found: " + err.Error()})
			return
		}
	}

	account.BankingBridgeAccountID = input.BankingBridgeAccountID
	if err := database.DB.Save(&account).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	database.DB.Preload("Owners").First(&account, account.ID)
	c.JSON(http.StatusOK, account)
}

// SyncAccountBalance syncs the balance of a single linked account from the Banking Bridge.
func SyncAccountBalance(c *gin.Context) {
	id := c.Param("id")
	var account models.Account
	if err := database.DB.First(&account, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return
	}

	if account.BankingBridgeAccountID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Account is not linked to Banking Bridge"})
		return
	}

	bridgeAccount, err := bridgeClient.GetAccount(*account.BankingBridgeAccountID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	oldBalance := account.Balance
	account.Balance = bridgeAccount.Balance
	if err := database.DB.Save(&account).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	database.DB.Preload("Owners").First(&account, account.ID)
	c.JSON(http.StatusOK, gin.H{
		"account":     account,
		"oldBalance":  oldBalance,
		"newBalance":  bridgeAccount.Balance,
		"lastUpdate":  bridgeAccount.LastUpdate,
	})
}

// SyncAllBalances syncs balances of all linked accounts from the Banking Bridge.
func SyncAllBalances(c *gin.Context) {
	var accounts []models.Account
	if err := database.DB.Where("banking_bridge_account_id IS NOT NULL").Find(&accounts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type syncResult struct {
		AccountID   uint    `json:"accountId"`
		AccountName string  `json:"accountName"`
		OldBalance  float64 `json:"oldBalance"`
		NewBalance  float64 `json:"newBalance"`
		Error       string  `json:"error,omitempty"`
	}

	var results []syncResult
	successCount := 0

	for _, account := range accounts {
		result := syncResult{
			AccountID:   account.ID,
			AccountName: account.Name,
			OldBalance:  account.Balance,
		}

		bridgeAccount, err := bridgeClient.GetAccount(*account.BankingBridgeAccountID)
		if err != nil {
			result.Error = err.Error()
			results = append(results, result)
			continue
		}

		account.Balance = bridgeAccount.Balance
		if err := database.DB.Save(&account).Error; err != nil {
			result.Error = err.Error()
			results = append(results, result)
			continue
		}

		result.NewBalance = bridgeAccount.Balance
		results = append(results, result)
		successCount++
	}

	c.JSON(http.StatusOK, gin.H{
		"synced":  successCount,
		"total":   len(accounts),
		"results": results,
	})
}

// AnalyzeRecurringTransactions analyzes transactions from the Banking Bridge to find recurring patterns.
func AnalyzeRecurringTransactions(c *gin.Context) {
	id := c.Param("id")
	var account models.Account
	if err := database.DB.First(&account, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return
	}

	if account.BankingBridgeAccountID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Account is not linked to Banking Bridge"})
		return
	}

	// Parse months parameter (default: 12)
	months := 12
	if m := c.Query("months"); m != "" {
		if parsed, err := strconv.Atoi(m); err == nil && parsed > 0 && parsed <= 24 {
			months = parsed
		}
	}

	// Calculate date range
	now := time.Now()
	to := now.Format("2006-01-02")
	from := now.AddDate(0, -months, 0).Format("2006-01-02")

	// Fetch transactions from Banking Bridge
	transactions, err := bridgeClient.GetTransactions(*account.BankingBridgeAccountID, from, to)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	// Analyze for recurring patterns
	patterns := bankingbridge.AnalyzeRecurringTransactions(transactions)

	// Load existing positions for this account to detect matches
	var existingPositions []models.Position
	database.DB.Where("account_id = ? OR source_account_id = ? OR target_account_id = ?",
		account.ID, account.ID, account.ID).
		Find(&existingPositions)

	// Annotate patterns with matching info
	type AnnotatedPattern struct {
		bankingbridge.RecurringPattern
		MatchingPositionID   *uint  `json:"matchingPositionId,omitempty"`
		MatchingPositionName string `json:"matchingPositionName,omitempty"`
		SuggestedAction      string `json:"suggestedAction"` // "create", "update", "none"
	}

	var annotated []AnnotatedPattern
	for _, pattern := range patterns {
		ap := AnnotatedPattern{
			RecurringPattern: pattern,
			SuggestedAction:  "create",
		}

		// Try to match with existing positions
		for _, pos := range existingPositions {
			nameMatch := strings.Contains(strings.ToLower(pos.Name), strings.ToLower(pattern.Name)) ||
				strings.Contains(strings.ToLower(pattern.Name), strings.ToLower(pos.Name))

			if nameMatch {
				ap.MatchingPositionID = &pos.ID
				ap.MatchingPositionName = pos.Name

				// Check if amount differs significantly
				amountDiff := math.Abs(pos.Amount - pattern.AverageAmount)
				if amountDiff > math.Abs(pos.Amount)*0.1 { // More than 10% difference
					ap.SuggestedAction = "update"
				} else {
					ap.SuggestedAction = "none"
				}
				break
			}
		}

		annotated = append(annotated, ap)
	}

	c.JSON(http.StatusOK, gin.H{
		"accountId":         account.ID,
		"accountName":       account.Name,
		"analyzedFrom":      from,
		"analyzedTo":        to,
		"transactionCount":  len(transactions),
		"patterns":          annotated,
	})
}
