package handlers

import (
	"net/http"

	"github.com/RomanWilkening/budget_projection/backend/database"
	"github.com/RomanWilkening/budget_projection/backend/models"
	"github.com/gin-gonic/gin"
)

// ListSeparators returns all position separators.
func ListSeparators(c *gin.Context) {
	var separators []models.PositionSeparator
	if err := database.DB.Order("sort_order ASC, id ASC").Find(&separators).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, separators)
}

// CreateSeparator creates a new position separator.
func CreateSeparator(c *gin.Context) {
	var separator models.PositionSeparator
	if err := c.ShouldBindJSON(&separator); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if separator.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	// Set sort order to the max + 1 so it appears at the end
	var maxOrder int
	database.DB.Raw(`
		SELECT COALESCE(MAX(sort_order), -1) FROM (
			SELECT sort_order FROM positions WHERE deleted_at IS NULL
			UNION ALL
			SELECT sort_order FROM position_separators WHERE deleted_at IS NULL
		)
	`).Scan(&maxOrder)
	separator.SortOrder = maxOrder + 1

	if err := database.DB.Create(&separator).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, separator)
}

// UpdateSeparator updates a separator's name.
func UpdateSeparator(c *gin.Context) {
	id := c.Param("id")
	var separator models.PositionSeparator
	if err := database.DB.First(&separator, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Separator not found"})
		return
	}

	var input struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	separator.Name = input.Name
	if err := database.DB.Save(&separator).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, separator)
}

// DeleteSeparator deletes a position separator.
func DeleteSeparator(c *gin.Context) {
	id := c.Param("id")
	if err := database.DB.Delete(&models.PositionSeparator{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Separator deleted"})
}
