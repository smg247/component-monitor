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
