package types

import (
	"database/sql"
	"time"

	"gorm.io/gorm"
)

// Outage represents a component outage with tracking information for incident management.
type Outage struct {
	gorm.Model
	ComponentName  string       `json:"component_name" gorm:"column:component_name;not null;index"`
	Severity       string       `json:"severity" gorm:"column:severity;not null"`
	StartTime      time.Time    `json:"start_time" gorm:"column:start_time;not null;index"`
	EndTime        sql.NullTime `json:"end_time" gorm:"column:end_time;index"`
	AutoResolve    bool         `json:"auto_resolve" gorm:"column:auto_resolve;not null;default:true"`
	Description    string       `json:"description" gorm:"column:description;type:text"`
	DiscoveredFrom string       `json:"discovered_from" gorm:"column:discovered_from;not null"`
	CreatedBy      string       `json:"created_by" gorm:"column:created_by;not null"`
	ResolvedBy     *string      `json:"resolved_by,omitempty" gorm:"column:resolved_by"`
	ConfirmedBy    *string      `json:"confirmed_by,omitempty" gorm:"column:confirmed_by"`
	ConfirmedAt    sql.NullTime `json:"confirmed_at" gorm:"column:confirmed_at"`
	TriageNotes    *string      `json:"triage_notes,omitempty" gorm:"column:triage_notes;type:text"`
}

// Config contains the application configuration including component definitions.
type Config struct {
	Components []Component `json:"components" yaml:"components"`
}

// Component represents a top-level system component with sub-components and ownership information.
type Component struct {
	Name          string         `json:"name" yaml:"name"`
	Description   string         `json:"description" yaml:"description"`
	ShipTeam      string         `json:"ship_team" yaml:"ship_team"`
	SlackChannel  string         `json:"slack_channel" yaml:"slack_channel"`
	Subcomponents []SubComponent `json:"sub_components" yaml:"sub_components"`
	Owners        []Owner        `json:"owners" yaml:"owners"`
}

// SubComponent represents a sub-component that can have outages tracked against it.
type SubComponent struct {
	Name                 string `json:"name" yaml:"name"`
	Description          string `json:"description" yaml:"description"`
	Managed              bool   `json:"managed" yaml:"managed"`
	RequiresConfirmation bool   `json:"requires_confirmation" yaml:"requires_confirmation"`
}

// Owner represents ownership information for a component, either via Rover group or service account.
type Owner struct {
	RoverGroup     string `json:"rover_group,omitempty" yaml:"rover_group,omitempty"`
	ServiceAccount string `json:"service_account,omitempty" yaml:"service_account,omitempty"`
}
