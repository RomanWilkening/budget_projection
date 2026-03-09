package handlers

import (
	"net/http"

	"github.com/RomanWilkening/budget_projection/backend/database"
	"github.com/RomanWilkening/budget_projection/backend/models"
	"github.com/gin-gonic/gin"
)

// ListDepots returns all depots with their linked accounts.
func ListDepots(c *gin.Context) {
	var depots []models.Depot
	if err := database.DB.Preload("Accounts").Find(&depots).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, depots)
}

// GetDepot returns a single depot by ID.
func GetDepot(c *gin.Context) {
	id := c.Param("id")
	var depot models.Depot
	if err := database.DB.Preload("Accounts").First(&depot, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Depot not found"})
		return
	}
	c.JSON(http.StatusOK, depot)
}

type createDepotInput struct {
	Name         string  `json:"name" binding:"required"`
	InterestRate float64 `json:"interestRate"`
	AccountIDs   []uint  `json:"accountIds"`
}

// CreateDepot creates a new virtual depot.
func CreateDepot(c *gin.Context) {
	var input createDepotInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	depot := models.Depot{
		Name:         input.Name,
		InterestRate: input.InterestRate,
	}

	if err := database.DB.Create(&depot).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Assign accounts if provided
	if len(input.AccountIDs) > 0 {
		var accounts []models.Account
		if err := database.DB.Find(&accounts, input.AccountIDs).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid account IDs"})
			return
		}
		if err := database.DB.Model(&depot).Association("Accounts").Replace(accounts); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// Reload with accounts
	database.DB.Preload("Accounts").First(&depot, depot.ID)
	c.JSON(http.StatusCreated, depot)
}

type updateDepotInput struct {
	Name         string  `json:"name" binding:"required"`
	InterestRate float64 `json:"interestRate"`
	AccountIDs   []uint  `json:"accountIds"`
}

// UpdateDepot updates an existing depot.
func UpdateDepot(c *gin.Context) {
	id := c.Param("id")
	var depot models.Depot
	if err := database.DB.First(&depot, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Depot not found"})
		return
	}

	var input updateDepotInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	depot.Name = input.Name
	depot.InterestRate = input.InterestRate

	if err := database.DB.Save(&depot).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update accounts
	if input.AccountIDs != nil {
		var accounts []models.Account
		if len(input.AccountIDs) > 0 {
			if err := database.DB.Find(&accounts, input.AccountIDs).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid account IDs"})
				return
			}
		}
		if err := database.DB.Model(&depot).Association("Accounts").Replace(accounts); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	database.DB.Preload("Accounts").First(&depot, depot.ID)
	c.JSON(http.StatusOK, depot)
}

// DeleteDepot deletes a depot.
func DeleteDepot(c *gin.Context) {
	id := c.Param("id")
	if err := database.DB.Delete(&models.Depot{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Depot deleted"})
}
