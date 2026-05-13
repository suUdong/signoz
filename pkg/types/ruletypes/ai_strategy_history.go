package ruletypes

import (
	"errors"
	"fmt"
	"strings"
)

const (
	AIStrategyHistoryContractVersion = "ds.ai_strategy_history.v1"
)

type AIStrategyHistoryLookupRequest struct {
	IncidentID       string `json:"incidentId,omitempty" schema:"incidentId"`
	AlertFingerprint string `json:"alertFingerprint,omitempty" schema:"alertFingerprint"`
}

type AIStrategyHistoryRecord struct {
	ContractVersion  string     `json:"contractVersion"`
	IncidentID       string     `json:"incidentId"`
	AlertFingerprint string     `json:"alertFingerprint,omitempty"`
	StrategyID       string     `json:"strategyId"`
	Status           string     `json:"status"`
	SOPID            string     `json:"sopId,omitempty"`
	SOPVersion       string     `json:"sopVersion,omitempty"`
	Confidence       string     `json:"confidence"`
	GeneratedAt      string     `json:"generatedAt"`
	Strategy         AIStrategy `json:"strategy"`
}

func NewAIStrategyHistoryRecord(strategy AIStrategy) (AIStrategyHistoryRecord, error) {
	record := AIStrategyHistoryRecord{
		ContractVersion:  AIStrategyHistoryContractVersion,
		IncidentID:       strings.TrimSpace(strategy.IncidentID),
		AlertFingerprint: strings.TrimSpace(strategy.AlertFingerprint),
		StrategyID:       strings.TrimSpace(strategy.StrategyID),
		Status:           strings.TrimSpace(strategy.Status),
		SOPID:            strings.TrimSpace(strategy.SOPID),
		SOPVersion:       strings.TrimSpace(strategy.SOPVersion),
		Confidence:       strings.TrimSpace(strategy.Confidence),
		GeneratedAt:      strings.TrimSpace(strategy.Audit.GeneratedAt),
		Strategy:         strategy,
	}

	return record, ValidateAIStrategyHistoryRecord(record)
}

func ValidateAIStrategyHistoryRecord(record AIStrategyHistoryRecord) error {
	var errs []error

	pilotRequireContractVersion(&errs, "contractVersion", record.ContractVersion, AIStrategyHistoryContractVersion)
	pilotRequireNonEmpty(&errs, "incidentId", record.IncidentID)
	pilotRequireNonEmpty(&errs, "strategyId", record.StrategyID)
	pilotRequireAllowed(&errs, "status", record.Status, allowedAIStrategyStatuses)
	pilotRequireAllowed(&errs, "confidence", record.Confidence, allowedAIConfidenceValues)
	pilotRequireNonEmpty(&errs, "generatedAt", record.GeneratedAt)
	pilotAppendSecretLikeStringErrors(&errs, "incidentId", record.IncidentID)
	pilotAppendSecretLikeStringErrors(&errs, "alertFingerprint", record.AlertFingerprint)
	pilotAppendSecretLikeStringErrors(&errs, "strategyId", record.StrategyID)
	pilotAppendSecretLikeStringErrors(&errs, "sopId", record.SOPID)
	pilotAppendSecretLikeStringErrors(&errs, "sopVersion", record.SOPVersion)
	pilotAppendSecretLikeStringErrors(&errs, "generatedAt", record.GeneratedAt)

	if err := ValidateAIStrategy(record.Strategy); err != nil {
		errs = append(errs, fmt.Errorf("strategy: %w", err))
	}
	if strings.TrimSpace(record.Strategy.IncidentID) != strings.TrimSpace(record.IncidentID) {
		errs = append(errs, fmt.Errorf("strategy.incidentId: must match history incidentId"))
	}
	if strings.TrimSpace(record.Strategy.AlertFingerprint) != strings.TrimSpace(record.AlertFingerprint) {
		errs = append(errs, fmt.Errorf("strategy.alertFingerprint: must match history alertFingerprint"))
	}
	if strings.TrimSpace(record.Strategy.StrategyID) != strings.TrimSpace(record.StrategyID) {
		errs = append(errs, fmt.Errorf("strategy.strategyId: must match history strategyId"))
	}
	if strings.TrimSpace(record.Strategy.Status) != strings.TrimSpace(record.Status) {
		errs = append(errs, fmt.Errorf("strategy.status: must match history status"))
	}
	if strings.TrimSpace(record.Strategy.Confidence) != strings.TrimSpace(record.Confidence) {
		errs = append(errs, fmt.Errorf("strategy.confidence: must match history confidence"))
	}
	if strings.TrimSpace(record.Strategy.Audit.GeneratedAt) != strings.TrimSpace(record.GeneratedAt) {
		errs = append(errs, fmt.Errorf("strategy.audit.generatedAt: must match history generatedAt"))
	}

	return errors.Join(errs...)
}

func ValidateAIStrategyHistoryLookup(req AIStrategyHistoryLookupRequest) error {
	var errs []error

	if len(AIStrategyHistoryLookupKeys(req)) == 0 {
		errs = append(errs, fmt.Errorf("incidentId or alertFingerprint: at least one lookup key is required"))
	}
	pilotAppendSecretLikeStringErrors(&errs, "incidentId", req.IncidentID)
	pilotAppendSecretLikeStringErrors(&errs, "alertFingerprint", req.AlertFingerprint)

	return errors.Join(errs...)
}

func AIStrategyHistoryLookupKeys(req AIStrategyHistoryLookupRequest) []string {
	keys := make([]string, 0, 2)
	if incidentID := strings.TrimSpace(req.IncidentID); incidentID != "" {
		keys = append(keys, "incident\x00"+incidentID)
	}
	if fingerprint := strings.TrimSpace(req.AlertFingerprint); fingerprint != "" {
		keys = append(keys, "fingerprint\x00"+fingerprint)
	}

	return keys
}

func AIStrategyHistoryLookupFromStrategy(strategy AIStrategy) AIStrategyHistoryLookupRequest {
	return AIStrategyHistoryLookupRequest{
		IncidentID:       strategy.IncidentID,
		AlertFingerprint: strategy.AlertFingerprint,
	}
}
