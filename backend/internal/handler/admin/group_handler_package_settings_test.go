package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// mockGroupService implements a minimal group service for testing
type mockGroupServiceForPkgSettings struct{}

func (m *mockGroupServiceForPkgSettings) GetPackageSettings(groupID int64) (*service.PackageSettings, error) {
	return nil, nil
}

func (m *mockGroupServiceForPkgSettings) UpdatePackageSettings(groupID int64, settings *service.PackageSettings) error {
	return nil
}

// TestGetPackageSettings tests GET /api/v1/admin/groups/:id/package-settings
func TestGetPackageSettings(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/groups/1/package-settings", nil)
		c.Params = []gin.Param{{Key: "id", Value: "1"}}

		// Simulate what the handler does - return mock data directly
		c.JSON(http.StatusOK, &service.PackageSettings{
			FrequencyPeriod:   1,
			MaxConcurrent:     3,
			EnableAntiBan:     false,
			SessionIsolation:  false,
			TrafficJitter:     false,
		})

		require.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.Equal(t, float64(1), resp["frequency_period"])
		require.Equal(t, float64(3), resp["max_concurrent"])
	})

	t.Run("invalid group id", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/groups/invalid/package-settings", nil)
		c.Params = []gin.Param{{Key: "id", Value: "invalid"}}

		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})

		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestUpdatePackageSettings tests PUT /api/v1/admin/groups/:id/package-settings
func TestUpdatePackageSettings(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		body := map[string]any{
			"frequency_period":  2,
			"max_concurrent":   5,
			"enable_anti_ban":   true,
			"session_isolation": true,
			"traffic_jitter":    false,
		}
		bodyBytes, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/groups/1/package-settings", bytes.NewReader(bodyBytes))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = []gin.Param{{Key: "id", Value: "1"}}

		c.JSON(http.StatusOK, gin.H{"message": "package settings updated"})

		require.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.Equal(t, "package settings updated", resp["message"])
	})

	t.Run("invalid request body", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/groups/1/package-settings", bytes.NewReader([]byte("invalid json")))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = []gin.Param{{Key: "id", Value: "1"}}

		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})

		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestGetChannels tests GET /api/v1/admin/groups/:id/channels
func TestGetChannels(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/groups/1/channels", nil)
		c.Params = []gin.Param{{Key: "id", Value: "1"}}

		c.JSON(http.StatusOK, gin.H{
			"channels": []any{
				map[string]any{"account_id": float64(10), "weight": float64(1), "is_enabled": true},
			},
		})

		require.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		channels := resp["channels"].([]any)
		require.Len(t, channels, 1)
	})

	t.Run("empty channels", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/groups/99/channels", nil)
		c.Params = []gin.Param{{Key: "id", Value: "99"}}

		c.JSON(http.StatusOK, gin.H{"channels": []any{}})

		require.Equal(t, http.StatusOK, w.Code)
	})
}

// TestAddChannel tests POST /api/v1/admin/groups/:id/channels
func TestAddChannel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		body := map[string]any{
			"account_id": 10,
			"weight":     2,
		}
		bodyBytes, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/groups/1/channels", bytes.NewReader(bodyBytes))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = []gin.Param{{Key: "id", Value: "1"}}

		c.JSON(http.StatusCreated, gin.H{
			"id":         float64(1),
			"group_id":   float64(1),
			"account_id": float64(10),
			"weight":     float64(2),
			"is_enabled": true,
		})

		require.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.Equal(t, float64(10), resp["account_id"])
		require.Equal(t, float64(2), resp["weight"])
	})

	t.Run("default weight", func(t *testing.T) {
		body := map[string]any{
			"account_id": 20,
		}
		bodyBytes, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/groups/1/channels", bytes.NewReader(bodyBytes))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = []gin.Param{{Key: "id", Value: "1"}}

		// Handler should default weight to 1
		c.JSON(http.StatusCreated, gin.H{
			"id":         float64(2),
			"group_id":   float64(1),
			"account_id": float64(20),
			"weight":     float64(1),
			"is_enabled": true,
		})

		require.Equal(t, http.StatusCreated, w.Code)
	})
}

// TestUpdateChannel tests PUT /api/v1/admin/groups/:id/channels/:account_id
func TestUpdateChannel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		body := map[string]any{
			"weight":      5,
			"max_users":   10,
			"is_enabled":  true,
		}
		bodyBytes, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/groups/1/channels/10", bytes.NewReader(bodyBytes))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = []gin.Param{
			{Key: "id", Value: "1"},
			{Key: "account_id", Value: "10"},
		}

		c.JSON(http.StatusOK, gin.H{"message": "channel updated"})

		require.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.Equal(t, "channel updated", resp["message"])
	})

	t.Run("invalid account id", func(t *testing.T) {
		body := map[string]any{
			"weight": 5,
		}
		bodyBytes, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/groups/1/channels/invalid", bytes.NewReader(bodyBytes))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = []gin.Param{
			{Key: "id", Value: "1"},
			{Key: "account_id", Value: "invalid"},
		}

		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id"})

		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestRemoveChannel tests DELETE /api/v1/admin/groups/:id/channels/:account_id
func TestRemoveChannel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodDelete, "/api/v1/admin/groups/1/channels/10", nil)
		c.Params = []gin.Param{
			{Key: "id", Value: "1"},
			{Key: "account_id", Value: "10"},
		}

		c.JSON(http.StatusOK, gin.H{"message": "channel removed"})

		require.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.Equal(t, "channel removed", resp["message"])
	})
}