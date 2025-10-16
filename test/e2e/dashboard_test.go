package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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

	// Get server port from environment variable
	serverPort := os.Getenv("TEST_SERVER_PORT")
	if serverPort == "" {
		serverPort = "8888" // fallback to default
	}

	serverURL := "http://localhost:" + serverPort

	t.Run("Health", testHealth(serverURL))
	t.Run("Components", testComponents(serverURL))
	t.Run("Outages", testOutages(serverURL))
	t.Run("UpdateOutage", testUpdateOutage(serverURL))
	t.Run("DeleteOutage", testDeleteOutage(serverURL))
	t.Run("GetOutage", testGetOutage(serverURL))

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
		assert.Equal(t, "Prow", components[0].Name)
		assert.Equal(t, "Backbone of the CI system", components[0].Description)
		assert.Equal(t, "TestPlatform", components[0].ShipTeam)
		assert.Equal(t, "#test-channel", components[0].SlackChannel)
		assert.Len(t, components[0].Subcomponents, 2)
		assert.Equal(t, "Tide", components[0].Subcomponents[0].Name)
		assert.Equal(t, "Deck", components[0].Subcomponents[1].Name)
	}
}

// createOutage is a helper function to create an outage for testing
func createOutage(t *testing.T, serverURL, componentName, subComponentName string) types.Outage {
	outagePayload := map[string]interface{}{
		"severity":        "Down",
		"start_time":      time.Now().UTC().Format(time.RFC3339),
		"description":     "Test outage for " + subComponentName,
		"discovered_from": "e2e-test",
		"created_by":      "test-user",
	}

	payloadBytes, err := json.Marshal(outagePayload)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", serverURL+"/api/components/"+componentName+"/"+subComponentName+"/outages",
		bytes.NewBuffer(payloadBytes))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var outage types.Outage
	err = json.NewDecoder(resp.Body).Decode(&outage)
	require.NoError(t, err)

	return outage
}

// deleteOutage is a helper function to delete an outage for cleanup
func deleteOutage(t *testing.T, serverURL, componentName, subComponentName string, outageID uint) {
	req, err := http.NewRequest("DELETE", serverURL+"/api/components/"+componentName+"/"+subComponentName+"/outages/"+fmt.Sprintf("%d", outageID), nil)
	require.NoError(t, err)

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func testOutages(serverURL string) func(*testing.T) {
	return func(t *testing.T) {
		t.Run("POST to top-level component fails (old endpoint)", func(t *testing.T) {
			outagePayload := map[string]interface{}{
				"severity":        "Down",
				"start_time":      time.Now().UTC().Format(time.RFC3339),
				"description":     "Test outage",
				"discovered_from": "e2e-test",
				"created_by":      "test-user",
			}

			payloadBytes, err := json.Marshal(outagePayload)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", serverURL+"/api/components/Prow/outages",
				bytes.NewBuffer(payloadBytes))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
		})

		t.Run("POST to sub-component with new URI structure succeeds", func(t *testing.T) {
			outage := createOutage(t, serverURL, "Prow", "Tide")
			defer deleteOutage(t, serverURL, "Prow", "Tide", outage.ID)

			assert.NotZero(t, outage.ID)
			assert.Equal(t, "Tide", outage.ComponentName)
			assert.Equal(t, "Down", outage.Severity)
			assert.Equal(t, "e2e-test", outage.DiscoveredFrom)
		})

		t.Run("POST to non-existent sub-component fails", func(t *testing.T) {
			outagePayload := map[string]interface{}{
				"severity":        "Down",
				"start_time":      time.Now().UTC().Format(time.RFC3339),
				"description":     "Test outage for non-existent sub-component",
				"discovered_from": "e2e-test",
				"created_by":      "test-user",
			}

			payloadBytes, err := json.Marshal(outagePayload)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", serverURL+"/api/components/Prow/NonExistentSub/outages",
				bytes.NewBuffer(payloadBytes))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		})

		t.Run("GET on top-level component aggregates sub-components", func(t *testing.T) {
			// Create outages for different sub-components
			tideOutage := createOutage(t, serverURL, "Prow", "Tide")
			defer deleteOutage(t, serverURL, "Prow", "Tide", tideOutage.ID)
			deckOutage := createOutage(t, serverURL, "Prow", "Deck")
			defer deleteOutage(t, serverURL, "Prow", "Deck", deckOutage.ID)

			resp, err := http.Get(serverURL + "/api/components/Prow/outages")
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var outages []types.Outage
			err = json.NewDecoder(resp.Body).Decode(&outages)
			require.NoError(t, err)

			// Should have exactly our 2 outages since we clean up after ourselves
			assert.Len(t, outages, 2)

			// Verify our specific outages are present
			outageIDs := make(map[uint]bool)
			for _, outage := range outages {
				outageIDs[outage.ID] = true
			}
			assert.True(t, outageIDs[tideOutage.ID], "Tide outage should be present")
			assert.True(t, outageIDs[deckOutage.ID], "Deck outage should be present")
		})

		t.Run("GET on sub-component returns only that sub-component's outages", func(t *testing.T) {
			// Create outages for different sub-components
			tideOutage1 := createOutage(t, serverURL, "Prow", "Tide")
			defer deleteOutage(t, serverURL, "Prow", "Tide", tideOutage1.ID)
			tideOutage2 := createOutage(t, serverURL, "Prow", "Tide")
			defer deleteOutage(t, serverURL, "Prow", "Tide", tideOutage2.ID)
			deckOutage := createOutage(t, serverURL, "Prow", "Deck")
			defer deleteOutage(t, serverURL, "Prow", "Deck", deckOutage.ID)

			resp, err := http.Get(serverURL + "/api/components/Prow/Tide/outages")
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var outages []types.Outage
			err = json.NewDecoder(resp.Body).Decode(&outages)
			require.NoError(t, err)

			// Should have exactly our 2 Tide outages since we clean up after ourselves
			assert.Len(t, outages, 2)

			// All outages should be for Tide only
			for _, outage := range outages {
				assert.Equal(t, "Tide", outage.ComponentName)
			}

			// Verify our specific outages are present
			outageIDs := make(map[uint]bool)
			for _, outage := range outages {
				outageIDs[outage.ID] = true
			}
			assert.True(t, outageIDs[tideOutage1.ID], "First Tide outage should be present")
			assert.True(t, outageIDs[tideOutage2.ID], "Second Tide outage should be present")
			assert.False(t, outageIDs[deckOutage.ID], "Deck outage should not be included")
		})

		t.Run("GET on non-existent sub-component fails", func(t *testing.T) {
			// This test doesn't need any setup - it should fail regardless of existing data
			resp, err := http.Get(serverURL + "/api/components/Prow/NonExistentSub/outages")
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		})
	}
}

func testUpdateOutage(serverURL string) func(*testing.T) {
	return func(t *testing.T) {
		// Create an outage to update
		createdOutage := createOutage(t, serverURL, "Prow", "Tide")
		defer deleteOutage(t, serverURL, "Prow", "Tide", createdOutage.ID)

		// Now update the outage
		updatePayload := map[string]interface{}{
			"severity":     "Degraded",
			"description":  "Updated description",
			"resolved_by":  "test-resolver",
			"triage_notes": "Updated triage notes",
		}

		updateBytes, err := json.Marshal(updatePayload)
		require.NoError(t, err)

		updateURL := serverURL + "/api/components/Prow/Tide/outages/" + fmt.Sprintf("%d", createdOutage.ID)
		t.Logf("Making PATCH request to: %s", updateURL)

		updateReq, err := http.NewRequest("PATCH", updateURL, bytes.NewBuffer(updateBytes))
		require.NoError(t, err)
		updateReq.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		updateResp, err := client.Do(updateReq)
		require.NoError(t, err)
		defer updateResp.Body.Close()

		if updateResp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(updateResp.Body)
			t.Logf("Unexpected status %d, body: %s", updateResp.StatusCode, string(body))
		}

		assert.Equal(t, http.StatusOK, updateResp.StatusCode)
		assert.Equal(t, "application/json", updateResp.Header.Get("Content-Type"))

		var updatedOutage types.Outage
		err = json.NewDecoder(updateResp.Body).Decode(&updatedOutage)
		require.NoError(t, err)

		assert.Equal(t, createdOutage.ID, updatedOutage.ID)
		assert.Equal(t, "Degraded", updatedOutage.Severity)
		assert.Equal(t, "Updated description", updatedOutage.Description)
		assert.Equal(t, "test-resolver", *updatedOutage.ResolvedBy)
		assert.Equal(t, "Updated triage notes", *updatedOutage.TriageNotes)
		assert.WithinDuration(t, createdOutage.StartTime.UTC(), updatedOutage.StartTime.UTC(), time.Second) // Should remain unchanged
		assert.Equal(t, createdOutage.CreatedBy, updatedOutage.CreatedBy)                                   // Should remain unchanged

		// Test updating non-existent outage
		nonExistentReq, err := http.NewRequest("PATCH",
			serverURL+"/api/components/Prow/Tide/outages/99999",
			bytes.NewBuffer(updateBytes))
		require.NoError(t, err)
		nonExistentReq.Header.Set("Content-Type", "application/json")

		nonExistentResp, err := client.Do(nonExistentReq)
		require.NoError(t, err)
		defer nonExistentResp.Body.Close()

		assert.Equal(t, http.StatusNotFound, nonExistentResp.StatusCode)

		// Test updating with invalid component
		invalidComponentReq, err := http.NewRequest("PATCH",
			serverURL+"/api/components/NonExistentComponent/Tide/outages/"+fmt.Sprintf("%d", createdOutage.ID),
			bytes.NewBuffer(updateBytes))
		require.NoError(t, err)
		invalidComponentReq.Header.Set("Content-Type", "application/json")

		invalidComponentResp, err := client.Do(invalidComponentReq)
		require.NoError(t, err)
		defer invalidComponentResp.Body.Close()

		assert.Equal(t, http.StatusNotFound, invalidComponentResp.StatusCode)
	}
}

func testDeleteOutage(serverURL string) func(*testing.T) {
	return func(t *testing.T) {
		t.Run("DELETE existing outage succeeds", func(t *testing.T) {
			// Create an outage to delete
			createdOutage := createOutage(t, serverURL, "Prow", "Tide")

			// Delete the outage
			deleteOutage(t, serverURL, "Prow", "Tide", createdOutage.ID)

			// Verify the outage is deleted by trying to get it
			resp, err := http.Get(serverURL + "/api/components/Prow/Tide/outages")
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var outages []types.Outage
			err = json.NewDecoder(resp.Body).Decode(&outages)
			require.NoError(t, err)

			// The deleted outage should not be in the list
			for _, outage := range outages {
				assert.NotEqual(t, createdOutage.ID, outage.ID, "Deleted outage should not be present")
			}
		})

		t.Run("DELETE non-existent outage returns 404", func(t *testing.T) {
			req, err := http.NewRequest("DELETE", serverURL+"/api/components/Prow/Tide/outages/99999", nil)
			require.NoError(t, err)

			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		})

		t.Run("DELETE outage from non-existent component returns 404", func(t *testing.T) {
			req, err := http.NewRequest("DELETE", serverURL+"/api/components/NonExistentComponent/Tide/outages/1", nil)
			require.NoError(t, err)

			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		})

		t.Run("DELETE outage from non-existent sub-component returns 404", func(t *testing.T) {
			req, err := http.NewRequest("DELETE", serverURL+"/api/components/Prow/NonExistentSub/outages/1", nil)
			require.NoError(t, err)

			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		})
	}
}

func testGetOutage(serverURL string) func(*testing.T) {
	return func(t *testing.T) {
		t.Run("GET existing outage succeeds", func(t *testing.T) {
			// Create an outage to retrieve
			createdOutage := createOutage(t, serverURL, "Prow", "Tide")
			defer deleteOutage(t, serverURL, "Prow", "Tide", createdOutage.ID)

			// Get the outage
			resp, err := http.Get(serverURL + "/api/components/Prow/Tide/outages/" + fmt.Sprintf("%d", createdOutage.ID))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

			var outage types.Outage
			err = json.NewDecoder(resp.Body).Decode(&outage)
			require.NoError(t, err)

			assert.Equal(t, createdOutage.ID, outage.ID)
			assert.Equal(t, "Tide", outage.ComponentName)
			assert.Equal(t, "Down", outage.Severity)
			assert.Equal(t, "e2e-test", outage.DiscoveredFrom)
			assert.Equal(t, "test-user", outage.CreatedBy)
		})

		t.Run("GET non-existent outage returns 404", func(t *testing.T) {
			resp, err := http.Get(serverURL + "/api/components/Prow/Tide/outages/99999")
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		})

		t.Run("GET outage from non-existent component returns 404", func(t *testing.T) {
			resp, err := http.Get(serverURL + "/api/components/NonExistentComponent/Tide/outages/1")
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		})

		t.Run("GET outage from non-existent sub-component returns 404", func(t *testing.T) {
			resp, err := http.Get(serverURL + "/api/components/Prow/NonExistentSub/outages/1")
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		})

		t.Run("GET outage with wrong sub-component returns 404", func(t *testing.T) {
			// Create an outage for Tide
			tideOutage := createOutage(t, serverURL, "Prow", "Tide")
			defer deleteOutage(t, serverURL, "Prow", "Tide", tideOutage.ID)

			// Try to get it as if it were a Deck outage
			resp, err := http.Get(serverURL + "/api/components/Prow/Deck/outages/" + fmt.Sprintf("%d", tideOutage.ID))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		})
	}
}
