package handlers

import (
	"net/http"

	"github.com/RomanWilkening/budget_projection/backend/database"
	"github.com/RomanWilkening/budget_projection/backend/models"
	"github.com/gin-gonic/gin"
)

// allowedSettings is the whitelist of setting keys that can be stored.
var allowedSettings = map[string]bool{
	"banking_bridge_url": true,
}

// loadSettingsMap returns all settings from the database as a key-value map.
func loadSettingsMap() (map[string]string, error) {
	var settings []models.Setting
	if err := database.DB.Find(&settings).Error; err != nil {
		return nil, err
	}
	result := make(map[string]string)
	for _, s := range settings {
		result[s.Key] = s.Value
	}
	return result, nil
}

// GetSettings returns all application settings as a key-value map.
func GetSettings(c *gin.Context) {
	result, err := loadSettingsMap()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
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

	// Validate keys against whitelist
	for key := range input {
		if !allowedSettings[key] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown setting: " + key})
			return
		}
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

	result, err := loadSettingsMap()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
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
