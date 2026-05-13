package ruletypes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewAIStrategyHistoryRecordSummarizesValidatedStrategy(t *testing.T) {
	strategy, err := GenerateLocalAIStrategy(validAIStrategyRequest())
	require.NoError(t, err)

	record, err := NewAIStrategyHistoryRecord(strategy)

	require.NoError(t, err)
	require.Equal(t, AIStrategyHistoryContractVersion, record.ContractVersion)
	require.Equal(t, strategy.IncidentID, record.IncidentID)
	require.Equal(t, strategy.AlertFingerprint, record.AlertFingerprint)
	require.Equal(t, strategy.StrategyID, record.StrategyID)
	require.Equal(t, strategy.Status, record.Status)
	require.Equal(t, strategy.SOPID, record.SOPID)
	require.Equal(t, strategy.SOPVersion, record.SOPVersion)
	require.Equal(t, strategy.Audit.GeneratedAt, record.GeneratedAt)
	require.Equal(t, strategy, record.Strategy)
	require.NoError(t, ValidateAIStrategyHistoryRecord(record))
}

func TestValidateAIStrategyHistoryRecordRejectsMismatchedSummary(t *testing.T) {
	strategy, err := GenerateLocalAIStrategy(validAIStrategyRequest())
	require.NoError(t, err)
	record, err := NewAIStrategyHistoryRecord(strategy)
	require.NoError(t, err)
	record.Status = AIStrategyStatusTimeout

	err = ValidateAIStrategyHistoryRecord(record)

	require.ErrorContains(t, err, "strategy.status: must match history status")
}

func TestAIStrategyHistoryLookupKeysPreferIncidentThenFingerprint(t *testing.T) {
	got := AIStrategyHistoryLookupKeys(AIStrategyHistoryLookupRequest{
		IncidentID:       " INC-20260513-001 ",
		AlertFingerprint: " fp-payment-api-5xx ",
	})

	require.Equal(t, []string{
		"incident\x00INC-20260513-001",
		"fingerprint\x00fp-payment-api-5xx",
	}, got)
	require.NoError(t, ValidateAIStrategyHistoryLookup(AIStrategyHistoryLookupRequest{
		AlertFingerprint: "fp-payment-api-5xx",
	}))
	require.ErrorContains(
		t,
		ValidateAIStrategyHistoryLookup(AIStrategyHistoryLookupRequest{}),
		"at least one lookup key is required",
	)
}
