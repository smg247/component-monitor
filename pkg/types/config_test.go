package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComponent_GetSubComponent(t *testing.T) {
	tests := []struct {
		name             string
		component        Component
		subComponentName string
		expectedResult   *SubComponent
	}{
		{
			name: "subcomponent found - single subcomponent",
			component: Component{
				Name: "Prow",
				Subcomponents: []SubComponent{
					{Name: "Tide", Description: "Merge bot"},
				},
			},
			subComponentName: "Tide",
			expectedResult:   &SubComponent{Name: "Tide", Description: "Merge bot"},
		},
		{
			name: "subcomponent found - multiple subcomponents",
			component: Component{
				Name: "Prow",
				Subcomponents: []SubComponent{
					{Name: "Tide", Description: "Merge bot"},
					{Name: "Deck", Description: "Dashboard"},
					{Name: "Hook", Description: "Webhook handler"},
				},
			},
			subComponentName: "Deck",
			expectedResult:   &SubComponent{Name: "Deck", Description: "Dashboard"},
		},
		{
			name: "subcomponent not found",
			component: Component{
				Name: "Prow",
				Subcomponents: []SubComponent{
					{Name: "Tide", Description: "Merge bot"},
					{Name: "Deck", Description: "Dashboard"},
				},
			},
			subComponentName: "NonExistent",
			expectedResult:   nil,
		},
		{
			name: "empty subcomponents list",
			component: Component{
				Name:          "Prow",
				Subcomponents: []SubComponent{},
			},
			subComponentName: "AnySubComponent",
			expectedResult:   nil,
		},
		{
			name: "case sensitive matching",
			component: Component{
				Name: "Prow",
				Subcomponents: []SubComponent{
					{Name: "Tide", Description: "Merge bot"},
				},
			},
			subComponentName: "tide",
			expectedResult:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.component.GetSubComponent(tt.subComponentName)

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
