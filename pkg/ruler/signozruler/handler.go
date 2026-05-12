package signozruler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/http/binding"
	"github.com/SigNoz/signoz/pkg/http/render"
	"github.com/SigNoz/signoz/pkg/ruler"
	"github.com/SigNoz/signoz/pkg/types/authtypes"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/SigNoz/signoz/pkg/valuer"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type handler struct {
	ruler                   ruler.Ruler
	managedMarkdownDisabled atomic.Bool
	sopDocumentsMu          sync.RWMutex
	sopDocuments            map[string]ruletypes.SOPDocument
}

func NewHandler(ruler ruler.Ruler) ruler.Handler {
	return &handler{ruler: ruler}
}

func (handler *handler) ListRules(rw http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
	defer cancel()

	rules, err := handler.ruler.ListRuleStates(ctx)
	if err != nil {
		render.Error(rw, err)
		return
	}

	view := make([]*ruletypes.Rule, 0, len(rules.Rules))
	for _, rule := range rules.Rules {
		view = append(view, ruletypes.NewRule(rule))
	}

	render.Success(rw, http.StatusOK, view)
}

func (handler *handler) GetRuleByID(rw http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
	defer cancel()

	id, err := valuer.NewUUID(mux.Vars(req)["id"])
	if err != nil {
		render.Error(rw, errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "id is not a valid uuid-v7"))
		return
	}

	rule, err := handler.ruler.GetRule(ctx, id)
	if err != nil {
		render.Error(rw, err)
		return
	}

	render.Success(rw, http.StatusOK, ruletypes.NewRule(rule))
}

func (handler *handler) CreateRule(rw http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
	defer cancel()

	body, err := io.ReadAll(req.Body)
	if err != nil {
		render.Error(rw, err)
		return
	}
	defer req.Body.Close() //nolint:errcheck

	rule, err := handler.ruler.CreateRule(ctx, string(body))
	if err != nil {
		render.Error(rw, err)
		return
	}

	render.Success(rw, http.StatusCreated, ruletypes.NewRule(rule))
}

func (handler *handler) UpdateRuleByID(rw http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
	defer cancel()

	id, err := valuer.NewUUID(mux.Vars(req)["id"])
	if err != nil {
		render.Error(rw, errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "id is not a valid uuid-v7"))
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		render.Error(rw, err)
		return
	}
	defer req.Body.Close() //nolint:errcheck

	err = handler.ruler.EditRule(ctx, string(body), id)
	if err != nil {
		render.Error(rw, err)
		return
	}

	render.Success(rw, http.StatusNoContent, nil)
}

func (handler *handler) DeleteRuleByID(rw http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
	defer cancel()

	id, err := valuer.NewUUID(mux.Vars(req)["id"])
	if err != nil {
		render.Error(rw, errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "id is not a valid uuid-v7"))
		return
	}

	err = handler.ruler.DeleteRule(ctx, id.StringValue())
	if err != nil {
		render.Error(rw, err)
		return
	}

	render.Success(rw, http.StatusNoContent, nil)
}

func (handler *handler) PatchRuleByID(rw http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
	defer cancel()

	id, err := valuer.NewUUID(mux.Vars(req)["id"])
	if err != nil {
		render.Error(rw, errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "id is not a valid uuid-v7"))
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		render.Error(rw, err)
		return
	}
	defer req.Body.Close() //nolint:errcheck

	rule, err := handler.ruler.PatchRule(ctx, string(body), id)
	if err != nil {
		render.Error(rw, err)
		return
	}

	render.Success(rw, http.StatusOK, ruletypes.NewRule(rule))
}

func (handler *handler) TestRule(rw http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 1*time.Minute)
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

	body, err := io.ReadAll(req.Body)
	if err != nil {
		render.Error(rw, err)
		return
	}
	defer req.Body.Close() //nolint:errcheck

	alertCount, err := handler.ruler.TestNotification(ctx, orgID, string(body))
	if err != nil {
		render.Error(rw, err)
		return
	}

	render.Success(rw, http.StatusOK, ruletypes.GettableTestRule{AlertCount: alertCount, Message: "notification sent"})
}

func (handler *handler) PreviewNotificationTemplate(rw http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
	defer cancel()

	var previewReq ruletypes.PreviewNotificationTemplateRequest
	if err := binding.JSON.BindBody(req.Body, &previewReq); err != nil {
		render.Error(rw, err)
		return
	}
	defer req.Body.Close() //nolint:errcheck

	preview, err := ruletypes.PreviewNotificationTemplate(ctx, previewReq)
	if err != nil {
		render.Error(rw, err)
		return
	}

	render.Success(rw, http.StatusOK, preview)
}

func (handler *handler) PreviewSOP(rw http.ResponseWriter, req *http.Request) {
	var previewReq ruletypes.PreviewSOPRequest
	if err := binding.JSON.BindBody(req.Body, &previewReq); err != nil {
		render.Error(rw, err)
		return
	}
	defer req.Body.Close() //nolint:errcheck

	render.Success(rw, http.StatusOK, ruletypes.PreviewSOP(previewReq))
}

// SetManagedMarkdownDisabled toggles the in-process managed_markdown rollback flag.
//
// This handler-level toggle is intentionally separate from
// ruletypes.PilotConfiguration.Enabled (the contract-level config field) for
// PoC scope: wiring the handler to read PilotConfiguration at request time
// requires a config-provider scaffold that will land in Phase 4 cockpit work.
// For now, operators flip this flag via a future admin endpoint or directly
// in tests; the contract-level field remains the canonical schema and will
// unify the toggle path once the cockpit ships.
func (handler *handler) SetManagedMarkdownDisabled(v bool) {
	handler.managedMarkdownDisabled.Store(v)
}

func (handler *handler) FetchPilotManagedMarkdownSOP(rw http.ResponseWriter, req *http.Request) {
	if handler.managedMarkdownDisabled.Load() {
		zap.L().Info("managed markdown SOP fetch rejected — administratively disabled",
			zap.String("path", req.URL.Path),
			zap.String("remote", req.RemoteAddr))
		http.Error(rw, "managed markdown SOP fetch is administratively disabled", http.StatusServiceUnavailable)
		return
	}

	var fetchReq ruletypes.PilotManagedMarkdownSOPFetchRequest
	if err := binding.JSON.BindBody(req.Body, &fetchReq); err != nil {
		render.Error(rw, err)
		return
	}
	defer req.Body.Close() //nolint:errcheck

	resp, err := ruletypes.FetchPilotManagedMarkdownSOP(fetchReq.Source, fetchReq.Fetch)
	if err != nil {
		render.Error(rw, errors.WrapInvalidInputf(err, errors.CodeInvalidInput, "pilot managed markdown SOP fetch validation failed"))
		return
	}

	render.Success(rw, http.StatusOK, resp)
}

func (handler *handler) ListDowntimeSchedules(rw http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
	defer cancel()

	claims, err := authtypes.ClaimsFromContext(ctx)
	if err != nil {
		render.Error(rw, err)
		return
	}

	var params ruletypes.ListPlannedMaintenanceParams
	if err := binding.Query.BindQuery(req.URL.Query(), &params); err != nil {
		render.Error(rw, err)
		return
	}

	schedules, err := handler.ruler.MaintenanceStore().ListPlannedMaintenance(ctx, claims.OrgID)
	if err != nil {
		render.Error(rw, err)
		return
	}

	if params.Active != nil {
		activeSchedules := make([]*ruletypes.PlannedMaintenance, 0)
		for _, schedule := range schedules {
			now := time.Now().In(time.FixedZone(schedule.Schedule.Timezone, 0))
			if schedule.IsActive(now) == *params.Active {
				activeSchedules = append(activeSchedules, schedule)
			}
		}
		schedules = activeSchedules
	}

	if params.Recurring != nil {
		recurringSchedules := make([]*ruletypes.PlannedMaintenance, 0)
		for _, schedule := range schedules {
			if schedule.IsRecurring() == *params.Recurring {
				recurringSchedules = append(recurringSchedules, schedule)
			}
		}
		schedules = recurringSchedules
	}

	render.Success(rw, http.StatusOK, schedules)
}

func (handler *handler) GetDowntimeScheduleByID(rw http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
	defer cancel()

	id, err := valuer.NewUUID(mux.Vars(req)["id"])
	if err != nil {
		render.Error(rw, errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "id is not a valid uuid-v7"))
		return
	}

	schedule, err := handler.ruler.MaintenanceStore().GetPlannedMaintenanceByID(ctx, id)
	if err != nil {
		render.Error(rw, err)
		return
	}

	render.Success(rw, http.StatusOK, schedule)
}

func (handler *handler) CreateDowntimeSchedule(rw http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
	defer cancel()

	schedule := new(ruletypes.PostablePlannedMaintenance)
	if err := binding.JSON.BindBody(req.Body, schedule); err != nil {
		render.Error(rw, err)
		return
	}

	if err := schedule.Validate(); err != nil {
		render.Error(rw, err)
		return
	}

	created, err := handler.ruler.MaintenanceStore().CreatePlannedMaintenance(ctx, schedule)
	if err != nil {
		render.Error(rw, err)
		return
	}

	render.Success(rw, http.StatusCreated, created)
}

func (handler *handler) UpdateDowntimeScheduleByID(rw http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
	defer cancel()

	id, err := valuer.NewUUID(mux.Vars(req)["id"])
	if err != nil {
		render.Error(rw, errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "id is not a valid uuid-v7"))
		return
	}

	schedule := new(ruletypes.PostablePlannedMaintenance)
	if err := binding.JSON.BindBody(req.Body, schedule); err != nil {
		render.Error(rw, err)
		return
	}

	if err := schedule.Validate(); err != nil {
		render.Error(rw, err)
		return
	}

	err = handler.ruler.MaintenanceStore().UpdatePlannedMaintenance(ctx, schedule, id)
	if err != nil {
		render.Error(rw, err)
		return
	}

	render.Success(rw, http.StatusNoContent, nil)
}

func (handler *handler) DeleteDowntimeScheduleByID(rw http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
	defer cancel()

	id, err := valuer.NewUUID(mux.Vars(req)["id"])
	if err != nil {
		render.Error(rw, errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "id is not a valid uuid-v7"))
		return
	}

	err = handler.ruler.MaintenanceStore().DeletePlannedMaintenance(ctx, id)
	if err != nil {
		render.Error(rw, err)
		return
	}

	render.Success(rw, http.StatusNoContent, nil)
}

const pilotManagedMarkdownDefaultSourceID = "src-managed-markdown-default"

// pilotManagedMarkdownDefaultSource returns the canonical live managed_markdown
// catalog entry that matches what FetchPilotManagedMarkdownSOP serves. The
// constructor returns a fresh struct on every call so handlers cannot share
// mutable package-level state.
func pilotManagedMarkdownDefaultSource() ruletypes.PilotManagedMarkdownSource {
	return ruletypes.PilotManagedMarkdownSource{
		SourceID:              pilotManagedMarkdownDefaultSourceID,
		DisplayName:           "Managed Markdown SOP Registry",
		Status:                ruletypes.PilotSOPSourceStatusHealthy,
		ServiceAccountProfile: "ds-sop-reader",
	}
}

func (handler *handler) ListPilotSOPSources(rw http.ResponseWriter, req *http.Request) {
	resp, err := ruletypes.NewPilotManagedMarkdownCatalog([]ruletypes.PilotManagedMarkdownSource{
		pilotManagedMarkdownDefaultSource(),
	})
	if err != nil {
		zap.L().Error("pilot sop source catalog validation failed", zap.Error(err))
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(rw).Encode(resp); err != nil {
		zap.L().Warn("pilot sop source catalog encode failed", zap.Error(err))
	}
}

func (handler *handler) GetPilotSOPSourceHealth(rw http.ResponseWriter, req *http.Request) {
	id := strings.TrimSpace(mux.Vars(req)["id"])
	if id == "" {
		http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if id != pilotManagedMarkdownDefaultSourceID {
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	checkedAt := time.Now().UTC().Format(time.RFC3339)
	resp, err := ruletypes.NewPilotManagedMarkdownHealth(pilotManagedMarkdownDefaultSource(), checkedAt)
	if err != nil {
		zap.L().Error("pilot sop source health validation failed", zap.String("sourceId", id), zap.Error(err))
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(rw).Encode(resp); err != nil {
		zap.L().Warn("pilot sop source health encode failed", zap.String("sourceId", id), zap.Error(err))
	}
}

func (handler *handler) CreateSOPDocument(rw http.ResponseWriter, req *http.Request) {
	var doc ruletypes.SOPDocument
	if err := binding.JSON.BindBody(req.Body, &doc); err != nil {
		render.Error(rw, err)
		return
	}
	defer req.Body.Close() //nolint:errcheck

	if err := ruletypes.ValidateSOPDocument(doc); err != nil {
		render.Error(rw, errors.WrapInvalidInputf(err, errors.CodeInvalidInput, "SOP document validation failed"))
		return
	}

	handler.storeSOPDocument(doc)
	render.Success(rw, http.StatusCreated, doc)
}

func (handler *handler) ListSOPDocuments(rw http.ResponseWriter, req *http.Request) {
	render.Success(rw, http.StatusOK, ruletypes.NewSOPDocumentListResponse(handler.snapshotSOPDocuments()))
}

func (handler *handler) GetSOPDocument(rw http.ResponseWriter, req *http.Request) {
	sopID := strings.TrimSpace(mux.Vars(req)["sopId"])
	if sopID == "" {
		http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	doc, ok := handler.latestSOPDocument(sopID)
	if !ok {
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	render.Success(rw, http.StatusOK, doc)
}

func (handler *handler) FetchSOPDocumentVersion(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sopID := strings.TrimSpace(vars["sopId"])
	version := strings.TrimSpace(vars["version"])
	if sopID == "" || version == "" {
		http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	doc, ok := handler.sopDocument(sopID, version)
	if !ok {
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	render.Success(rw, http.StatusOK, doc)
}

func (handler *handler) PreviewSOPDocumentBinding(rw http.ResponseWriter, req *http.Request) {
	var previewReq ruletypes.SOPBindingPreviewRequest
	if err := binding.JSON.BindBody(req.Body, &previewReq); err != nil {
		render.Error(rw, err)
		return
	}
	defer req.Body.Close() //nolint:errcheck

	resp, err := ruletypes.PreviewSOPDocumentBinding(handler.snapshotSOPDocuments(), previewReq)
	if err != nil {
		render.Error(rw, errors.WrapInvalidInputf(err, errors.CodeInvalidInput, "SOP binding preview validation failed"))
		return
	}

	render.Success(rw, http.StatusOK, resp)
}

func (handler *handler) storeSOPDocument(doc ruletypes.SOPDocument) {
	handler.sopDocumentsMu.Lock()
	defer handler.sopDocumentsMu.Unlock()

	if handler.sopDocuments == nil {
		handler.sopDocuments = map[string]ruletypes.SOPDocument{}
	}
	handler.sopDocuments[sopDocumentKey(doc.SOPID, doc.Version)] = doc
}

func (handler *handler) snapshotSOPDocuments() []ruletypes.SOPDocument {
	handler.sopDocumentsMu.RLock()
	defer handler.sopDocumentsMu.RUnlock()

	docs := make([]ruletypes.SOPDocument, 0, len(handler.sopDocuments))
	for _, doc := range handler.sopDocuments {
		docs = append(docs, doc)
	}
	sort.Slice(docs, func(i, j int) bool {
		if docs[i].SOPID == docs[j].SOPID {
			return docs[i].Version < docs[j].Version
		}
		return docs[i].SOPID < docs[j].SOPID
	})

	return docs
}

func (handler *handler) latestSOPDocument(sopID string) (ruletypes.SOPDocument, bool) {
	var latest ruletypes.SOPDocument
	found := false
	for _, doc := range handler.snapshotSOPDocuments() {
		if doc.SOPID != sopID {
			continue
		}
		if !found || doc.Version > latest.Version {
			latest = doc
			found = true
		}
	}
	return latest, found
}

func (handler *handler) sopDocument(sopID string, version string) (ruletypes.SOPDocument, bool) {
	handler.sopDocumentsMu.RLock()
	defer handler.sopDocumentsMu.RUnlock()

	doc, ok := handler.sopDocuments[sopDocumentKey(sopID, version)]
	return doc, ok
}

func sopDocumentKey(sopID string, version string) string {
	return strings.TrimSpace(sopID) + "\x00" + strings.TrimSpace(version)
}
