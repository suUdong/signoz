package ruletypes

import (
	"testing"

	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
	"github.com/stretchr/testify/require"
)

func TestGenerateLocalAIStrategyReturnsReadyGroundedStrategy(t *testing.T) {
	strategy, err := GenerateLocalAIStrategy(validAIStrategyRequest())

	require.NoError(t, err)
	require.Equal(t, AIStrategyContractVersion, strategy.ContractVersion)
	require.Equal(t, AIStrategyStatusReady, strategy.Status)
	require.Equal(t, "INC-20260512-001", strategy.IncidentID)
	require.Equal(t, "SOP-PAY-001", strategy.SOPID)
	require.Equal(t, "2026-04-20.3", strategy.SOPVersion)
	require.Equal(t, AIConfidenceMedium, strategy.Confidence)
	require.Len(t, strategy.Hypotheses, 1)
	require.Equal(t, []string{"metric:error_rate:1", "trace:error:1"}, strategy.Hypotheses[0].EvidenceRefs)
	require.Equal(t, []string{"SOP-PAY-001#1"}, strategy.Hypotheses[0].SOPStepRefs)
	require.Len(t, strategy.FirstActions, 1)
	require.True(t, strategy.FirstActions[0].RequiresHumanApproval)
	require.Equal(t, "SOP-PAY-001#1", strategy.FirstActions[0].SOPStepRef)
	require.NotEmpty(t, strategy.CustomerUpdateDraft)
	require.NotEmpty(t, strategy.VendorRequestDraft)
	require.NoError(t, ValidateAIStrategy(strategy))
}

func TestGenerateLocalAIStrategyIsDeterministic(t *testing.T) {
	req := validAIStrategyRequest()

	first, err := GenerateLocalAIStrategy(req)
	require.NoError(t, err)
	second, err := GenerateLocalAIStrategy(req)
	require.NoError(t, err)

	require.Equal(t, first, second)
}

func TestGenerateLocalAIStrategyHandlesMissingEvidenceWithoutRootCauseClaim(t *testing.T) {
	req := validAIStrategyRequest()
	req.EvidenceRefs = nil

	strategy, err := GenerateLocalAIStrategy(req)

	require.NoError(t, err)
	require.Equal(t, AIStrategyStatusEvidenceUnavailable, strategy.Status)
	require.Empty(t, strategy.Hypotheses)
	require.Len(t, strategy.FirstActions, 1)
	require.Contains(t, strategy.FirstActions[0].Text, "SOP-PAY-001")
	require.NotEmpty(t, strategy.Limitations)
	require.NoError(t, ValidateAIStrategy(strategy))
}

func TestGenerateLocalAIStrategyHandlesMissingSOPWithoutFabricatingSteps(t *testing.T) {
	req := validAIStrategyRequest()
	req.SOPDocument = SOPDocument{}

	strategy, err := GenerateLocalAIStrategy(req)

	require.NoError(t, err)
	require.Equal(t, AIStrategyStatusSOPMissing, strategy.Status)
	require.Empty(t, strategy.SOPID)
	require.Empty(t, strategy.FirstActions)
	require.Empty(t, strategy.Hypotheses)
	require.NotEmpty(t, strategy.Limitations)
	require.NoError(t, ValidateAIStrategy(strategy))
}

func TestGenerateLocalAIStrategyBlocksUnsafeSOPBeforeGeneration(t *testing.T) {
	req := validAIStrategyRequest()
	req.SOPDocument.BodyMarkdown = "Rotate credential with access_token=hidden"

	strategy, err := GenerateLocalAIStrategy(req)

	require.NoError(t, err)
	require.Equal(t, AIStrategyStatusBlockedByPolicy, strategy.Status)
	require.Empty(t, strategy.Hypotheses)
	require.Empty(t, strategy.FirstActions)
	require.NotContains(t, strategy.Headline, "access_token")
	require.NoError(t, ValidateAIStrategy(strategy))
}

func TestGenerateLocalAIStrategyBlocksMissingTenantLabels(t *testing.T) {
	req := validAIStrategyRequest()
	delete(req.Labels, "project_id")

	strategy, err := GenerateLocalAIStrategy(req)

	require.NoError(t, err)
	require.Equal(t, AIStrategyStatusBlockedByPolicy, strategy.Status)
	require.Empty(t, strategy.Hypotheses)
	require.Empty(t, strategy.FirstActions)
	require.Contains(t, strategy.Limitations, SOPTenantPolicyMissingLabelsWarning)
	require.NoError(t, ValidateAIStrategy(strategy))
}

func TestGenerateLocalAIStrategyBlocksCrossTenantSOP(t *testing.T) {
	req := validAIStrategyRequest()
	req.Labels["project_id"] = "customer-b"
	req.Labels["environment"] = "stage"

	strategy, err := GenerateLocalAIStrategy(req)

	require.NoError(t, err)
	require.Equal(t, AIStrategyStatusBlockedByPolicy, strategy.Status)
	require.Equal(t, "SOP-PAY-001", strategy.SOPID)
	require.Empty(t, strategy.Hypotheses)
	require.Empty(t, strategy.FirstActions)
	require.Contains(t, strategy.Limitations, SOPTenantPolicyDeniedWarning)
	require.NoError(t, ValidateAIStrategy(strategy))
}

func TestGenerateLocalAIStrategyFailsOpenWhenProviderDisabled(t *testing.T) {
	req := validAIStrategyRequest()
	req.Controls.ProviderEnabled = boolPointer(false)

	strategy, err := GenerateLocalAIStrategy(req)

	require.NoError(t, err)
	require.Equal(t, AIStrategyStatusUnavailable, strategy.Status)
	require.Equal(t, "SOP-PAY-001", strategy.SOPID)
	require.Empty(t, strategy.Hypotheses)
	require.Empty(t, strategy.FirstActions)
	require.Contains(t, strategy.Limitations, AIProviderDisabledLimitation)
	require.NoError(t, ValidateAIStrategy(strategy))

	annotations := AIStrategyIncidentAnnotations(strategy)
	require.Equal(
		t,
		AIStrategyStatusUnavailable,
		annotations[alertmanagertypes.IncidentAnnotationAIStrategyStatus],
	)
	require.Contains(
		t,
		annotations[alertmanagertypes.IncidentAnnotationAILimitations],
		AIProviderDisabledLimitation,
	)
}

func TestGenerateLocalAIStrategyBlocksUnlicensedTenant(t *testing.T) {
	req := validAIStrategyRequest()
	req.Controls.LicenseAllowed = boolPointer(false)

	strategy, err := GenerateLocalAIStrategy(req)

	require.NoError(t, err)
	require.Equal(t, AIStrategyStatusBlockedByPolicy, strategy.Status)
	require.Empty(t, strategy.Hypotheses)
	require.Empty(t, strategy.FirstActions)
	require.Contains(t, strategy.Limitations, AILicenseUnavailableLimitation)
	require.NoError(t, ValidateAIStrategy(strategy))
}

func TestGenerateLocalAIStrategyTracksQuotaExhaustion(t *testing.T) {
	req := validAIStrategyRequest()
	req.Controls.QuotaLimit = 10
	req.Controls.QuotaUsed = 10

	strategy, err := GenerateLocalAIStrategy(req)

	require.NoError(t, err)
	require.Equal(t, AIStrategyStatusQuotaExhausted, strategy.Status)
	require.Contains(t, strategy.Limitations, AIQuotaExhaustedLimitation)
	require.NotNil(t, strategy.Audit.QuotaLimit)
	require.NotNil(t, strategy.Audit.QuotaUsed)
	require.NotNil(t, strategy.Audit.QuotaRemaining)
	require.Equal(t, int64(10), *strategy.Audit.QuotaLimit)
	require.Equal(t, int64(10), *strategy.Audit.QuotaUsed)
	require.Equal(t, int64(0), *strategy.Audit.QuotaRemaining)
	require.NoError(t, ValidateAIStrategy(strategy))
}

func TestGenerateLocalAIStrategyTracksTimeoutBudget(t *testing.T) {
	req := validAIStrategyRequest()
	req.Controls.TimeoutBudgetMillis = 1500
	req.Controls.ExecutionElapsedMillis = 1501

	strategy, err := GenerateLocalAIStrategy(req)

	require.NoError(t, err)
	require.Equal(t, AIStrategyStatusTimeout, strategy.Status)
	require.Contains(t, strategy.Limitations, AITimeoutBudgetExceededLimitation)
	require.NotNil(t, strategy.Audit.TimeoutBudgetMillis)
	require.NotNil(t, strategy.Audit.ExecutionElapsedMillis)
	require.Equal(t, int64(1500), *strategy.Audit.TimeoutBudgetMillis)
	require.Equal(t, int64(1501), *strategy.Audit.ExecutionElapsedMillis)
	require.NoError(t, ValidateAIStrategy(strategy))
}

func TestGenerateLocalAIStrategyTracksUsageOnReadyStrategy(t *testing.T) {
	req := validAIStrategyRequest()
	req.Controls.QuotaLimit = 100
	req.Controls.QuotaUsed = 12
	req.Controls.TimeoutBudgetMillis = 5000
	req.Controls.ExecutionElapsedMillis = 120

	strategy, err := GenerateLocalAIStrategy(req)

	require.NoError(t, err)
	require.Equal(t, AIStrategyStatusReady, strategy.Status)
	require.NotNil(t, strategy.Audit.QuotaRemaining)
	require.Equal(t, int64(88), *strategy.Audit.QuotaRemaining)
	require.NotNil(t, strategy.Audit.ExecutionElapsedMillis)
	require.Equal(t, int64(120), *strategy.Audit.ExecutionElapsedMillis)
	require.NoError(t, ValidateAIStrategy(strategy))
}

func TestValidateAIStrategyRejectsUngroundedReadyOutput(t *testing.T) {
	strategy, err := GenerateLocalAIStrategy(validAIStrategyRequest())
	require.NoError(t, err)
	strategy.Hypotheses[0].EvidenceRefs = nil
	strategy.Hypotheses[0].SOPStepRefs = nil

	err = ValidateAIStrategy(strategy)
	require.ErrorContains(t, err, "hypotheses[0]: must include at least one evidenceRefs or sopStepRefs entry")
}

func TestValidateAIStrategyRejectsAutomaticExecutionClaims(t *testing.T) {
	strategy, err := GenerateLocalAIStrategy(validAIStrategyRequest())
	require.NoError(t, err)
	strategy.FirstActions[0].Text = "서버를 자동 재시작했습니다."

	err = ValidateAIStrategy(strategy)
	require.ErrorContains(t, err, "firstActions[0].text: must not claim automatic operational execution")
}

func TestValidateAIStrategyRejectsSecretLikeOutput(t *testing.T) {
	strategy, err := GenerateLocalAIStrategy(validAIStrategyRequest())
	require.NoError(t, err)
	strategy.CustomerUpdateDraft = "token=hidden"

	err = ValidateAIStrategy(strategy)
	require.ErrorContains(t, err, "customerUpdateDraft: contains secret-like value")
}

func TestGenerateLocalAIStrategyRequiresIncidentID(t *testing.T) {
	req := validAIStrategyRequest()
	req.IncidentID = ""

	_, err := GenerateLocalAIStrategy(req)
	require.ErrorContains(t, err, "incidentId: field is required")
}

func TestAIStrategyIncidentAnnotationsMapsReadyStrategy(t *testing.T) {
	strategy, err := GenerateLocalAIStrategy(validAIStrategyRequest())
	require.NoError(t, err)

	got := AIStrategyIncidentAnnotations(strategy)

	require.Equal(t, strategy.StrategyID, got[alertmanagertypes.IncidentAnnotationAIStrategyID])
	require.Equal(t, AIStrategyStatusReady, got[alertmanagertypes.IncidentAnnotationAIStrategyStatus])
	require.Equal(t, strategy.Headline, got[alertmanagertypes.IncidentAnnotationAIHeadline])
	require.Equal(t, AIConfidenceMedium, got[alertmanagertypes.IncidentAnnotationAIConfidence])
	require.Contains(t, got[alertmanagertypes.IncidentAnnotationAIFirstActions], "SOP-PAY-001")
	require.Equal(t, "metric:error_rate:1, trace:error:1", got[alertmanagertypes.IncidentAnnotationAIEvidenceRefs])
	require.Equal(t, strategy.CustomerUpdateDraft, got[alertmanagertypes.IncidentAnnotationCustomerUpdate])
	require.Equal(t, strategy.VendorRequestDraft, got[alertmanagertypes.IncidentAnnotationVendorRequest])
	require.NotEmpty(t, got[alertmanagertypes.IncidentAnnotationAILimitations])
}

func TestAIStrategyIncidentAnnotationsMapsFallbackWithoutFabricatedActions(t *testing.T) {
	req := validAIStrategyRequest()
	req.SOPDocument = SOPDocument{}
	strategy, err := GenerateLocalAIStrategy(req)
	require.NoError(t, err)

	got := AIStrategyIncidentAnnotations(strategy)

	require.Equal(t, AIStrategyStatusSOPMissing, got[alertmanagertypes.IncidentAnnotationAIStrategyStatus])
	require.NotEmpty(t, got[alertmanagertypes.IncidentAnnotationAILimitations])
	require.NotContains(t, got, alertmanagertypes.IncidentAnnotationAIFirstActions)
	require.NotContains(t, got, alertmanagertypes.IncidentAnnotationAIEvidenceRefs)
}

func validAIStrategyRequest() AIStrategyRequest {
	source := validPilotManagedMarkdownSource()
	doc := NewSOPDocumentFromManagedMarkdown(source, source.Documents[0], "payments", SOPApprovalStatusApproved)
	doc.BodyMarkdown = "# Payment API 5xx response\n\n1. 결제 성공률 dashboard와 PG timeout log를 확인\n2. 큐 적체 여부 확인"
	return AIStrategyRequest{
		IncidentID:       "INC-20260512-001",
		AlertFingerprint: "fp-payment-api-5xx",
		Labels: map[string]string{
			"environment":  "prod",
			"project_id":   "customer-a",
			"service.name": "payment-api",
			"severity":     "critical",
			"sop_id":       "SOP-PAY-001",
		},
		SOPDocument: doc,
		EvidenceRefs: []AIEvidenceRef{
			{
				RefID:       "metric:error_rate:1",
				Type:        "metric",
				Observation: "5xx rate rose from 0.2% to 12%",
				Permalink:   "https://signoz.example/dashboard/payment-api",
				Confidence:  AIConfidenceHigh,
			},
			{
				RefID:       "trace:error:1",
				Type:        "trace",
				Observation: "PG timeout spans increased",
				Permalink:   "https://signoz.example/traces/trace-sample",
				Confidence:  AIConfidenceMedium,
			},
		},
		GeneratedAt: "2026-05-12T00:00:00Z",
	}
}

func boolPointer(value bool) *bool {
	return &value
}
