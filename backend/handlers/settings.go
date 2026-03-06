package handlers

import (
	"net/http"

	"github.com/RomanWilkening/budget_projection/backend/database"
	"github.com/RomanWilkening/budget_projection/backend/models"
	"github.com/gin-gonic/gin"
)

// GetSettings returns all application settings as a key-value map.
func GetSettings(c *gin.Context) {
	var settings []models.Setting
	if err := database.DB.Find(&settings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result := make(map[string]string)
	for _, s := range settings {
		result[s.Key] = s.Value
	}
	c.JSON(http.StatusOK, result)
}

// UpdateSettings updates one or more application settings.
func UpdateSettings(c *gin.Context) {
	var input map[string]string
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	for key, value := range input {
		setting := models.Setting{Key: key, Value: value}
		if err := database.DB.Save(&setting).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// If Banking Bridge URL was updated, apply it to the client
	if url, ok := input["banking_bridge_url"]; ok {
		bridgeClient.SetBaseURL(url)
	}

	// Return all settings
	var settings []models.Setting
	if err := database.DB.Find(&settings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result := make(map[string]string)
	for _, s := range settings {
		result[s.Key] = s.Value
	}
	c.JSON(http.StatusOK, result)
}

// InitBridgeClientFromDB loads the Banking Bridge URL from the database
// and updates the client. Called on startup after DB init.
func InitBridgeClientFromDB() {
	var setting models.Setting
	if err := database.DB.Where("key = ?", "banking_bridge_url").First(&setting).Error; err == nil {
		if setting.Value != "" {
			bridgeClient.SetBaseURL(setting.Value)
		}
	}
}
