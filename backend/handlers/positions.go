package handlers

import (
	"fmt"
	"net/http"

	"github.com/RomanWilkening/budget_projection/backend/database"
	"github.com/RomanWilkening/budget_projection/backend/models"
	"github.com/gin-gonic/gin"
)

// ListPositions returns all positions.
func ListPositions(c *gin.Context) {
	var positions []models.Position
	if err := database.DB.
		Preload("Account").
		Preload("SourceAccount").
		Preload("TargetAccount").
		Order("sort_order ASC, id ASC").
		Find(&positions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, positions)
}

// GetPosition returns a single position by ID.
func GetPosition(c *gin.Context) {
	id := c.Param("id")
	var position models.Position
	if err := database.DB.
		Preload("Account").
		Preload("SourceAccount").
		Preload("TargetAccount").
		First(&position, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Position not found"})
		return
	}
	c.JSON(http.StatusOK, position)
}

// CreatePosition creates a new position.
func CreatePosition(c *gin.Context) {
	var position models.Position
	if err := c.ShouldBindJSON(&position); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := validatePosition(&position); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set sort order to the max + 1 so it appears at the end
	position.SortOrder = nextSortOrder()

	if err := database.DB.Create(&position).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Reload with associations
	database.DB.
		Preload("Account").
		Preload("SourceAccount").
		Preload("TargetAccount").
		First(&position, position.ID)

	c.JSON(http.StatusCreated, position)
}

// UpdatePosition updates a position.
func UpdatePosition(c *gin.Context) {
	id := c.Param("id")
	var position models.Position
	if err := database.DB.First(&position, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Position not found"})
		return
	}

	if err := c.ShouldBindJSON(&position); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := validatePosition(&position); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Save(&position).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	database.DB.
		Preload("Account").
		Preload("SourceAccount").
		Preload("TargetAccount").
		First(&position, position.ID)

	c.JSON(http.StatusOK, position)
}

// DeletePosition deletes a position.
func DeletePosition(c *gin.Context) {
	id := c.Param("id")
	if err := database.DB.Delete(&models.Position{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Position deleted"})
}

func validatePosition(p *models.Position) error {
	if p.Name == "" {
		return errorf("name is required")
	}
	if p.Amount <= 0 {
		return errorf("amount must be positive")
	}

	switch p.Type {
	case models.PositionIncome:
		if p.AccountID == nil {
			return errorf("accountId is required for income positions")
		}
		// sourceAccountId is optional: if set, the source account is debited
	case models.PositionExpense:
		if p.AccountID == nil {
			return errorf("accountId is required for expense positions")
		}
		// targetAccountId is optional: if set, the target account is credited
	case models.PositionTransfer:
		if p.SourceAccountID == nil || p.TargetAccountID == nil {
			return errorf("sourceAccountId and targetAccountId are required for transfer positions")
		}
	default:
		return errorf("type must be 'income', 'expense', or 'transfer'")
	}

	switch p.FrequencyType {
	case models.FrequencyDaily, models.FrequencyWeekly, models.FrequencyBiweekly,
		models.FrequencyMonthly, models.FrequencyQuarterly,
		models.FrequencySemiAnnually, models.FrequencyAnnually:
		// valid
	default:
		return errorf("invalid frequencyType: %s", p.FrequencyType)
	}

	if p.Interval < 1 {
		p.Interval = 1
	}

	return nil
}

// ReorderItem represents a single item in the reorder request.
type ReorderItem struct {
	Type string `json:"type"` // "position" or "separator"
	ID   uint   `json:"id"`
}

// ReorderPositions updates the sort order of positions and separators.
func ReorderPositions(c *gin.Context) {
	var items []ReorderItem
	if err := c.ShouldBindJSON(&items); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx := database.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": tx.Error.Error()})
		return
	}
	for i, item := range items {
		switch item.Type {
		case "position":
			if err := tx.Model(&models.Position{}).Where("id = ?", item.ID).Update("sort_order", i).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		case "separator":
			if err := tx.Model(&models.PositionSeparator{}).Where("id = ?", item.ID).Update("sort_order", i).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		default:
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid type: %s", item.Type)})
			return
		}
	}
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Order updated"})
}

type validationError struct {
	msg string
}

func (e *validationError) Error() string {
	return e.msg
}

func errorf(format string, args ...interface{}) error {
	return &validationError{msg: fmt.Sprintf(format, args...)}
}

// nextSortOrder returns the next available sort order for positions and separators.
func nextSortOrder() int {
	var maxOrder int
	database.DB.Raw(`
		SELECT COALESCE(MAX(sort_order), -1) FROM (
			SELECT sort_order FROM positions WHERE deleted_at IS NULL
			UNION ALL
			SELECT sort_order FROM position_separators WHERE deleted_at IS NULL
		)
	`).Scan(&maxOrder)
	return maxOrder + 1
}
