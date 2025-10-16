package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"ship-status-dash/pkg/types"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_Dashboard(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	connStr := os.Getenv("TEST_DATABASE_DSN")
	if connStr == "" {
		t.Fatal("TEST_DATABASE_DSN environment variable is required")
	}

	t.Logf("Using PostgreSQL at: %s", connStr)

	// Run migration
	t.Log("Running migration...")
	migrateCmd := exec.Command("go", "run", "../../cmd/migrate", "--dsn", connStr)
	migrateOutput, err := migrateCmd.CombinedOutput()
	require.NoError(t, err, "Migration failed: %s", string(migrateOutput))
	t.Logf("Migration output: %s", string(migrateOutput))

	// Get path to test config file
	configPath, err := filepath.Abs("config.yaml")
	require.NoError(t, err)
	t.Logf("Using test config at: %s", configPath)

	// Start dashboard server
	t.Log("Starting dashboard server...")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dashboardCmd := exec.CommandContext(ctx, "go", "run", "../../cmd/dashboard", "--config", configPath, "--port", "8888", "--dsn", connStr)
	err = dashboardCmd.Start()
	require.NoError(t, err)

	// Wait for server to be ready
	serverURL := "http://localhost:8888"
	require.Eventually(t, func() bool {
		resp, err := http.Get(serverURL + "/health")
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 10*time.Second, 500*time.Millisecond, "Server failed to start")

	t.Log("Dashboard server is ready")

	t.Run("Health", testHealth(serverURL))
	t.Run("Components", testComponents(serverURL))
	t.Run("Outages", testOutages(serverURL))

	t.Log("All tests passed!")
}

func testHealth(serverURL string) func(*testing.T) {
	return func(t *testing.T) {
		resp, err := http.Get(serverURL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		var health map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&health)
		require.NoError(t, err)

		assert.Equal(t, "ok", health["status"])
		assert.NotEmpty(t, health["time"])
	}
}

func testComponents(serverURL string) func(*testing.T) {
	return func(t *testing.T) {
		resp, err := http.Get(serverURL + "/api/components")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		var components []types.Component
		err = json.NewDecoder(resp.Body).Decode(&components)
		require.NoError(t, err)

		assert.Len(t, components, 1)
		assert.Equal(t, "TestComponent", components[0].Name)
		assert.Equal(t, "A test component", components[0].Description)
		assert.Equal(t, "TestTeam", components[0].ShipTeam)
		assert.Equal(t, "#test-channel", components[0].SlackChannel)
		assert.Len(t, components[0].Subcomponents, 1)
		assert.Equal(t, "SubTest", components[0].Subcomponents[0].Name)
	}
}

func testOutages(serverURL string) func(*testing.T) {
	return func(t *testing.T) {
		t.Run("POST to top-level component fails", func(t *testing.T) {
			outagePayload := map[string]interface{}{
				"severity":        "Down",
				"start_time":      time.Now().UTC().Format(time.RFC3339),
				"description":     "Test outage",
				"discovered_from": "e2e-test",
				"created_by":      "test-user",
			}

			payloadBytes, err := json.Marshal(outagePayload)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", serverURL+"/api/components/TestComponent/outages",
				bytes.NewBuffer(payloadBytes))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})

		t.Run("POST to sub-component succeeds", func(t *testing.T) {
			outagePayload := map[string]interface{}{
				"severity":        "Down",
				"start_time":      time.Now().UTC().Format(time.RFC3339),
				"description":     "Test outage for sub-component",
				"discovered_from": "e2e-test",
				"created_by":      "test-user",
			}

			payloadBytes, err := json.Marshal(outagePayload)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", serverURL+"/api/components/SubTest/outages",
				bytes.NewBuffer(payloadBytes))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusCreated, resp.StatusCode)
			assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

			var outage types.Outage
			err = json.NewDecoder(resp.Body).Decode(&outage)
			require.NoError(t, err)

			assert.NotZero(t, outage.ID)
			assert.Equal(t, "SubTest", outage.ComponentName)
			assert.Equal(t, "Down", outage.Severity)
			assert.Equal(t, "e2e-test", outage.DiscoveredFrom)
		})

		t.Run("GET on top-level component aggregates sub-components", func(t *testing.T) {
			resp, err := http.Get(serverURL + "/api/components/TestComponent/outages")
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var outages []types.Outage
			err = json.NewDecoder(resp.Body).Decode(&outages)
			require.NoError(t, err)

			assert.Len(t, outages, 1)
			assert.Equal(t, "SubTest", outages[0].ComponentName)
		})
	}
}
