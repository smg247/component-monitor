package types

type Status string

const (
	StatusHealthy   Status = "Healthy"
	StatusDegraded  Status = "Degraded"
	StatusDown      Status = "Down"
	StatusSuspected Status = "Suspected"
	StatusPartial   Status = "Partial" // Indicates that some sub-components are healthy, and some are degraded or down
)

type ComponentStatus struct {
	Status        Status   `json:"status"`
	ActiveOutages []Outage `json:"active_outages"`
}
