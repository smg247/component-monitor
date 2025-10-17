package main

import (
	"testing"

	"ship-status-dash/pkg/types"

	"github.com/stretchr/testify/assert"
)

func TestDetermineStatusFromSeverity(t *testing.T) {
	tests := []struct {
		name     string
		outages  []types.Outage
		expected types.Status
	}{
		{
			name: "single outage - down severity",
			outages: []types.Outage{
				{Severity: types.SeverityDown},
			},
			expected: types.StatusDown,
		},
		{
			name: "multiple outages - highest severity wins",
			outages: []types.Outage{
				{Severity: types.SeveritySuspected},
				{Severity: types.SeverityDown},
				{Severity: types.SeverityDegraded},
			},
			expected: types.StatusDown,
		},
		{
			name: "multiple outages - degraded highest",
			outages: []types.Outage{
				{Severity: types.SeveritySuspected},
				{Severity: types.SeverityDegraded},
				{Severity: types.SeveritySuspected},
			},
			expected: types.StatusDegraded,
		},
		{
			name: "all same severity",
			outages: []types.Outage{
				{Severity: types.SeveritySuspected},
				{Severity: types.SeveritySuspected},
			},
			expected: types.StatusSuspected,
		},
		{
			name:     "empty outages slice",
			outages:  []types.Outage{},
			expected: types.StatusHealthy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineStatusFromSeverity(tt.outages)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetComponent(t *testing.T) {
	tests := []struct {
		name           string
		components     []types.Component
		componentName  string
		expectedResult *types.Component
	}{
		{
			name: "component found - single component",
			components: []types.Component{
				{Name: "Prow", Description: "CI/CD system"},
			},
			componentName:  "Prow",
			expectedResult: &types.Component{Name: "Prow", Description: "CI/CD system"},
		},
		{
			name: "component found - multiple components",
			components: []types.Component{
				{Name: "Prow", Description: "CI/CD system"},
				{Name: "Deck", Description: "Dashboard"},
				{Name: "Tide", Description: "Merge bot"},
			},
			componentName:  "Deck",
			expectedResult: &types.Component{Name: "Deck", Description: "Dashboard"},
		},
		{
			name: "component not found",
			components: []types.Component{
				{Name: "Prow", Description: "CI/CD system"},
				{Name: "Deck", Description: "Dashboard"},
			},
			componentName:  "NonExistent",
			expectedResult: nil,
		},
		{
			name:           "empty components list",
			components:     []types.Component{},
			componentName:  "AnyComponent",
			expectedResult: nil,
		},
		{
			name: "case sensitive matching",
			components: []types.Component{
				{Name: "Prow", Description: "CI/CD system"},
			},
			componentName:  "prow",
			expectedResult: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := &Handlers{
				config: &types.Config{Components: tt.components},
			}

			result := handlers.getComponent(tt.componentName)

			if tt.expectedResult == nil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectedResult.Name, result.Name)
				assert.Equal(t, tt.expectedResult.Description, result.Description)
			}
		})
	}
}
