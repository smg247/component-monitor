package main

import (
	"database/sql"
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

func (h *Handlers) getComponent(componentName string) *types.Component {
	for _, component := range h.config.Components {
		if component.Name == componentName {
			return &component
		}
	}
	return nil
}

func (h *Handlers) validateOutage(outage *types.Outage) (string, bool) {
	if outage.Severity == "" {
		return "Severity is required", false
	}
	if !types.IsValidSeverity(string(outage.Severity)) {
		return "Invalid severity. Must be one of: Down, Degraded, Suspected", false
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

	component := h.getComponent(componentName)
	if component == nil {
		respondWithError(w, http.StatusNotFound, "Component not found")
		return
	}
	subComponents := []string{}
	for _, subComponent := range component.Subcomponents {
		subComponents = append(subComponents, subComponent.Name)
	}

	var outages []types.Outage
	if err := h.db.Where("component_name IN ?", subComponents).Order("start_time DESC").Find(&outages).Error; err != nil {
		logger.WithField("error", err).Error("Failed to query outages from database")
		respondWithError(w, http.StatusInternalServerError, "Failed to get outages")
		return
	}

	respondWithJSON(w, http.StatusOK, outages)
}

// GetSubComponentOutages retrieves outages for a specific sub-component.
func (h *Handlers) GetSubComponentOutages(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	componentName := vars["componentName"]
	subComponentName := vars["subComponentName"]

	logger := h.logger.WithFields(logrus.Fields{
		"component":     componentName,
		"sub_component": subComponentName,
	})

	component := h.getComponent(componentName)
	if component == nil {
		respondWithError(w, http.StatusNotFound, "Component not found")
		return
	}

	subComponent := component.GetSubComponent(subComponentName)
	if subComponent == nil {
		respondWithError(w, http.StatusNotFound, "Sub-component not found")
		return
	}

	var outages []types.Outage
	if err := h.db.Where("component_name = ?", subComponentName).Order("start_time DESC").Find(&outages).Error; err != nil {
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

	component := h.getComponent(componentName)
	if component == nil {
		respondWithError(w, http.StatusNotFound, "Component not found")
		return
	}
	subComponent := component.GetSubComponent(subComponentName)
	if subComponent == nil {
		respondWithError(w, http.StatusNotFound, "Sub-Component not found")
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

	component := h.getComponent(componentName)
	if component == nil {
		respondWithError(w, http.StatusNotFound, "Component not found")
		return
	}

	subComponent := component.GetSubComponent(subComponentName)
	if subComponent == nil {
		respondWithError(w, http.StatusNotFound, "Sub-Component not found")
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

	if updateReq.Severity != nil {
		if !types.IsValidSeverity(*updateReq.Severity) {
			respondWithError(w, http.StatusBadRequest, "Invalid severity. Must be one of: Down, Degraded, Suspected")
			return
		}
		outage.Severity = types.Severity(*updateReq.Severity)
	}
	if updateReq.StartTime != nil {
		outage.StartTime = *updateReq.StartTime
	}
	if updateReq.EndTime != nil {
		outage.EndTime = sql.NullTime{Time: *updateReq.EndTime, Valid: true}
	}
	if updateReq.Description != nil {
		outage.Description = *updateReq.Description
	}
	if updateReq.ResolvedBy != nil {
		outage.ResolvedBy = updateReq.ResolvedBy
	}
	if updateReq.ConfirmedAt != nil {
		outage.ConfirmedAt = sql.NullTime{Time: *updateReq.ConfirmedAt, Valid: true}
	}
	if updateReq.TriageNotes != nil {
		outage.TriageNotes = updateReq.TriageNotes
	}

	if err := h.db.Save(&outage).Error; err != nil {
		logger.WithField("error", err).Error("Failed to update outage in database")
		respondWithError(w, http.StatusInternalServerError, "Failed to update outage")
		return
	}

	logger.Info("Successfully updated outage")

	respondWithJSON(w, http.StatusOK, outage)
}

// GetOutage retrieves a specific outage by ID for a specific sub-component.
func (h *Handlers) GetOutage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	componentName := vars["componentName"]
	subComponentName := vars["subComponentName"]
	outageId := vars["outageId"]

	logger := h.logger.WithFields(logrus.Fields{
		"component":     componentName,
		"sub_component": subComponentName,
		"outage_id":     outageId,
	})

	component := h.getComponent(componentName)
	if component == nil {
		respondWithError(w, http.StatusNotFound, "Component not found")
		return
	}

	subComponent := component.GetSubComponent(subComponentName)
	if subComponent == nil {
		respondWithError(w, http.StatusNotFound, "Sub-component not found")
		return
	}

	var outage types.Outage
	if err := h.db.Where("id = ? AND component_name = ?", outageId, subComponentName).First(&outage).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			respondWithError(w, http.StatusNotFound, "Outage not found")
			return
		}
		logger.WithField("error", err).Error("Failed to query outage from database")
		respondWithError(w, http.StatusInternalServerError, "Failed to get outage")
		return
	}

	logger.Info("Successfully retrieved outage")
	respondWithJSON(w, http.StatusOK, outage)
}

// DeleteOutage deletes an outage by ID for a specific sub-component.
func (h *Handlers) DeleteOutage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	componentName := vars["componentName"]
	subComponentName := vars["subComponentName"]
	outageId := vars["outageId"]

	logger := h.logger.WithFields(logrus.Fields{
		"component":     componentName,
		"sub_component": subComponentName,
		"outage_id":     outageId,
	})

	component := h.getComponent(componentName)
	if component == nil {
		respondWithError(w, http.StatusNotFound, "Component not found")
		return
	}

	subComponent := component.GetSubComponent(subComponentName)
	if subComponent == nil {
		respondWithError(w, http.StatusNotFound, "Sub-component not found")
		return
	}

	var outage types.Outage
	if err := h.db.Where("id = ? AND component_name = ?", outageId, subComponentName).First(&outage).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			respondWithError(w, http.StatusNotFound, "Outage not found")
			return
		}
		logger.WithField("error", err).Error("Failed to query outage from database")
		respondWithError(w, http.StatusInternalServerError, "Failed to get outage")
		return
	}

	if err := h.db.Delete(&outage).Error; err != nil {
		logger.WithField("error", err).Error("Failed to delete outage from database")
		respondWithError(w, http.StatusInternalServerError, "Failed to delete outage")
		return
	}

	logger.Info("Successfully deleted outage")
	w.WriteHeader(http.StatusNoContent)
}

// GetSubComponentStatus returns the status of a subcomponent based on active outages
func (h *Handlers) GetSubComponentStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	componentName := vars["componentName"]
	subComponentName := vars["subComponentName"]

	logger := h.logger.WithFields(logrus.Fields{
		"component":     componentName,
		"sub_component": subComponentName,
	})

	component := h.getComponent(componentName)
	if component == nil {
		respondWithError(w, http.StatusNotFound, "Component not found")
		return
	}

	subComponent := component.GetSubComponent(subComponentName)
	if subComponent == nil {
		respondWithError(w, http.StatusNotFound, "Sub-component not found")
		return
	}

	var outages []types.Outage
	if err := h.db.Where("component_name = ? AND (end_time IS NULL OR end_time > ?)", subComponentName, time.Now()).Order("start_time DESC").Find(&outages).Error; err != nil {
		logger.WithField("error", err).Error("Failed to query active outages from database")
		respondWithError(w, http.StatusInternalServerError, "Failed to get subcomponent status")
		return
	}

	status := types.StatusHealthy
	if len(outages) > 0 {
		mostCriticalSeverity := outages[0].Severity
		highestLevel := types.GetSeverityLevel(mostCriticalSeverity)

		for _, outage := range outages {
			level := types.GetSeverityLevel(outage.Severity)
			if level > highestLevel {
				highestLevel = level
				mostCriticalSeverity = outage.Severity
			}
		}
		status = mostCriticalSeverity.ToStatus()
	}

	response := types.ComponentStatus{
		Status:        status,
		ActiveOutages: outages,
	}

	logger.WithField("status", status).Info("Successfully retrieved subcomponent status")
	respondWithJSON(w, http.StatusOK, response)
}
