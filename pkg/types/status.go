package types

type Status string

const (
	StatusHealthy   Status = "Healthy"
	StatusDegraded  Status = "Degraded"
	StatusDown      Status = "Down"
	StatusSuspected Status = "Suspected"
)

type ComponentStatus struct {
	Status        Status   `json:"status"`
	ActiveOutages []Outage `json:"active_outages"`
}
