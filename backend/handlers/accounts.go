package handlers

import (
	"net/http"

	"github.com/RomanWilkening/budget_projection/backend/database"
	"github.com/RomanWilkening/budget_projection/backend/models"
	"github.com/gin-gonic/gin"
)

// ListAccounts returns all accounts with their owners.
func ListAccounts(c *gin.Context) {
	var accounts []models.Account
	if err := database.DB.Preload("Owners").Order("sort_order ASC, id ASC").Find(&accounts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, accounts)
}

// GetAccount returns a single account by ID.
func GetAccount(c *gin.Context) {
	id := c.Param("id")
	var account models.Account
	if err := database.DB.Preload("Owners").First(&account, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return
	}
	c.JSON(http.StatusOK, account)
}

type createAccountInput struct {
	Name                   string `json:"name" binding:"required"`
	Balance                float64 `json:"balance"`
	Currency               string `json:"currency"`
	ShowInProjection       *bool  `json:"showInProjection"`
	OwnerIDs               []uint `json:"ownerIds"`
	BankingBridgeAccountID *int   `json:"bankingBridgeAccountId"`
}

// CreateAccount creates a new account with optional owners.
func CreateAccount(c *gin.Context) {
	var input createAccountInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	showInProjection := true
	account := models.Account{
		Name:                   input.Name,
		Balance:                input.Balance,
		Currency:               input.Currency,
		ShowInProjection:       &showInProjection,
		BankingBridgeAccountID: input.BankingBridgeAccountID,
	}
	if input.ShowInProjection != nil {
		account.ShowInProjection = input.ShowInProjection
	}
	if account.Currency == "" {
		account.Currency = "EUR"
	}

	if err := database.DB.Create(&account).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Assign owners if provided
	if len(input.OwnerIDs) > 0 {
		var owners []models.Person
		if err := database.DB.Find(&owners, input.OwnerIDs).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid owner IDs"})
			return
		}
		if err := database.DB.Model(&account).Association("Owners").Replace(owners); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// Reload with owners
	database.DB.Preload("Owners").First(&account, account.ID)
	c.JSON(http.StatusCreated, account)
}

type updateAccountInput struct {
	Name                   string  `json:"name" binding:"required"`
	Balance                float64 `json:"balance"`
	Currency               string  `json:"currency"`
	ShowInProjection       *bool   `json:"showInProjection"`
	OwnerIDs               []uint  `json:"ownerIds"`
	BankingBridgeAccountID *int    `json:"bankingBridgeAccountId"`
}

// UpdateAccount updates an account.
func UpdateAccount(c *gin.Context) {
	id := c.Param("id")
	var account models.Account
	if err := database.DB.First(&account, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return
	}

	var input updateAccountInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	account.Name = input.Name
	account.Balance = input.Balance
	if input.Currency != "" {
		account.Currency = input.Currency
	}
	if input.ShowInProjection != nil {
		account.ShowInProjection = input.ShowInProjection
	}
	account.BankingBridgeAccountID = input.BankingBridgeAccountID

	if err := database.DB.Save(&account).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update owners
	if input.OwnerIDs != nil {
		var owners []models.Person
		if len(input.OwnerIDs) > 0 {
			if err := database.DB.Find(&owners, input.OwnerIDs).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid owner IDs"})
				return
			}
		}
		if err := database.DB.Model(&account).Association("Owners").Replace(owners); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	database.DB.Preload("Owners").First(&account, account.ID)
	c.JSON(http.StatusOK, account)
}

// DeleteAccount deletes an account.
func DeleteAccount(c *gin.Context) {
	id := c.Param("id")
	if err := database.DB.Delete(&models.Account{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Account deleted"})
}

// AddAccountOwner adds a person as owner to an account.
func AddAccountOwner(c *gin.Context) {
	accountID := c.Param("id")
	var account models.Account
	if err := database.DB.First(&account, accountID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return
	}

	var input struct {
		PersonID uint `json:"personId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var person models.Person
	if err := database.DB.First(&person, input.PersonID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Person not found"})
		return
	}

	if err := database.DB.Model(&account).Association("Owners").Append(&person); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	database.DB.Preload("Owners").First(&account, account.ID)
	c.JSON(http.StatusOK, account)
}

// RemoveAccountOwner removes a person as owner from an account.
func RemoveAccountOwner(c *gin.Context) {
	accountID := c.Param("id")
	personID := c.Param("personId")

	var account models.Account
	if err := database.DB.First(&account, accountID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return
	}

	var person models.Person
	if err := database.DB.First(&person, personID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Person not found"})
		return
	}

	if err := database.DB.Model(&account).Association("Owners").Delete(&person); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	database.DB.Preload("Owners").First(&account, account.ID)
	c.JSON(http.StatusOK, account)
}

// ReorderAccounts updates the sort order of accounts.
func ReorderAccounts(c *gin.Context) {
	var ids []uint
	if err := c.ShouldBindJSON(&ids); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx := database.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": tx.Error.Error()})
		return
	}
	for i, id := range ids {
		if err := tx.Model(&models.Account{}).Where("id = ?", id).Update("sort_order", i).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Order updated"})
}
