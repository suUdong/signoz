package dashboardtypes

import (
	"time"

	"github.com/SigNoz/signoz/pkg/valuer"
)

type GettableSystemDashboard struct {
	ID        string                `json:"id"`
	OrgID     valuer.UUID           `json:"org_id"`
	Source    string                `json:"source"`
	Data      StorableDashboardData `json:"data"`
	CreatedAt time.Time             `json:"createdAt"`
	CreatedBy string                `json:"createdBy"`
	UpdatedAt time.Time             `json:"updatedAt"`
	UpdatedBy string                `json:"updatedBy"`
}

type UpdatableSystemDashboard struct {
	Data StorableDashboardData `json:"data"`
}

func NewGettableSystemDashboardFromDashboard(dashboard *Dashboard) *GettableSystemDashboard {
	return &GettableSystemDashboard{
		ID:        dashboard.ID,
		OrgID:     dashboard.OrgID,
		Source:    dashboard.Source,
		Data:      dashboard.Data,
		CreatedAt: dashboard.CreatedAt,
		CreatedBy: dashboard.CreatedBy,
		UpdatedAt: dashboard.UpdatedAt,
		UpdatedBy: dashboard.UpdatedBy,
	}
}
