package implsystemdashboard

import (
	"context"
	"time"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/modules/systemdashboard"
	"github.com/SigNoz/signoz/pkg/types/dashboardtypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

type module struct {
	store dashboardtypes.Store
}

func NewModule(store dashboardtypes.Store) systemdashboard.Module {
	return &module{store: store}
}

func (module *module) Get(ctx context.Context, orgID valuer.UUID, source dashboardtypes.Source) (*dashboardtypes.Dashboard, error) {
	storableDashboard, err := module.store.GetBySource(ctx, orgID, source.StringValue())
	if err != nil {
		return nil, err
	}

	return dashboardtypes.NewDashboardFromStorableDashboard(storableDashboard), nil
}

// Update applies the new payload as last-writer-wins. The Get and Update run inside one transaction so a
// concurrent Reset cannot interleave and leave the response with a stale id from before the reset.
func (module *module) Update(ctx context.Context, orgID valuer.UUID, source dashboardtypes.Source, updatedBy string, data dashboardtypes.UpdatableDashboard) (*dashboardtypes.Dashboard, error) {
	var updated *dashboardtypes.Dashboard
	err := module.store.RunInTx(ctx, func(ctx context.Context) error {
		storableDashboard, err := module.store.GetBySource(ctx, orgID, source.StringValue())
		if err != nil {
			return err
		}

		storableDashboard.Data = data
		storableDashboard.UpdatedBy = updatedBy
		storableDashboard.UpdatedAt = time.Now()

		if err := module.store.UpdateBySource(ctx, orgID, source.StringValue(), storableDashboard); err != nil {
			return err
		}

		updated = dashboardtypes.NewDashboardFromStorableDashboard(storableDashboard)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return updated, nil
}

// Reset drops the org's customized system dashboard for the given source and writes the default back in a single transaction. Returns the freshly seeded dashboard so callers can render it without a follow-up Get. A NotFound from the delete is treated as "nothing to drop, proceed with seed" so Reset is self-healing when a row was lost (e.g. hand-deleted in SQL or a prior partial seed).
func (module *module) Reset(ctx context.Context, orgID valuer.UUID, source dashboardtypes.Source) (*dashboardtypes.Dashboard, error) {
	var resetDashboard *dashboardtypes.Dashboard
	err := module.store.RunInTx(ctx, func(ctx context.Context) error {
		if err := module.store.DeleteBySource(ctx, orgID, source.StringValue()); err != nil && !errors.Ast(err, errors.TypeNotFound) {
			return err
		}

		defaultDashboard, err := dashboardtypes.NewDefaultSystemDashboard(orgID, source)
		if err != nil {
			return err
		}

		storable, err := dashboardtypes.NewStorableDashboardFromDashboard(defaultDashboard)
		if err != nil {
			return err
		}

		if err := module.store.Create(ctx, storable); err != nil {
			return err
		}

		resetDashboard = defaultDashboard
		return nil
	})
	if err != nil {
		return nil, err
	}

	return resetDashboard, nil
}

func (module *module) SetDefaultConfig(ctx context.Context, orgID valuer.UUID) error {
	for _, source := range dashboardtypes.AllSources {
		if err := module.setDefaultForSource(ctx, orgID, source); err != nil {
			return err
		}
	}

	return nil
}

func (module *module) setDefaultForSource(ctx context.Context, orgID valuer.UUID, source dashboardtypes.Source) error {
	existing, err := module.store.GetBySource(ctx, orgID, source.StringValue())
	if err != nil && !errors.Ast(err, errors.TypeNotFound) {
		return err
	}
	if existing != nil {
		return nil
	}

	dashboard, err := dashboardtypes.NewDefaultSystemDashboard(orgID, source)
	if err != nil {
		return err
	}

	storableDashboard, err := dashboardtypes.NewStorableDashboardFromDashboard(dashboard)
	if err != nil {
		return err
	}

	return module.store.Create(ctx, storableDashboard)
}
