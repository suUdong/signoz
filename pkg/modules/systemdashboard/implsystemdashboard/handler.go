package implsystemdashboard

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	
	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/http/render"
	"github.com/SigNoz/signoz/pkg/modules/systemdashboard"
	"github.com/SigNoz/signoz/pkg/types/authtypes"
	"github.com/SigNoz/signoz/pkg/types/dashboardtypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

type handler struct {
	module systemdashboard.Module
}

func NewHandler(module systemdashboard.Module) systemdashboard.Handler {
	return &handler{module: module}
}

func (handler *handler) Get(rw http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	claims, err := authtypes.ClaimsFromContext(ctx)
	if err != nil {
		render.Error(rw, err)
		return
	}

	orgID, err := valuer.NewUUID(claims.OrgID)
	if err != nil {
		render.Error(rw, err)
		return
	}

	source, err := parseSource(r)
	if err != nil {
		render.Error(rw, err)
		return
	}

	dashboard, err := handler.module.Get(ctx, orgID, source)
	if err != nil {
		render.Error(rw, err)
		return
	}

	render.Success(rw, http.StatusOK, dashboardtypes.NewGettableSystemDashboardFromDashboard(dashboard))
}

func (handler *handler) Update(rw http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	claims, err := authtypes.ClaimsFromContext(ctx)
	if err != nil {
		render.Error(rw, err)
		return
	}

	orgID, err := valuer.NewUUID(claims.OrgID)
	if err != nil {
		render.Error(rw, err)
		return
	}

	source, err := parseSource(r)
	if err != nil {
		render.Error(rw, err)
		return
	}

	req := dashboardtypes.UpdatableSystemDashboard{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Error(rw, errors.Wrapf(err, errors.TypeInvalidInput, errors.CodeInvalidInput, "invalid request body"))
		return
	}

	if req.Data == nil {
		render.Error(rw, errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "data is required"))
		return
	}

	dashboard, err := handler.module.Update(ctx, orgID, source, claims.Email, req.Data)
	if err != nil {
		render.Error(rw, err)
		return
	}

	render.Success(rw, http.StatusOK, dashboardtypes.NewGettableSystemDashboardFromDashboard(dashboard))
}

func (handler *handler) Reset(rw http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	claims, err := authtypes.ClaimsFromContext(ctx)
	if err != nil {
		render.Error(rw, err)
		return
	}

	orgID, err := valuer.NewUUID(claims.OrgID)
	if err != nil {
		render.Error(rw, err)
		return
	}

	source, err := parseSource(r)
	if err != nil {
		render.Error(rw, err)
		return
	}

	dashboard, err := handler.module.Reset(ctx, orgID, source)
	if err != nil {
		render.Error(rw, err)
		return
	}

	render.Success(rw, http.StatusOK, dashboardtypes.NewGettableSystemDashboardFromDashboard(dashboard))
}

func parseSource(r *http.Request) (dashboardtypes.Source, error) {
	source := mux.Vars(r)["source"]
	if source == "" {
		return dashboardtypes.Source{}, errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "source is missing in the path")
	}

	return dashboardtypes.NewSource(source)
}
