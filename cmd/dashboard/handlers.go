package main

import (
	"encoding/json"
	"net/http"
	"ship-status-dash/pkg/types"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Handlers contains the HTTP request handlers for the dashboard API.
type Handlers struct {
	logger *logrus.Logger
	config *types.Config
	db     *gorm.DB
}

// NewHandlers creates a new Handlers instance with the provided dependencies.
func NewHandlers(logger *logrus.Logger, config *types.Config, db *gorm.DB) *Handlers {
	return &Handlers{
		logger: logger,
		config: config,
		db:     db,
	}
}

func respondWithJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func respondWithError(w http.ResponseWriter, statusCode int, message string) {
	respondWithJSON(w, statusCode, map[string]string{
		"error": message,
	})
}

func (h *Handlers) findComponent(componentName string) (*types.Component, *types.SubComponent) {
	for _, component := range h.config.Components {
		if component.Name == componentName {
			return &component, nil
		}
		for _, subComponent := range component.Subcomponents {
			if subComponent.Name == componentName {
				return &component, &subComponent
			}
		}
	}
	return nil, nil
}

func (h *Handlers) componentExists(componentName string) bool {
	component, subComponent := h.findComponent(componentName)
	return component != nil || subComponent != nil
}

func (h *Handlers) isTopLevelComponent(componentName string) bool {
	component, subComponent := h.findComponent(componentName)
	return component != nil && subComponent == nil
}

func (h *Handlers) getSubComponentNames(topLevelComponentName string) []string {
	component, _ := h.findComponent(topLevelComponentName)
	if component == nil {
		return nil
	}

	names := make([]string, len(component.Subcomponents))
	for i, sub := range component.Subcomponents {
		names[i] = sub.Name
	}
	return names
}

func (h *Handlers) validateSubComponentBelongsToComponent(componentName, subComponentName string) bool {
	for _, component := range h.config.Components {
		if component.Name == componentName {
			for _, subComponent := range component.Subcomponents {
				if subComponent.Name == subComponentName {
					return true
				}
			}
			return false
		}
	}
	return false
}

func (h *Handlers) validateOutage(outage *types.Outage) (string, bool) {
	if outage.Severity == "" {
		return "Severity is required", false
	}
	if outage.StartTime.IsZero() {
		return "StartTime is required", false
	}
	if outage.DiscoveredFrom == "" {
		return "DiscoveredFrom is required", false
	}
	if outage.CreatedBy == "" {
		return "CreatedBy is required", false
	}
	return "", true
}

// Health returns the health status of the dashboard service.
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	}
	respondWithJSON(w, http.StatusOK, response)
}

// GetComponents returns the list of configured components.
func (h *Handlers) GetComponents(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, http.StatusOK, h.config.Components)
}

// GetOutages retrieves outages for a specific component, aggregating sub-component outages for top-level components.
func (h *Handlers) GetOutages(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	componentName := vars["componentName"]

	logger := h.logger.WithField("component", componentName)

	if !h.componentExists(componentName) {
		respondWithError(w, http.StatusNotFound, "Component not found")
		return
	}

	var outages []types.Outage
	var componentNames []string

	if h.isTopLevelComponent(componentName) {
		subComponents := h.getSubComponentNames(componentName)
		if len(subComponents) == 0 {
			respondWithJSON(w, http.StatusOK, []types.Outage{})
			return
		}
		componentNames = subComponents
	} else {
		componentNames = []string{componentName}
	}

	if err := h.db.Where("component_name IN ?", componentNames).Order("start_time DESC").Find(&outages).Error; err != nil {
		logger.WithField("error", err).Error("Failed to query outages from database")
		respondWithError(w, http.StatusInternalServerError, "Failed to get outages")
		return
	}

	respondWithJSON(w, http.StatusOK, outages)
}

// CreateOutage creates a new outage for a sub-component.
func (h *Handlers) CreateOutage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	componentName := vars["componentName"]
	subComponentName := vars["subComponentName"]

	if !h.componentExists(componentName) {
		respondWithError(w, http.StatusNotFound, "Component not found")
		return
	}

	if !h.validateSubComponentBelongsToComponent(componentName, subComponentName) {
		respondWithError(w, http.StatusNotFound, "Sub-component not found or does not belong to the specified component")
		return
	}

	var outage types.Outage
	if err := json.NewDecoder(r.Body).Decode(&outage); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	outage.ComponentName = subComponentName

	if message, valid := h.validateOutage(&outage); !valid {
		respondWithError(w, http.StatusBadRequest, message)
		return
	}

	logger := h.logger.WithFields(logrus.Fields{
		"component":       componentName,
		"sub_component":   subComponentName,
		"severity":        outage.Severity,
		"created_by":      outage.CreatedBy,
		"discovered_from": outage.DiscoveredFrom,
	})

	if err := h.db.Create(&outage).Error; err != nil {
		logger.WithField("error", err).Error("Failed to create outage in database")
		respondWithError(w, http.StatusInternalServerError, "Failed to create outage")
		return
	}

	logger.Infof("Successfully created outage: %d", outage.ID)

	respondWithJSON(w, http.StatusCreated, outage)
}

// UpdateOutageRequest represents the fields that can be updated in a PATCH request.
type UpdateOutageRequest struct {
	Severity    *string    `json:"severity,omitempty"`
	StartTime   *time.Time `json:"start_time,omitempty"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	Description *string    `json:"description,omitempty"`
	ResolvedBy  *string    `json:"resolved_by,omitempty"`
	ConfirmedAt *time.Time `json:"confirmed_at,omitempty"`
	TriageNotes *string    `json:"triage_notes,omitempty"`
}

// UpdateOutage updates an existing outage with the provided fields.
func (h *Handlers) UpdateOutage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	componentName := vars["componentName"]
	subComponentName := vars["subComponentName"]
	outageIDStr := vars["outageId"]

	outageID, err := strconv.ParseUint(outageIDStr, 10, 32)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid outage ID")
		return
	}

	logger := h.logger.WithFields(logrus.Fields{
		"outage_id":     outageID,
		"component":     componentName,
		"sub_component": subComponentName,
	})
	logger.Info("Updating outage")

	if !h.componentExists(componentName) {
		respondWithError(w, http.StatusNotFound, "Component not found")
		return
	}

	if !h.validateSubComponentBelongsToComponent(componentName, subComponentName) {
		respondWithError(w, http.StatusNotFound, "Sub-component not found or does not belong to the specified component")
		return
	}

	var outage types.Outage
	if err := h.db.Where("id = ? AND component_name = ?", uint(outageID), subComponentName).First(&outage).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			respondWithError(w, http.StatusNotFound, "Outage not found")
			return
		}
		logger.WithField("error", err).Error("Failed to query outage from database")
		respondWithError(w, http.StatusInternalServerError, "Failed to get outage")
		return
	}

	var updateReq UpdateOutageRequest
	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	updates := make(map[string]interface{})

	if updateReq.Severity != nil {
		updates["severity"] = *updateReq.Severity
	}
	if updateReq.StartTime != nil {
		updates["start_time"] = *updateReq.StartTime
	}
	if updateReq.EndTime != nil {
		updates["end_time"] = *updateReq.EndTime
	}
	if updateReq.Description != nil {
		updates["description"] = *updateReq.Description
	}
	if updateReq.ResolvedBy != nil {
		updates["resolved_by"] = *updateReq.ResolvedBy
	}
	if updateReq.ConfirmedAt != nil {
		updates["confirmed_at"] = *updateReq.ConfirmedAt
	}
	if updateReq.TriageNotes != nil {
		updates["triage_notes"] = *updateReq.TriageNotes
	}

	if len(updates) == 0 {
		respondWithJSON(w, http.StatusOK, outage)
		return
	}

	if err := h.db.Model(&outage).Updates(updates).Error; err != nil {
		logger.WithField("error", err).Error("Failed to update outage in database")
		respondWithError(w, http.StatusInternalServerError, "Failed to update outage")
		return
	}

	if err := h.db.Where("id = ?", uint(outageID)).First(&outage).Error; err != nil {
		logger.WithField("error", err).Error("Failed to fetch updated outage from database")
		respondWithError(w, http.StatusInternalServerError, "Failed to get updated outage")
		return
	}

	logger.Info("Successfully updated outage")

	respondWithJSON(w, http.StatusOK, outage)
}
