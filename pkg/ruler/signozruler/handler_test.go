package signozruler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/SigNoz/signoz/pkg/types/authtypes"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

type recordingPilotAuditSink struct {
	mu     sync.Mutex
	events []ruletypes.PilotAuditEvent
	err    error
}

func (s *recordingPilotAuditSink) Record(_ context.Context, event ruletypes.PilotAuditEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	return s.err
}

func (s *recordingPilotAuditSink) Events() []ruletypes.PilotAuditEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]ruletypes.PilotAuditEvent, len(s.events))
	copy(cp, s.events)
	return cp
}

// validPilotManagedMarkdownSOPFetchRequestBody returns the canonical request
// body fixture used by both happy-path and disable-flag tests. Factored out so
// contract changes only need to update one location.
func validPilotManagedMarkdownSOPFetchRequestBody(t *testing.T) []byte {
	t.Helper()
	body, err := json.Marshal(ruletypes.PilotManagedMarkdownSOPFetchRequest{
		Source: ruletypes.PilotManagedMarkdownSource{
			SourceID:              "src-managed-markdown-default",
			DisplayName:           "Managed Markdown SOP Registry",
			Status:                ruletypes.PilotSOPSourceStatusHealthy,
			LastHealthCheckAt:     "2026-04-30T00:00:00Z",
			LastSyncAt:            "2026-04-30T00:00:00Z",
			ServiceAccountProfile: "ds-sop-reader",
			TenantScope: ruletypes.PilotTenantScope{
				ProjectIDs:   []string{"customer-a"},
				Environments: []string{"prod"},
			},
			Documents: []ruletypes.PilotManagedMarkdownDocument{
				{
					SOPID:        "SOP-PAY-001",
					Version:      "2026-04-20.3",
					Title:        "Payment API 5xx response",
					BodyMarkdown: "Restart payment-api only after confirming queue drain.",
					DisplayURL:   "https://kb.example/sop/SOP-PAY-001",
				},
			},
		},
		Fetch: ruletypes.PilotSOPFetchRequest{
			SourceID:              "src-managed-markdown-default",
			SOPID:                 "SOP-PAY-001",
			Version:               "2026-04-20.3",
			OccurredAt:            "2026-04-30T00:00:00Z",
			AuditEventID:          "audit-20260430-000001",
			AuditMode:             ruletypes.PilotAuditModeRequired,
			AuditAccepted:         true,
			ServiceAccountProfile: "ds-sop-reader",
			Actor: ruletypes.PilotAuditActor{
				Kind: ruletypes.PilotAuditActorKindUser,
				ID:   "user-123",
			},
			Tenant: ruletypes.PilotAuditTenant{
				ProjectID:   "customer-a",
				Environment: "prod",
			},
			RequestContext: ruletypes.PilotAuditRequestContext{
				IncidentID:  "INC-20260430-001",
				ServiceName: "payment-api",
			},
		},
	})
	require.NoError(t, err)
	return body
}

func TestFetchPilotManagedMarkdownSOPHandler(t *testing.T) {
	body := validPilotManagedMarkdownSOPFetchRequestBody(t)

	rw := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v2/rules/sop/pilot/managed_markdown/fetch", bytes.NewReader(body))

	(&handler{}).FetchPilotManagedMarkdownSOP(rw, req)

	require.Equal(t, http.StatusOK, rw.Code)
	var got struct {
		Status string                          `json:"status"`
		Data   ruletypes.PilotSOPFetchResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rw.Body.Bytes(), &got))
	require.Equal(t, "success", got.Status)
	require.Equal(t, ruletypes.PilotSOPFetchStatusFetched, got.Data.Status)
	require.Equal(t, "SOP-PAY-001", got.Data.SOPID)
	require.Contains(t, got.Data.BodyMarkdown, "Restart payment-api")
	require.False(t, got.Data.SecurityContext.BrowserCredentialsUsed)
	require.False(t, got.Data.SecurityContext.SecretRefVisible)
}

func TestFetchPilotManagedMarkdownSOPHandlerDispatchesAuditEvent(t *testing.T) {
	ruletypes.RegisterPilotAuditEventSink(nil)
	t.Cleanup(func() { ruletypes.RegisterPilotAuditEventSink(nil) })

	recorder := &recordingPilotAuditSink{}
	ruletypes.RegisterPilotAuditEventSink(recorder)

	body := validPilotManagedMarkdownSOPFetchRequestBody(t)
	rw := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v2/rules/sop/pilot/managed_markdown/fetch", bytes.NewReader(body))

	(&handler{}).FetchPilotManagedMarkdownSOP(rw, req)

	require.Equal(t, http.StatusOK, rw.Code)
	events := recorder.Events()
	require.Len(t, events, 1)
	require.Equal(t, "audit-20260430-000001", events[0].EventID)
	require.Equal(t, ruletypes.PilotAuditOutcomeAllowed, events[0].Outcome)
	require.Equal(t, "INC-20260430-001", events[0].RequestContext.IncidentID)
	require.Equal(t, "payment-api", events[0].RequestContext.ServiceName)
}

func TestFetchPilotManagedMarkdownSOPHandlerAuditSinkFailureIsFailOpen(t *testing.T) {
	ruletypes.RegisterPilotAuditEventSink(nil)
	t.Cleanup(func() { ruletypes.RegisterPilotAuditEventSink(nil) })

	recorder := &recordingPilotAuditSink{err: errors.New("audit sink unavailable")}
	ruletypes.RegisterPilotAuditEventSink(recorder)

	body := validPilotManagedMarkdownSOPFetchRequestBody(t)
	rw := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v2/rules/sop/pilot/managed_markdown/fetch", bytes.NewReader(body))

	(&handler{}).FetchPilotManagedMarkdownSOP(rw, req)

	require.Equal(t, http.StatusOK, rw.Code)
	require.Len(t, recorder.Events(), 1)
}

func TestFetchPilotManagedMarkdownSOPHandlerDisableFlag(t *testing.T) {
	body := validPilotManagedMarkdownSOPFetchRequestBody(t)

	h := &handler{}

	// 1) flag default false → fetch reaches the body and returns 200
	rw := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v2/rules/sop/pilot/managed_markdown/fetch", bytes.NewReader(body))
	h.FetchPilotManagedMarkdownSOP(rw, req)
	require.Equal(t, http.StatusOK, rw.Code)

	// 2) flip flag → next fetch returns 503 with no body delegation
	h.SetManagedMarkdownDisabled(true)
	rw2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/v2/rules/sop/pilot/managed_markdown/fetch", bytes.NewReader(body))
	h.FetchPilotManagedMarkdownSOP(rw2, req2)
	require.Equal(t, http.StatusServiceUnavailable, rw2.Code)
}

func TestFetchPilotManagedMarkdownSOPHandlerReturnsInvalidInput(t *testing.T) {
	rw := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v2/rules/sop/pilot/managed_markdown/fetch", bytes.NewReader([]byte(`{}`)))

	(&handler{}).FetchPilotManagedMarkdownSOP(rw, req)

	require.Equal(t, http.StatusBadRequest, rw.Code)
}

func TestListPilotSOPSources_HappyPath(t *testing.T) {
	rw := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v2/ds/sop/sources", nil)

	(&handler{}).ListPilotSOPSources(rw, req)

	require.Equal(t, http.StatusOK, rw.Code)

	var got ruletypes.PilotSOPSourceCatalogResponse
	require.NoError(t, json.Unmarshal(rw.Body.Bytes(), &got))
	require.NoError(t, ruletypes.ValidatePilotSOPSourceCatalog(got))
	require.NotEmpty(t, got.Sources, "at least one source must be present")
	require.NotEmpty(t, got.Sources[0].SourceID, "source id must be non-empty")
}

func TestGetPilotSOPSourceHealth_HappyPath(t *testing.T) {
	rw := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v2/ds/sop/sources/src-managed-markdown-default/health", nil)
	req = muxSetVar(req, "id", "src-managed-markdown-default")

	(&handler{}).GetPilotSOPSourceHealth(rw, req)

	require.Equal(t, http.StatusOK, rw.Code)

	var got ruletypes.PilotSOPSourceHealthResponse
	require.NoError(t, json.Unmarshal(rw.Body.Bytes(), &got))
	require.NoError(t, ruletypes.ValidatePilotSOPSourceHealth(got))
}

func TestGetPilotSOPSourceHealth_UnknownID(t *testing.T) {
	rw := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v2/ds/sop/sources/does-not-exist/health", nil)
	req = muxSetVar(req, "id", "does-not-exist")

	(&handler{}).GetPilotSOPSourceHealth(rw, req)

	require.Equal(t, http.StatusNotFound, rw.Code)
	require.Empty(t, rw.Body.Bytes())
}

func TestSOPDocumentHandlersCreateListGetFetchAndBind(t *testing.T) {
	h := &handler{}
	body := validSOPDocumentRequestBody(t, "2026-05-12.1", ruletypes.SOPApprovalStatusApproved)

	createRW := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/v2/ds/sop/documents", bytes.NewReader(body))
	createReq = withSOPTestClaims(createReq)
	h.CreateSOPDocument(createRW, createReq)

	require.Equal(t, http.StatusCreated, createRW.Code)
	var created struct {
		Status string                `json:"status"`
		Data   ruletypes.SOPDocument `json:"data"`
	}
	require.NoError(t, json.Unmarshal(createRW.Body.Bytes(), &created))
	require.Equal(t, "success", created.Status)
	require.Equal(t, "SOP-PAY-001", created.Data.SOPID)
	require.Equal(t, "2026-05-12.1", created.Data.Version)

	listRW := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/v2/ds/sop/documents", nil)
	listReq = withSOPTestClaims(listReq)
	h.ListSOPDocuments(listRW, listReq)

	require.Equal(t, http.StatusOK, listRW.Code)
	require.NotContains(t, listRW.Body.String(), "bodyMarkdown")
	var listed struct {
		Status string                            `json:"status"`
		Data   ruletypes.SOPDocumentListResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(listRW.Body.Bytes(), &listed))
	require.Equal(t, ruletypes.SOPDocumentListContractVersion, listed.Data.ContractVersion)
	require.Len(t, listed.Data.Documents, 1)

	getRW := httptest.NewRecorder()
	getReq := httptest.NewRequest(http.MethodGet, "/api/v2/ds/sop/documents/SOP-PAY-001", nil)
	getReq = withSOPTestClaims(getReq)
	getReq = muxSetVar(getReq, "sopId", "SOP-PAY-001")
	h.GetSOPDocument(getRW, getReq)

	require.Equal(t, http.StatusOK, getRW.Code)
	var gotLatest struct {
		Data ruletypes.SOPDocument `json:"data"`
	}
	require.NoError(t, json.Unmarshal(getRW.Body.Bytes(), &gotLatest))
	require.Equal(t, "SOP-PAY-001", gotLatest.Data.SOPID)
	require.Contains(t, gotLatest.Data.BodyMarkdown, "Restart payment-api")

	fetchRW := httptest.NewRecorder()
	fetchReq := httptest.NewRequest(http.MethodGet, "/api/v2/ds/sop/documents/SOP-PAY-001/versions/2026-05-12.1", nil)
	fetchReq = withSOPTestClaims(fetchReq)
	fetchReq = muxSetVar(fetchReq, "sopId", "SOP-PAY-001")
	fetchReq = muxSetVar(fetchReq, "version", "2026-05-12.1")
	h.FetchSOPDocumentVersion(fetchRW, fetchReq)

	require.Equal(t, http.StatusOK, fetchRW.Code)
	var fetched struct {
		Data ruletypes.SOPDocument `json:"data"`
	}
	require.NoError(t, json.Unmarshal(fetchRW.Body.Bytes(), &fetched))
	require.Equal(t, "2026-05-12.1", fetched.Data.Version)

	bindingBody, err := json.Marshal(ruletypes.SOPBindingPreviewRequest{
		Labels: map[string]string{
			"environment": "prod",
			"project_id":  "customer-a",
			"sop_id":      "SOP-PAY-001",
		},
	})
	require.NoError(t, err)
	bindRW := httptest.NewRecorder()
	bindReq := httptest.NewRequest(http.MethodPost, "/api/v2/ds/sop/bindings/preview", bytes.NewReader(bindingBody))
	bindReq = withSOPTestClaims(bindReq)
	h.PreviewSOPDocumentBinding(bindRW, bindReq)

	require.Equal(t, http.StatusOK, bindRW.Code)
	var binding struct {
		Data ruletypes.SOPBindingPreviewResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(bindRW.Body.Bytes(), &binding))
	require.Equal(t, ruletypes.SOPBindingStatusBound, binding.Data.Status)
	require.Equal(t, ruletypes.SOPBindingResolutionExplicitLabel, binding.Data.Resolution)
	require.Equal(t, "SOP-PAY-001", binding.Data.SOPID)
}

func TestSOPDocumentHandlersRejectUnsafeCreateAndReportMissing(t *testing.T) {
	h := &handler{}
	unsafeDoc := validSOPDocumentRequest(t, "2026-05-12.1", ruletypes.SOPApprovalStatusApproved)
	unsafeDoc.BodyMarkdown = "Rotate with access_token=hidden"
	body, err := json.Marshal(unsafeDoc)
	require.NoError(t, err)

	createRW := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/v2/ds/sop/documents", bytes.NewReader(body))
	createReq = withSOPTestClaims(createReq)
	h.CreateSOPDocument(createRW, createReq)
	require.Equal(t, http.StatusBadRequest, createRW.Code)

	missingRW := httptest.NewRecorder()
	missingReq := httptest.NewRequest(http.MethodGet, "/api/v2/ds/sop/documents/SOP-UNKNOWN", nil)
	missingReq = withSOPTestClaims(missingReq)
	missingReq = muxSetVar(missingReq, "sopId", "SOP-UNKNOWN")
	h.GetSOPDocument(missingRW, missingReq)
	require.Equal(t, http.StatusNotFound, missingRW.Code)
}

func TestPreviewSOPDocumentBindingHandlerReportsDisabled(t *testing.T) {
	h := &handler{}
	body := validSOPDocumentRequestBody(t, "2026-05-12.1", ruletypes.SOPApprovalStatusDisabled)
	createRW := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/v2/ds/sop/documents", bytes.NewReader(body))
	createReq = withSOPTestClaims(createReq)
	h.CreateSOPDocument(createRW, createReq)
	require.Equal(t, http.StatusCreated, createRW.Code)

	bindingBody, err := json.Marshal(ruletypes.SOPBindingPreviewRequest{
		Labels: map[string]string{
			"environment": "prod",
			"project_id":  "customer-a",
			"sop_id":      "SOP-PAY-001",
		},
	})
	require.NoError(t, err)
	bindRW := httptest.NewRecorder()
	bindReq := httptest.NewRequest(http.MethodPost, "/api/v2/ds/sop/bindings/preview", bytes.NewReader(bindingBody))
	bindReq = withSOPTestClaims(bindReq)
	h.PreviewSOPDocumentBinding(bindRW, bindReq)

	require.Equal(t, http.StatusOK, bindRW.Code)
	var binding struct {
		Data ruletypes.SOPBindingPreviewResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(bindRW.Body.Bytes(), &binding))
	require.Equal(t, ruletypes.SOPBindingStatusDisabled, binding.Data.Status)
	require.Contains(t, binding.Data.Warnings, "sop document is disabled")
}

func TestSOPDocumentHandlersRequireClaims(t *testing.T) {
	body := validSOPDocumentRequestBody(t, "2026-05-12.1", ruletypes.SOPApprovalStatusApproved)
	rw := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v2/ds/sop/documents", bytes.NewReader(body))

	(&handler{}).CreateSOPDocument(rw, req)

	require.Equal(t, http.StatusUnauthorized, rw.Code)
}

func TestPreviewSOPDocumentBindingHandlerRejectsCrossTenantScope(t *testing.T) {
	h := &handler{}
	body := validSOPDocumentRequestBody(t, "2026-05-12.1", ruletypes.SOPApprovalStatusApproved)
	createRW := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/v2/ds/sop/documents", bytes.NewReader(body))
	createReq = withSOPTestClaims(createReq)
	h.CreateSOPDocument(createRW, createReq)
	require.Equal(t, http.StatusCreated, createRW.Code)

	bindingBody, err := json.Marshal(ruletypes.SOPBindingPreviewRequest{
		Labels: map[string]string{
			"environment": "stage",
			"project_id":  "customer-b",
			"sop_id":      "SOP-PAY-001",
		},
	})
	require.NoError(t, err)
	rw := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v2/ds/sop/bindings/preview", bytes.NewReader(bindingBody))
	req = withSOPTestClaims(req)

	h.PreviewSOPDocumentBinding(rw, req)

	require.Equal(t, http.StatusForbidden, rw.Code)
}

func TestPreviewAIStrategyHandlerAllowsAndBlocksTenantScope(t *testing.T) {
	doc := validSOPDocumentRequest(t, "2026-05-12.1", ruletypes.SOPApprovalStatusApproved)
	reqBody := ruletypes.AIStrategyRequest{
		IncidentID:       "INC-20260513-001",
		AlertFingerprint: "fp-payment-api-5xx",
		Labels: map[string]string{
			"environment":  "prod",
			"project_id":   "customer-a",
			"service.name": "payment-api",
			"severity":     "critical",
			"sop_id":       "SOP-PAY-001",
		},
		SOPDocument: doc,
		EvidenceRefs: []ruletypes.AIEvidenceRef{{
			RefID:       "metric:error_rate:1",
			Type:        "metric",
			Observation: "5xx rate rose from 0.2% to 12%",
			Confidence:  ruletypes.AIConfidenceHigh,
		}},
		GeneratedAt: "2026-05-13T00:00:00Z",
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)
	rw := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v2/ds/ai/strategy/preview", bytes.NewReader(body))
	req = withSOPTestClaims(req)

	(&handler{}).PreviewAIStrategy(rw, req)

	require.Equal(t, http.StatusOK, rw.Code)
	var got struct {
		Data ruletypes.AIStrategy `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rw.Body.Bytes(), &got))
	require.Equal(t, ruletypes.AIStrategyStatusReady, got.Data.Status)

	reqBody.Labels["project_id"] = "customer-b"
	reqBody.Labels["environment"] = "stage"
	deniedBody, err := json.Marshal(reqBody)
	require.NoError(t, err)
	deniedRW := httptest.NewRecorder()
	deniedReq := httptest.NewRequest(http.MethodPost, "/api/v2/ds/ai/strategy/preview", bytes.NewReader(deniedBody))
	deniedReq = withSOPTestClaims(deniedReq)

	(&handler{}).PreviewAIStrategy(deniedRW, deniedReq)

	require.Equal(t, http.StatusOK, deniedRW.Code)
	require.NoError(t, json.Unmarshal(deniedRW.Body.Bytes(), &got))
	require.Equal(t, ruletypes.AIStrategyStatusBlockedByPolicy, got.Data.Status)
	require.Contains(t, got.Data.Limitations, ruletypes.SOPTenantPolicyDeniedWarning)
}

func TestPreviewAIStrategyHandlerPersistsLatestHistory(t *testing.T) {
	doc := validSOPDocumentRequest(t, "2026-05-12.1", ruletypes.SOPApprovalStatusApproved)
	reqBody := ruletypes.AIStrategyRequest{
		IncidentID:       "INC-20260513-002",
		AlertFingerprint: "fp-payment-api-history",
		Labels: map[string]string{
			"environment":  "prod",
			"project_id":   "customer-a",
			"service.name": "payment-api",
			"severity":     "critical",
			"sop_id":       "SOP-PAY-001",
		},
		SOPDocument: doc,
		EvidenceRefs: []ruletypes.AIEvidenceRef{{
			RefID:       "metric:error_rate:1",
			Type:        "metric",
			Observation: "5xx rate rose from 0.2% to 12%",
			Confidence:  ruletypes.AIConfidenceHigh,
		}},
		GeneratedAt: "2026-05-13T00:00:00Z",
	}

	h := &handler{}
	previewAIStrategyForTest(t, h, reqBody)

	latestRW := httptest.NewRecorder()
	latestReq := httptest.NewRequest(
		http.MethodGet,
		"/api/v2/ds/ai/strategy/history/latest?incidentId=INC-20260513-002",
		nil,
	)
	latestReq = withSOPTestClaims(latestReq)

	h.GetLatestAIStrategyHistory(latestRW, latestReq)

	require.Equal(t, http.StatusOK, latestRW.Code)
	var latest struct {
		Data ruletypes.AIStrategyHistoryRecord `json:"data"`
	}
	require.NoError(t, json.Unmarshal(latestRW.Body.Bytes(), &latest))
	require.Equal(t, ruletypes.AIStrategyStatusReady, latest.Data.Status)
	require.Equal(t, "INC-20260513-002", latest.Data.IncidentID)
	require.Equal(t, "fp-payment-api-history", latest.Data.AlertFingerprint)

	reqBody.GeneratedAt = "2026-05-13T00:01:00Z"
	reqBody.Controls.QuotaLimit = 1
	reqBody.Controls.QuotaUsed = 1
	previewAIStrategyForTest(t, h, reqBody)

	fingerprintRW := httptest.NewRecorder()
	fingerprintReq := httptest.NewRequest(
		http.MethodGet,
		"/api/v2/ds/ai/strategy/history/latest?alertFingerprint=fp-payment-api-history",
		nil,
	)
	fingerprintReq = withSOPTestClaims(fingerprintReq)

	h.GetLatestAIStrategyHistory(fingerprintRW, fingerprintReq)

	require.Equal(t, http.StatusOK, fingerprintRW.Code)
	require.NoError(t, json.Unmarshal(fingerprintRW.Body.Bytes(), &latest))
	require.Equal(t, ruletypes.AIStrategyStatusQuotaExhausted, latest.Data.Status)
	require.Equal(t, "2026-05-13T00:01:00Z", latest.Data.GeneratedAt)
	require.Contains(t, latest.Data.Strategy.Limitations, ruletypes.AIQuotaExhaustedLimitation)
}

func TestGetLatestAIStrategyHistoryReportsInvalidAndMissingLookup(t *testing.T) {
	h := &handler{}

	missingKeyRW := httptest.NewRecorder()
	missingKeyReq := httptest.NewRequest(http.MethodGet, "/api/v2/ds/ai/strategy/history/latest", nil)
	missingKeyReq = withSOPTestClaims(missingKeyReq)
	h.GetLatestAIStrategyHistory(missingKeyRW, missingKeyReq)
	require.Equal(t, http.StatusBadRequest, missingKeyRW.Code)

	notFoundRW := httptest.NewRecorder()
	notFoundReq := httptest.NewRequest(
		http.MethodGet,
		"/api/v2/ds/ai/strategy/history/latest?incidentId=INC-MISSING",
		nil,
	)
	notFoundReq = withSOPTestClaims(notFoundReq)
	h.GetLatestAIStrategyHistory(notFoundRW, notFoundReq)
	require.Equal(t, http.StatusNotFound, notFoundRW.Code)
}

func validSOPDocumentRequestBody(t *testing.T, version string, approvalStatus string) []byte {
	t.Helper()
	body, err := json.Marshal(validSOPDocumentRequest(t, version, approvalStatus))
	require.NoError(t, err)
	return body
}

func validSOPDocumentRequest(t *testing.T, version string, approvalStatus string) ruletypes.SOPDocument {
	t.Helper()
	source := ruletypes.PilotManagedMarkdownSource{
		SourceID:              "src-managed-markdown-default",
		ServiceAccountProfile: "ds-sop-reader",
		TenantScope: ruletypes.PilotTenantScope{
			ProjectIDs:   []string{"customer-a"},
			Environments: []string{"prod"},
		},
	}
	doc := ruletypes.PilotManagedMarkdownDocument{
		SOPID:        "SOP-PAY-001",
		Version:      version,
		Title:        "Payment API 5xx response",
		BodyMarkdown: "Restart payment-api only after confirming queue drain.",
		DisplayURL:   "https://kb.example/sop/SOP-PAY-001",
		UpdatedAt:    "2026-05-12T00:00:00Z",
		Tags:         []string{"payment-api", "prod", "critical"},
	}

	return ruletypes.NewSOPDocumentFromManagedMarkdown(source, doc, "payments", approvalStatus)
}

func previewAIStrategyForTest(t *testing.T, h *handler, reqBody ruletypes.AIStrategyRequest) ruletypes.AIStrategy {
	t.Helper()
	body, err := json.Marshal(reqBody)
	require.NoError(t, err)
	rw := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v2/ds/ai/strategy/preview", bytes.NewReader(body))
	req = withSOPTestClaims(req)

	h.PreviewAIStrategy(rw, req)

	require.Equal(t, http.StatusOK, rw.Code)
	var got struct {
		Data ruletypes.AIStrategy `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rw.Body.Bytes(), &got))
	return got.Data
}

// muxSetVar injects gorilla/mux path variables into the request context,
// mirroring what the mux router does at runtime.
func muxSetVar(req *http.Request, key, value string) *http.Request {
	vars := mux.Vars(req)
	if vars == nil {
		vars = map[string]string{}
	}
	vars[key] = value
	return mux.SetURLVars(req, vars)
}

func withSOPTestClaims(req *http.Request) *http.Request {
	claims := authtypes.Claims{
		UserID:         "user-123",
		Principal:      authtypes.PrincipalUser,
		Email:          "operator@example.com",
		OrgID:          "00000000-0000-0000-0000-000000000001",
		IdentNProvider: authtypes.IdentNProviderTokenizer,
	}
	return req.WithContext(authtypes.NewContextWithClaims(req.Context(), claims))
}
