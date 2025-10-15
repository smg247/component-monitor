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

type Handlers struct {
	logger *logrus.Logger
	config *types.Config
	db     *gorm.DB
}

func NewHandlers(logger *logrus.Logger, config *types.Config, db *gorm.DB) *Handlers {
	return &Handlers{
		logger: logger,
		config: config,
		db:     db,
	}
}

func respondWithJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
}

func respondWithError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

func (h *Handlers) componentExists(componentName string) bool {
	for _, component := range h.config.Components {
		if component.Name == componentName {
			return true
		}
		for _, subComponent := range component.Subcomponents {
			if subComponent.Name == componentName {
				return true
			}
		}
	}
	return false
}

func (h *Handlers) isTopLevelComponent(componentName string) bool {
	for _, component := range h.config.Components {
		if component.Name == componentName {
			return true
		}
	}
	return false
}

func (h *Handlers) getSubComponentNames(topLevelComponentName string) []string {
	for _, component := range h.config.Components {
		if component.Name == topLevelComponentName {
			names := make([]string, len(component.Subcomponents))
			for i, sub := range component.Subcomponents {
				names[i] = sub.Name
			}
			return names
		}
	}
	return nil
}

func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	}
	respondWithJSON(w, response)
}

func (h *Handlers) GetComponents(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, h.config.Components)
}

func (h *Handlers) GetOutages(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	componentName := vars["componentName"]

	if !h.componentExists(componentName) {
		respondWithError(w, http.StatusNotFound, "Component not found")
		return
	}

	var outages []types.Outage

	// If it's a top-level component, get outages from all sub-components
	if h.isTopLevelComponent(componentName) {
		subComponents := h.getSubComponentNames(componentName)
		if len(subComponents) == 0 {
			// No sub-components, return empty array
			respondWithJSON(w, []types.Outage{})
			return
		}

		if err := h.db.Where("component_name IN ?", subComponents).Order("start_time DESC").Find(&outages).Error; err != nil {
			h.logger.Errorf("Failed to get outages: %v", err)
			respondWithError(w, http.StatusInternalServerError, "Failed to get outages")
			return
		}
	} else {
		// Sub-component, get outages directly
		if err := h.db.Where("component_name = ?", componentName).Order("start_time DESC").Find(&outages).Error; err != nil {
			h.logger.Errorf("Failed to get outages: %v", err)
			respondWithError(w, http.StatusInternalServerError, "Failed to get outages")
			return
		}
	}

	respondWithJSON(w, outages)
}

func (h *Handlers) CreateOutage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	componentName := vars["componentName"]

	if !h.componentExists(componentName) {
		respondWithError(w, http.StatusNotFound, "Component not found")
		return
	}

	// Only allow creating outages for sub-components
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

	// Validate required fields
	if outage.Severity == "" {
		respondWithError(w, http.StatusBadRequest, "Severity is required")
		return
	}
	if outage.StartTime.IsZero() {
		respondWithError(w, http.StatusBadRequest, "StartTime is required")
		return
	}
	if outage.DiscoveredFrom == "" {
		respondWithError(w, http.StatusBadRequest, "DiscoveredFrom is required")
		return
	}
	if outage.CreatedBy == "" {
		respondWithError(w, http.StatusBadRequest, "CreatedBy is required")
		return
	}

	// Create outage in database
	if err := h.db.Create(&outage).Error; err != nil {
		h.logger.Errorf("Failed to create outage: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to create outage")
		return
	}

	h.logger.Infof("Created outage %d for component %s", outage.ID, componentName)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(outage)
}
