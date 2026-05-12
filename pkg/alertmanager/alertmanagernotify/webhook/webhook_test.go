// Copyright (c) 2026 SigNoz, Inc.
// Copyright 2019 Prometheus Team
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	commoncfg "github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	"github.com/prometheus/common/promslog"
	"github.com/stretchr/testify/require"

	"github.com/prometheus/alertmanager/config"
	"github.com/prometheus/alertmanager/notify"
	"github.com/prometheus/alertmanager/notify/test"
	"github.com/prometheus/alertmanager/types"

	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
)

func TestWebhookRetry(t *testing.T) {
	notifier, err := New(
		&config.WebhookConfig{
			URL:        config.SecretTemplateURL("http://example.com"),
			HTTPConfig: &commoncfg.HTTPClientConfig{},
		},
		test.CreateTmpl(t),
		promslog.NewNopLogger(),
	)
	if err != nil {
		require.NoError(t, err)
	}

	t.Run("test retry status code", func(t *testing.T) {
		for statusCode, expected := range test.RetryTests(test.DefaultRetryCodes()) {
			actual, _ := notifier.retrier.Check(statusCode, nil)
			require.Equal(t, expected, actual, "error on status %d", statusCode)
		}
	})

	t.Run("test retry error details", func(t *testing.T) {
		for _, tc := range []struct {
			status int
			body   io.Reader

			exp string
		}{
			{
				status: http.StatusBadRequest,
				body: bytes.NewBuffer([]byte(
					`{"status":"invalid event"}`,
				)),

				exp: fmt.Sprintf(`unexpected status code %d: {"status":"invalid event"}`, http.StatusBadRequest),
			},
			{
				status: http.StatusBadRequest,

				exp: fmt.Sprintf(`unexpected status code %d`, http.StatusBadRequest),
			},
		} {
			t.Run("", func(t *testing.T) {
				_, err = notifier.retrier.Check(tc.status, tc.body)
				require.Equal(t, tc.exp, err.Error())
			})
		}
	})
}

func TestWebhookTruncateAlerts(t *testing.T) {
	alerts := make([]*types.Alert, 10)

	truncatedAlerts, numTruncated := truncateAlerts(0, alerts)
	require.Len(t, truncatedAlerts, 10)
	require.EqualValues(t, 0, numTruncated)

	truncatedAlerts, numTruncated = truncateAlerts(4, alerts)
	require.Len(t, truncatedAlerts, 4)
	require.EqualValues(t, 6, numTruncated)

	truncatedAlerts, numTruncated = truncateAlerts(100, alerts)
	require.Len(t, truncatedAlerts, 10)
	require.EqualValues(t, 0, numTruncated)
}

func TestWebhookRedactedURL(t *testing.T) {
	ctx, u, fn := test.GetContextWithCancelingURL()
	defer fn()

	secret := "secret"
	notifier, err := New(
		&config.WebhookConfig{
			URL:        config.SecretTemplateURL(u.String()),
			HTTPConfig: &commoncfg.HTTPClientConfig{},
		},
		test.CreateTmpl(t),
		promslog.NewNopLogger(),
	)
	require.NoError(t, err)

	test.AssertNotifyLeaksNoSecret(ctx, t, notifier, secret)
}

func TestWebhookReadingURLFromFile(t *testing.T) {
	ctx, u, fn := test.GetContextWithCancelingURL()
	defer fn()

	f, err := os.CreateTemp(t.TempDir(), "webhook_url")
	require.NoError(t, err, "creating temp file failed")
	_, err = f.WriteString(u.String() + "\n")
	require.NoError(t, err, "writing to temp file failed")

	notifier, err := New(
		&config.WebhookConfig{
			URLFile:    f.Name(),
			HTTPConfig: &commoncfg.HTTPClientConfig{},
		},
		test.CreateTmpl(t),
		promslog.NewNopLogger(),
	)
	require.NoError(t, err)

	test.AssertNotifyLeaksNoSecret(ctx, t, notifier, u.String())
}

func TestWebhookURLTemplating(t *testing.T) {
	var calledURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calledURL = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tests := []struct {
		name           string
		url            string
		groupLabels    model.LabelSet
		alertLabels    model.LabelSet
		expectError    bool
		expectedErrMsg string
		expectedPath   string
	}{
		{
			name:         "templating with alert labels",
			url:          srv.URL + "/{{ .GroupLabels.alertname }}/{{ .CommonLabels.severity }}",
			groupLabels:  model.LabelSet{"alertname": "TestAlert"},
			alertLabels:  model.LabelSet{"alertname": "TestAlert", "severity": "critical"},
			expectError:  false,
			expectedPath: "/TestAlert/critical",
		},
		{
			name:           "invalid template field",
			url:            srv.URL + "/{{ .InvalidField }}",
			groupLabels:    model.LabelSet{"alertname": "TestAlert"},
			alertLabels:    model.LabelSet{"alertname": "TestAlert"},
			expectError:    true,
			expectedErrMsg: "failed to template webhook URL",
		},
		{
			name:           "template renders to empty string",
			url:            "{{ if .CommonLabels.nonexistent }}http://example.com{{ end }}",
			groupLabels:    model.LabelSet{"alertname": "TestAlert"},
			alertLabels:    model.LabelSet{"alertname": "TestAlert"},
			expectError:    true,
			expectedErrMsg: "webhook URL is empty after templating",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			calledURL = "" // Reset for each test

			notifier, err := New(
				&config.WebhookConfig{
					URL:        config.SecretTemplateURL(tc.url),
					HTTPConfig: &commoncfg.HTTPClientConfig{},
				},
				test.CreateTmpl(t),
				promslog.NewNopLogger(),
			)
			require.NoError(t, err)

			ctx := context.Background()
			ctx = notify.WithGroupKey(ctx, "test-group")
			if tc.groupLabels != nil {
				ctx = notify.WithGroupLabels(ctx, tc.groupLabels)
			}

			alerts := []*types.Alert{
				{
					Alert: model.Alert{
						Labels:   tc.alertLabels,
						StartsAt: time.Now(),
						EndsAt:   time.Now().Add(time.Hour),
					},
				},
			}

			_, err = notifier.Notify(ctx, alerts...)

			if tc.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedErrMsg)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedPath, calledURL)
			}
		})
	}
}

func TestWebhookIncludesIncidentContext(t *testing.T) {
	var payload struct {
		Incident *alertmanagertypes.IncidentInfo `json:"incident"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	notifier, err := New(
		&config.WebhookConfig{
			URL:        config.SecretTemplateURL(srv.URL),
			HTTPConfig: &commoncfg.HTTPClientConfig{},
		},
		test.CreateTmpl(t),
		promslog.NewNopLogger(),
	)
	require.NoError(t, err)

	ctx := context.Background()
	ctx = notify.WithGroupKey(ctx, "test-group")
	ctx = notify.WithGroupLabels(ctx, model.LabelSet{"alertname": "CheckoutLatencyHigh"})

	_, err = notifier.Notify(ctx, &types.Alert{
		Alert: model.Alert{
			Labels: model.LabelSet{
				"alertname": "CheckoutLatencyHigh",
				model.LabelName(alertmanagertypes.IncidentLabelProjectID):   "customer-a",
				model.LabelName(alertmanagertypes.IncidentLabelEnvironment): "prod",
				model.LabelName(alertmanagertypes.IncidentLabelServiceName): "checkout-api",
				model.LabelName(alertmanagertypes.IncidentLabelOwnerTeam):   "sm-payments",
				model.LabelName(alertmanagertypes.IncidentLabelSeverity):    "critical",
				model.LabelName(alertmanagertypes.IncidentLabelSopID):       "SOP-PAY-001",
			},
			Annotations: model.LabelSet{
				model.LabelName(alertmanagertypes.IncidentAnnotationImpactSummary):    "Checkout latency can affect customer payments.",
				model.LabelName(alertmanagertypes.IncidentAnnotationNextAction):       "Ask vendor to inspect slow traces.",
				model.LabelName(alertmanagertypes.IncidentAnnotationVendorRequest):    "Need cause, mitigation, and ETA.",
				model.LabelName(alertmanagertypes.IncidentAnnotationCustomerUpdate):   "Payment latency is under investigation.",
				model.LabelName(alertmanagertypes.IncidentAnnotationSopURL):           "https://runbooks.example.com/payment-latency",
				model.LabelName(alertmanagertypes.IncidentAnnotationSopSource):        "confluence",
				model.LabelName(alertmanagertypes.IncidentAnnotationSopTitle):         "Payment API 5xx response",
				model.LabelName(alertmanagertypes.IncidentAnnotationSopVersion):       "2026-04-20.3",
				model.LabelName(alertmanagertypes.IncidentAnnotationSopBindingID):     "payment-api-prod-critical",
				model.LabelName(alertmanagertypes.IncidentAnnotationAIStrategyID):     "AIS-20260512-0001",
				model.LabelName(alertmanagertypes.IncidentAnnotationAIStrategyStatus): "ready",
				model.LabelName(alertmanagertypes.IncidentAnnotationAIHeadline):       "SOP 기준 결제 지연 확인이 필요합니다.",
				model.LabelName(alertmanagertypes.IncidentAnnotationAIFirstActions):   "PG timeout 로그를 확인",
				model.LabelName(alertmanagertypes.IncidentAnnotationAIConfidence):     "medium",
				model.LabelName(alertmanagertypes.IncidentAnnotationAILimitations):    "최근 배포 정보는 연결되지 않음",
				model.LabelName(alertmanagertypes.IncidentAnnotationAIEvidenceRefs):   "metric:error_rate:1",
			},
			StartsAt: time.Now(),
			EndsAt:   time.Now().Add(time.Hour),
		},
	})
	require.NoError(t, err)
	require.NotNil(t, payload.Incident)
	require.Equal(t, &alertmanagertypes.IncidentInfo{
		ProjectID:        "customer-a",
		Environment:      "prod",
		ServiceName:      "checkout-api",
		OwnerTeam:        "sm-payments",
		Severity:         "critical",
		ImpactSummary:    "Checkout latency can affect customer payments.",
		NextAction:       "Ask vendor to inspect slow traces.",
		VendorRequest:    "Need cause, mitigation, and ETA.",
		CustomerUpdate:   "Payment latency is under investigation.",
		SopID:            "SOP-PAY-001",
		SopURL:           "https://runbooks.example.com/payment-latency",
		SopSource:        "confluence",
		SopTitle:         "Payment API 5xx response",
		SopVersion:       "2026-04-20.3",
		SopBindingID:     "payment-api-prod-critical",
		AIStrategyID:     "AIS-20260512-0001",
		AIStrategyStatus: "ready",
		AIHeadline:       "SOP 기준 결제 지연 확인이 필요합니다.",
		AIFirstActions:   "PG timeout 로그를 확인",
		AIConfidence:     "medium",
		AILimitations:    "최근 배포 정보는 연결되지 않음",
		AIEvidenceRefs:   "metric:error_rate:1",
	}, payload.Incident)
}
