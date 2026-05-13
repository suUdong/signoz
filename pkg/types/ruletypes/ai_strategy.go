package ruletypes

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
)

const (
	AIStrategyContractVersion = "ds.ai_strategy.v1"

	AIStrategyStatusReady               = "ready"
	AIStrategyStatusUnavailable         = "unavailable"
	AIStrategyStatusTimeout             = "timeout"
	AIStrategyStatusBlockedByPolicy     = "blocked_by_policy"
	AIStrategyStatusQuotaExhausted      = "quota_exhausted"
	AIStrategyStatusSOPMissing          = "sop_missing"
	AIStrategyStatusEvidenceUnavailable = "evidence_unavailable"
	AIStrategyStatusLowConfidence       = "low_confidence"

	AIConfidenceHigh   = "high"
	AIConfidenceMedium = "medium"
	AIConfidenceLow    = "low"

	defaultAIStrategyLanguage      = "ko-KR"
	defaultAIStrategyPromptVersion = "ds-ir-ko-v1"
	defaultAIStrategyModel         = "deterministic-local"
	defaultAIStrategyGeneratedAt   = "1970-01-01T00:00:00Z"

	AIProviderDisabledLimitation      = "AI provider is disabled by tenant or deployment controls."
	AILicenseUnavailableLimitation    = "AI strategy generation is not licensed for this tenant."
	AIQuotaExhaustedLimitation        = "AI strategy quota is exhausted for this period."
	AITimeoutBudgetExceededLimitation = "AI strategy generation exceeded the configured timeout budget."
)

type AIStrategyRequest struct {
	StrategyID       string             `json:"strategyId,omitempty"`
	IncidentID       string             `json:"incidentId"`
	AlertFingerprint string             `json:"alertFingerprint,omitempty"`
	Language         string             `json:"language,omitempty"`
	Labels           map[string]string  `json:"labels,omitempty"`
	Annotations      map[string]string  `json:"annotations,omitempty"`
	SOPDocument      SOPDocument        `json:"sopDocument,omitempty"`
	EvidenceRefs     []AIEvidenceRef    `json:"evidenceRefs,omitempty"`
	PromptVersion    string             `json:"promptVersion,omitempty"`
	Model            string             `json:"model,omitempty"`
	Controls         AIStrategyControls `json:"controls,omitempty"`
	GeneratedAt      string             `json:"generatedAt,omitempty"`
}

type AIStrategyControls struct {
	ProviderEnabled        *bool `json:"providerEnabled,omitempty"`
	LicenseAllowed         *bool `json:"licenseAllowed,omitempty"`
	QuotaLimit             int64 `json:"quotaLimit,omitempty"`
	QuotaUsed              int64 `json:"quotaUsed,omitempty"`
	TimeoutBudgetMillis    int64 `json:"timeoutBudgetMillis,omitempty"`
	ExecutionElapsedMillis int64 `json:"executionElapsedMillis,omitempty"`
}

type AIStrategy struct {
	ContractVersion     string          `json:"contractVersion"`
	StrategyID          string          `json:"strategyId"`
	IncidentID          string          `json:"incidentId"`
	AlertFingerprint    string          `json:"alertFingerprint,omitempty"`
	Status              string          `json:"status"`
	Language            string          `json:"language"`
	SOPID               string          `json:"sopId,omitempty"`
	SOPVersion          string          `json:"sopVersion,omitempty"`
	Headline            string          `json:"headline,omitempty"`
	Hypotheses          []AIHypothesis  `json:"hypotheses,omitempty"`
	FirstActions        []AIFirstAction `json:"firstActions,omitempty"`
	CustomerUpdateDraft string          `json:"customerUpdateDraft,omitempty"`
	VendorRequestDraft  string          `json:"vendorRequestDraft,omitempty"`
	Confidence          string          `json:"confidence"`
	EvidenceRefs        []AIEvidenceRef `json:"evidenceRefs,omitempty"`
	Limitations         []string        `json:"limitations,omitempty"`
	Audit               AIStrategyAudit `json:"audit"`
}

type AIHypothesis struct {
	Rank         int      `json:"rank"`
	Text         string   `json:"text"`
	Confidence   string   `json:"confidence"`
	EvidenceRefs []string `json:"evidenceRefs,omitempty"`
	SOPStepRefs  []string `json:"sopStepRefs,omitempty"`
}

type AIFirstAction struct {
	Text                  string   `json:"text"`
	SOPStepRef            string   `json:"sopStepRef,omitempty"`
	EvidenceRefs          []string `json:"evidenceRefs,omitempty"`
	RequiresHumanApproval bool     `json:"requiresHumanApproval"`
}

type AIEvidenceRef struct {
	RefID       string `json:"refId"`
	Type        string `json:"type"`
	Observation string `json:"observation"`
	Permalink   string `json:"permalink,omitempty"`
	Confidence  string `json:"confidence,omitempty"`
}

type AIStrategyAudit struct {
	PromptVersion          string `json:"promptVersion"`
	Model                  string `json:"model"`
	GeneratedAt            string `json:"generatedAt"`
	RedactionApplied       bool   `json:"redactionApplied"`
	QuotaLimit             *int64 `json:"quotaLimit,omitempty"`
	QuotaUsed              *int64 `json:"quotaUsed,omitempty"`
	QuotaRemaining         *int64 `json:"quotaRemaining,omitempty"`
	TimeoutBudgetMillis    *int64 `json:"timeoutBudgetMillis,omitempty"`
	ExecutionElapsedMillis *int64 `json:"executionElapsedMillis,omitempty"`
}

func GenerateLocalAIStrategy(req AIStrategyRequest) (AIStrategy, error) {
	strategy := baseAIStrategy(req)
	if strings.TrimSpace(req.IncidentID) == "" {
		return strategy, ValidateAIStrategy(strategy)
	}
	if strings.TrimSpace(req.SOPDocument.SOPID) != "" {
		strategy.SOPID = strings.TrimSpace(req.SOPDocument.SOPID)
		strategy.SOPVersion = strings.TrimSpace(req.SOPDocument.Version)
	}
	if applyAIStrategyControls(&strategy, req.Controls) {
		return strategy, ValidateAIStrategy(strategy)
	}

	if strings.TrimSpace(req.SOPDocument.SOPID) == "" {
		strategy.Status = AIStrategyStatusSOPMissing
		strategy.Confidence = AIConfidenceLow
		strategy.Headline = "연결된 SOP 문서가 없어 기본 알림만 전송합니다."
		strategy.Limitations = []string{"Bound SOP document is missing."}
		return strategy, ValidateAIStrategy(strategy)
	}

	tenant := PilotTenantFromLabels(req.Labels)
	if !PilotTenantIsComplete(tenant) {
		strategy.Status = AIStrategyStatusBlockedByPolicy
		strategy.Confidence = AIConfidenceLow
		strategy.Headline = "테넌트 라벨이 없어 AI 대응전략을 생성하지 않았습니다."
		strategy.Limitations = []string{SOPTenantPolicyMissingLabelsWarning}
		return strategy, ValidateAIStrategy(strategy)
	}
	if !PilotTenantScopeAllows(req.SOPDocument.TenantScope, tenant) {
		strategy.Status = AIStrategyStatusBlockedByPolicy
		strategy.Confidence = AIConfidenceLow
		strategy.Headline = "SOP 문서의 테넌트 범위가 알림 라벨과 일치하지 않아 AI 대응전략을 생성하지 않았습니다."
		strategy.Limitations = []string{SOPTenantPolicyDeniedWarning}
		return strategy, ValidateAIStrategy(strategy)
	}
	if err := ValidateSOPDocument(req.SOPDocument); err != nil {
		strategy.Status = AIStrategyStatusBlockedByPolicy
		strategy.Confidence = AIConfidenceLow
		strategy.Headline = "SOP 문서가 안전 정책을 통과하지 못해 AI 대응전략을 생성하지 않았습니다."
		strategy.Limitations = []string{"SOP document validation failed before AI generation."}
		return strategy, ValidateAIStrategy(strategy)
	}

	firstActionText := firstActionFromSOP(req.SOPDocument)
	sopStepRef := req.SOPDocument.SOPID + "#1"
	if len(req.EvidenceRefs) == 0 {
		strategy.Status = AIStrategyStatusEvidenceUnavailable
		strategy.Confidence = AIConfidenceLow
		strategy.Headline = fmt.Sprintf("%s SOP 기준 첫 조치 확인이 필요합니다.", req.SOPDocument.Title)
		strategy.FirstActions = []AIFirstAction{{
			Text:                  fmt.Sprintf("%s 1단계에 따라 %s", req.SOPDocument.SOPID, firstActionText),
			SOPStepRef:            sopStepRef,
			RequiresHumanApproval: true,
		}}
		strategy.CustomerUpdateDraft = "장애 알림을 확인했으며, 현재 SOP 기준 초동 확인을 진행 중입니다."
		strategy.Limitations = []string{"No evidence refs were available, so root-cause hypotheses were not generated."}
		return strategy, ValidateAIStrategy(strategy)
	}

	evidenceIDs := evidenceRefIDs(req.EvidenceRefs)
	serviceName := firstNonEmpty(req.Labels[alertmanagertypes.IncidentLabelServiceName], "대상 서비스")
	severity := firstNonEmpty(req.Labels[alertmanagertypes.IncidentLabelSeverity], "unknown")
	observation := firstNonEmpty(req.EvidenceRefs[0].Observation, "최근 SigNoz evidence에서 이상 징후가 관측되었습니다")

	strategy.Status = AIStrategyStatusReady
	strategy.Confidence = AIConfidenceMedium
	strategy.Headline = fmt.Sprintf("%s %s 알림은 SOP %s 기준으로 즉시 확인이 필요합니다.", serviceName, severity, req.SOPDocument.SOPID)
	strategy.Hypotheses = []AIHypothesis{{
		Rank:         1,
		Text:         fmt.Sprintf("%s와 연관된 장애 가능성이 있습니다.", observation),
		Confidence:   AIConfidenceMedium,
		EvidenceRefs: evidenceIDs,
		SOPStepRefs:  []string{sopStepRef},
	}}
	strategy.FirstActions = []AIFirstAction{{
		Text:                  fmt.Sprintf("%s 1단계에 따라 %s", req.SOPDocument.SOPID, firstActionText),
		SOPStepRef:            sopStepRef,
		EvidenceRefs:          evidenceIDs[:1],
		RequiresHumanApproval: true,
	}}
	strategy.CustomerUpdateDraft = fmt.Sprintf("현재 %s 장애 알림을 확인하여 SOP 기준 초동 분석 중입니다. 다음 업데이트는 15분 내 공유하겠습니다.", serviceName)
	strategy.VendorRequestDraft = fmt.Sprintf("%s 관련 외부 의존성 또는 공급자 장애 공지 여부를 확인 부탁드립니다.", serviceName)
	strategy.EvidenceRefs = req.EvidenceRefs
	strategy.Limitations = []string{"Deterministic local strategy; external LLM provider is not used."}

	return strategy, ValidateAIStrategy(strategy)
}

func ValidateAIStrategy(strategy AIStrategy) error {
	var errs []error

	pilotRequireContractVersion(&errs, "contractVersion", strategy.ContractVersion, AIStrategyContractVersion)
	pilotRequireNonEmpty(&errs, "strategyId", strategy.StrategyID)
	pilotRequireNonEmpty(&errs, "incidentId", strategy.IncidentID)
	pilotRequireAllowed(&errs, "status", strategy.Status, allowedAIStrategyStatuses)
	pilotRequireNonEmpty(&errs, "language", strategy.Language)
	pilotRequireAllowed(&errs, "confidence", strategy.Confidence, allowedAIConfidenceValues)
	pilotRequireNonEmpty(&errs, "audit.promptVersion", strategy.Audit.PromptVersion)
	pilotRequireNonEmpty(&errs, "audit.model", strategy.Audit.Model)
	pilotRequireNonEmpty(&errs, "audit.generatedAt", strategy.Audit.GeneratedAt)
	pilotRequireNonNegativeAIControl(&errs, "audit.quotaLimit", strategy.Audit.QuotaLimit)
	pilotRequireNonNegativeAIControl(&errs, "audit.quotaUsed", strategy.Audit.QuotaUsed)
	pilotRequireNonNegativeAIControl(&errs, "audit.quotaRemaining", strategy.Audit.QuotaRemaining)
	pilotRequireNonNegativeAIControl(&errs, "audit.timeoutBudgetMillis", strategy.Audit.TimeoutBudgetMillis)
	pilotRequireNonNegativeAIControl(&errs, "audit.executionElapsedMillis", strategy.Audit.ExecutionElapsedMillis)
	if !strategy.Audit.RedactionApplied {
		errs = append(errs, fmt.Errorf("audit.redactionApplied: must be true before AI strategy output is used"))
	}

	if strategy.Status == AIStrategyStatusReady {
		pilotRequireNonEmpty(&errs, "sopId", strategy.SOPID)
		pilotRequireNonEmpty(&errs, "sopVersion", strategy.SOPVersion)
		pilotRequireNonEmpty(&errs, "headline", strategy.Headline)
		if len(strategy.Hypotheses) == 0 {
			errs = append(errs, fmt.Errorf("hypotheses: ready strategy must include at least one hypothesis"))
		}
		if len(strategy.FirstActions) == 0 {
			errs = append(errs, fmt.Errorf("firstActions: ready strategy must include at least one action"))
		}
		if len(strategy.EvidenceRefs) == 0 {
			errs = append(errs, fmt.Errorf("evidenceRefs: ready strategy must include at least one evidence ref"))
		}
	}
	if strategy.Status != AIStrategyStatusReady && len(strategy.Limitations) == 0 {
		errs = append(errs, fmt.Errorf("limitations: non-ready strategy must explain the fallback reason"))
	}

	validateAIHypotheses(&errs, strategy.Hypotheses)
	validateAIFirstActions(&errs, strategy.FirstActions)
	validateAIEvidenceRefs(&errs, strategy.EvidenceRefs)
	appendAIStrategySecretAndSafetyErrors(&errs, strategy)

	return errors.Join(errs...)
}

func applyAIStrategyControls(strategy *AIStrategy, controls AIStrategyControls) bool {
	applyAIStrategyControlAudit(strategy, controls)

	if controls.ProviderEnabled != nil && !*controls.ProviderEnabled {
		markAIStrategyFallback(
			strategy,
			AIStrategyStatusUnavailable,
			"AI 제공자가 비활성화되어 SOP 기본 알림만 전송합니다.",
			AIProviderDisabledLimitation,
		)
		return true
	}
	if controls.LicenseAllowed != nil && !*controls.LicenseAllowed {
		markAIStrategyFallback(
			strategy,
			AIStrategyStatusBlockedByPolicy,
			"라이선스 정책상 AI 대응전략을 생성하지 않았습니다.",
			AILicenseUnavailableLimitation,
		)
		return true
	}
	if controls.QuotaLimit > 0 && controls.QuotaUsed >= controls.QuotaLimit {
		markAIStrategyFallback(
			strategy,
			AIStrategyStatusQuotaExhausted,
			"AI 사용량 한도에 도달하여 SOP 기본 알림만 전송합니다.",
			AIQuotaExhaustedLimitation,
		)
		return true
	}
	if controls.TimeoutBudgetMillis > 0 && controls.ExecutionElapsedMillis > controls.TimeoutBudgetMillis {
		markAIStrategyFallback(
			strategy,
			AIStrategyStatusTimeout,
			"AI 대응전략 생성 시간이 초과되어 SOP 기본 알림만 전송합니다.",
			AITimeoutBudgetExceededLimitation,
		)
		return true
	}

	return false
}

func applyAIStrategyControlAudit(strategy *AIStrategy, controls AIStrategyControls) {
	if controls.QuotaLimit != 0 || controls.QuotaUsed != 0 {
		strategy.Audit.QuotaLimit = aiStrategyInt64Pointer(controls.QuotaLimit)
		strategy.Audit.QuotaUsed = aiStrategyInt64Pointer(controls.QuotaUsed)
		if controls.QuotaLimit > 0 {
			remaining := controls.QuotaLimit - controls.QuotaUsed
			if remaining < 0 {
				remaining = 0
			}
			strategy.Audit.QuotaRemaining = aiStrategyInt64Pointer(remaining)
		}
	}
	if controls.TimeoutBudgetMillis != 0 || controls.ExecutionElapsedMillis != 0 {
		strategy.Audit.TimeoutBudgetMillis = aiStrategyInt64Pointer(controls.TimeoutBudgetMillis)
		strategy.Audit.ExecutionElapsedMillis = aiStrategyInt64Pointer(controls.ExecutionElapsedMillis)
	}
}

func markAIStrategyFallback(strategy *AIStrategy, status string, headline string, limitation string) {
	strategy.Status = status
	strategy.Confidence = AIConfidenceLow
	strategy.Headline = headline
	strategy.Hypotheses = nil
	strategy.FirstActions = nil
	strategy.EvidenceRefs = nil
	strategy.CustomerUpdateDraft = ""
	strategy.VendorRequestDraft = ""
	strategy.Limitations = []string{limitation}
}

// AIStrategyIncidentAnnotations converts a validated strategy into public
// Alertmanager annotations consumed by notification templates and webhooks.
func AIStrategyIncidentAnnotations(strategy AIStrategy) map[string]string {
	annotations := make(map[string]string)
	set := func(key string, value string) {
		value = strings.TrimSpace(value)
		if value != "" {
			annotations[key] = value
		}
	}

	set(alertmanagertypes.IncidentAnnotationAIStrategyID, strategy.StrategyID)
	set(alertmanagertypes.IncidentAnnotationAIStrategyStatus, strategy.Status)
	set(alertmanagertypes.IncidentAnnotationAIHeadline, strategy.Headline)
	set(alertmanagertypes.IncidentAnnotationAIConfidence, strategy.Confidence)
	set(alertmanagertypes.IncidentAnnotationCustomerUpdate, strategy.CustomerUpdateDraft)
	set(alertmanagertypes.IncidentAnnotationVendorRequest, strategy.VendorRequestDraft)

	if len(strategy.FirstActions) > 0 {
		set(alertmanagertypes.IncidentAnnotationAIFirstActions, strings.Join(aiFirstActionTexts(strategy.FirstActions), "\n"))
	}
	if len(strategy.Limitations) > 0 {
		set(alertmanagertypes.IncidentAnnotationAILimitations, strings.Join(strategy.Limitations, "\n"))
	}
	if evidenceIDs := evidenceRefIDs(strategy.EvidenceRefs); len(evidenceIDs) > 0 {
		set(alertmanagertypes.IncidentAnnotationAIEvidenceRefs, strings.Join(evidenceIDs, ", "))
	}

	return annotations
}

func baseAIStrategy(req AIStrategyRequest) AIStrategy {
	strategyID := strings.TrimSpace(req.StrategyID)
	if strategyID == "" {
		strategyID = deterministicAIStrategyID(req)
	}

	return AIStrategy{
		ContractVersion:  AIStrategyContractVersion,
		StrategyID:       strategyID,
		IncidentID:       strings.TrimSpace(req.IncidentID),
		AlertFingerprint: strings.TrimSpace(req.AlertFingerprint),
		Status:           AIStrategyStatusUnavailable,
		Language:         firstNonEmpty(req.Language, defaultAIStrategyLanguage),
		Confidence:       AIConfidenceLow,
		Audit: AIStrategyAudit{
			PromptVersion:    firstNonEmpty(req.PromptVersion, defaultAIStrategyPromptVersion),
			Model:            firstNonEmpty(req.Model, defaultAIStrategyModel),
			GeneratedAt:      firstNonEmpty(req.GeneratedAt, defaultAIStrategyGeneratedAt),
			RedactionApplied: true,
		},
	}
}

func validateAIHypotheses(errs *[]error, hypotheses []AIHypothesis) {
	for i, hypothesis := range hypotheses {
		path := fmt.Sprintf("hypotheses[%d]", i)
		if hypothesis.Rank <= 0 {
			*errs = append(*errs, fmt.Errorf("%s.rank: must be greater than zero", path))
		}
		pilotRequireNonEmpty(errs, path+".text", hypothesis.Text)
		pilotRequireAllowed(errs, path+".confidence", hypothesis.Confidence, allowedAIConfidenceValues)
		if len(hypothesis.EvidenceRefs) == 0 && len(hypothesis.SOPStepRefs) == 0 {
			*errs = append(*errs, fmt.Errorf("%s: must include at least one evidenceRefs or sopStepRefs entry", path))
		}
		pilotAppendSecretLikeStringErrors(errs, path+".text", hypothesis.Text)
		for j, ref := range hypothesis.EvidenceRefs {
			pilotRequireNonEmpty(errs, fmt.Sprintf("%s.evidenceRefs[%d]", path, j), ref)
			pilotAppendSecretLikeStringErrors(errs, fmt.Sprintf("%s.evidenceRefs[%d]", path, j), ref)
		}
		for j, ref := range hypothesis.SOPStepRefs {
			pilotRequireNonEmpty(errs, fmt.Sprintf("%s.sopStepRefs[%d]", path, j), ref)
			pilotAppendSecretLikeStringErrors(errs, fmt.Sprintf("%s.sopStepRefs[%d]", path, j), ref)
		}
		if aiContainsAutomaticOperationClaim(hypothesis.Text) {
			*errs = append(*errs, fmt.Errorf("%s.text: must not claim automatic operational execution", path))
		}
	}
}

func validateAIFirstActions(errs *[]error, actions []AIFirstAction) {
	for i, action := range actions {
		path := fmt.Sprintf("firstActions[%d]", i)
		pilotRequireNonEmpty(errs, path+".text", action.Text)
		if action.SOPStepRef == "" && len(action.EvidenceRefs) == 0 {
			*errs = append(*errs, fmt.Errorf("%s: must include sopStepRef or evidenceRefs", path))
		}
		if !action.RequiresHumanApproval {
			*errs = append(*errs, fmt.Errorf("%s.requiresHumanApproval: must be true", path))
		}
		pilotAppendSecretLikeStringErrors(errs, path+".text", action.Text)
		pilotAppendSecretLikeStringErrors(errs, path+".sopStepRef", action.SOPStepRef)
		for j, ref := range action.EvidenceRefs {
			pilotRequireNonEmpty(errs, fmt.Sprintf("%s.evidenceRefs[%d]", path, j), ref)
			pilotAppendSecretLikeStringErrors(errs, fmt.Sprintf("%s.evidenceRefs[%d]", path, j), ref)
		}
		if aiContainsAutomaticOperationClaim(action.Text) {
			*errs = append(*errs, fmt.Errorf("%s.text: must not claim automatic operational execution", path))
		}
	}
}

func validateAIEvidenceRefs(errs *[]error, refs []AIEvidenceRef) {
	for i, ref := range refs {
		path := fmt.Sprintf("evidenceRefs[%d]", i)
		pilotRequireNonEmpty(errs, path+".refId", ref.RefID)
		pilotRequireNonEmpty(errs, path+".type", ref.Type)
		pilotRequireNonEmpty(errs, path+".observation", ref.Observation)
		if strings.TrimSpace(ref.Confidence) != "" {
			pilotRequireAllowed(errs, path+".confidence", ref.Confidence, allowedAIConfidenceValues)
		}
		pilotAppendSecretLikeStringErrors(errs, path+".refId", ref.RefID)
		pilotAppendSecretLikeStringErrors(errs, path+".type", ref.Type)
		pilotAppendSecretLikeStringErrors(errs, path+".observation", ref.Observation)
		pilotAppendSecretLikeStringErrors(errs, path+".permalink", ref.Permalink)
	}
}

func appendAIStrategySecretAndSafetyErrors(errs *[]error, strategy AIStrategy) {
	pilotAppendSecretLikeStringErrors(errs, "strategyId", strategy.StrategyID)
	pilotAppendSecretLikeStringErrors(errs, "incidentId", strategy.IncidentID)
	pilotAppendSecretLikeStringErrors(errs, "alertFingerprint", strategy.AlertFingerprint)
	pilotAppendSecretLikeStringErrors(errs, "sopId", strategy.SOPID)
	pilotAppendSecretLikeStringErrors(errs, "sopVersion", strategy.SOPVersion)
	pilotAppendSecretLikeStringErrors(errs, "headline", strategy.Headline)
	pilotAppendSecretLikeStringErrors(errs, "customerUpdateDraft", strategy.CustomerUpdateDraft)
	pilotAppendSecretLikeStringErrors(errs, "vendorRequestDraft", strategy.VendorRequestDraft)
	pilotAppendSecretLikeStringErrors(errs, "audit.promptVersion", strategy.Audit.PromptVersion)
	pilotAppendSecretLikeStringErrors(errs, "audit.model", strategy.Audit.Model)
	for i, limitation := range strategy.Limitations {
		pilotAppendSecretLikeStringErrors(errs, fmt.Sprintf("limitations[%d]", i), limitation)
	}
	for _, field := range []struct {
		name  string
		value string
	}{
		{"headline", strategy.Headline},
		{"customerUpdateDraft", strategy.CustomerUpdateDraft},
		{"vendorRequestDraft", strategy.VendorRequestDraft},
	} {
		if aiContainsAutomaticOperationClaim(field.value) {
			*errs = append(*errs, fmt.Errorf("%s: must not claim automatic operational execution", field.name))
		}
	}
}

func pilotRequireNonNegativeAIControl(errs *[]error, name string, value *int64) {
	if value != nil && *value < 0 {
		*errs = append(*errs, fmt.Errorf("%s: must be greater than or equal to zero", name))
	}
}

var allowedAIStrategyStatuses = map[string]struct{}{
	AIStrategyStatusReady:               {},
	AIStrategyStatusUnavailable:         {},
	AIStrategyStatusTimeout:             {},
	AIStrategyStatusBlockedByPolicy:     {},
	AIStrategyStatusQuotaExhausted:      {},
	AIStrategyStatusSOPMissing:          {},
	AIStrategyStatusEvidenceUnavailable: {},
	AIStrategyStatusLowConfidence:       {},
}

var allowedAIConfidenceValues = map[string]struct{}{
	AIConfidenceHigh:   {},
	AIConfidenceMedium: {},
	AIConfidenceLow:    {},
}

var automaticOperationClaimPattern = regexp.MustCompile(`(?i)(자동\s*(재시작|삭제|차단|failover)|automatically\s+(restarted|deleted|blocked|failed over)|\b(restarted|deleted|blocked)\b.*\bautomatically\b|재시작했습니다|삭제했습니다|차단했습니다|failover했습니다)`)

func aiContainsAutomaticOperationClaim(value string) bool {
	return automaticOperationClaimPattern.MatchString(strings.TrimSpace(value))
}

func deterministicAIStrategyID(req AIStrategyRequest) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		strings.TrimSpace(req.IncidentID),
		strings.TrimSpace(req.AlertFingerprint),
		strings.TrimSpace(req.SOPDocument.SOPID),
		strings.TrimSpace(req.SOPDocument.Version),
	}, "|")))

	return "AIS-" + hex.EncodeToString(sum[:])[:16]
}

func evidenceRefIDs(refs []AIEvidenceRef) []string {
	ids := make([]string, 0, len(refs))
	for _, ref := range refs {
		if strings.TrimSpace(ref.RefID) != "" {
			ids = append(ids, strings.TrimSpace(ref.RefID))
		}
	}

	return ids
}

func aiFirstActionTexts(actions []AIFirstAction) []string {
	texts := make([]string, 0, len(actions))
	for _, action := range actions {
		text := strings.TrimSpace(action.Text)
		if text != "" {
			texts = append(texts, text)
		}
	}

	return texts
}

func firstActionFromSOP(doc SOPDocument) string {
	for _, line := range strings.Split(doc.BodyMarkdown, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimLeft(line, "-*0123456789. )\t")
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}

	return "SOP 문서의 첫 확인 항목을 검토"
}

func aiStrategyInt64Pointer(value int64) *int64 {
	return &value
}
