package signozruler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/stretchr/testify/require"
)

func TestFetchPilotManagedMarkdownSOPHandler(t *testing.T) {
	body, err := json.Marshal(ruletypes.PilotManagedMarkdownSOPFetchRequest{
		Source: ruletypes.PilotManagedMarkdownSource{
			SourceID:              "src-managed-markdown-default",
			DisplayName:           "Managed Markdown SOP Registry",
			Status:                ruletypes.PilotSOPSourceStatusHealthy,
			LastHealthCheckAt:     "2026-04-30T00:00:00Z",
			LastSyncAt:            "2026-04-30T00:00:00Z",
			ServiceAccountProfile: "ds-sop-reader",
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

func TestFetchPilotManagedMarkdownSOPHandlerReturnsInvalidInput(t *testing.T) {
	rw := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v2/rules/sop/pilot/managed_markdown/fetch", bytes.NewReader([]byte(`{}`)))

	(&handler{}).FetchPilotManagedMarkdownSOP(rw, req)

	require.Equal(t, http.StatusBadRequest, rw.Code)
}
