package main

import (
	"component-monitor/pkg/types"
	"encoding/json"
	"net/http"
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
	component, _ := h.findComponent(componentName)
	return component != nil
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
		h.logger.WithFields(logrus.Fields{
			"component": componentName,
			"error":     err,
		}).Error("Failed to query outages from database")
		respondWithError(w, http.StatusInternalServerError, "Failed to get outages")
		return
	}

	respondWithJSON(w, http.StatusOK, outages)
}

// CreateOutage creates a new outage for a sub-component.
func (h *Handlers) CreateOutage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	componentName := vars["componentName"]

	if !h.componentExists(componentName) {
		respondWithError(w, http.StatusNotFound, "Component not found")
		return
	}

	if h.isTopLevelComponent(componentName) {
		respondWithError(w, http.StatusBadRequest, "Cannot create outages for top-level components. Create outages for sub-components instead.")
		return
	}

	var outage types.Outage
	if err := json.NewDecoder(r.Body).Decode(&outage); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	outage.ComponentName = componentName

	if message, valid := h.validateOutage(&outage); !valid {
		respondWithError(w, http.StatusBadRequest, message)
		return
	}

	if err := h.db.Create(&outage).Error; err != nil {
		h.logger.WithFields(logrus.Fields{
			"component": componentName,
			"severity":  outage.Severity,
			"error":     err,
		}).Error("Failed to create outage in database")
		respondWithError(w, http.StatusInternalServerError, "Failed to create outage")
		return
	}

	h.logger.WithFields(logrus.Fields{
		"outage_id":  outage.ID,
		"component":  componentName,
		"severity":   outage.Severity,
		"created_by": outage.CreatedBy,
	}).Info("Successfully created outage")

	respondWithJSON(w, http.StatusCreated, outage)
}
