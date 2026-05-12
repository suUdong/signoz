package ruletypes

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateSOPDocumentAcceptsManagedMarkdownDocument(t *testing.T) {
	doc := validSOPDocument()

	require.NoError(t, ValidateSOPDocument(doc))
	require.Equal(t, SOPDocumentContractVersion, doc.ContractVersion)
	require.Equal(t, PilotSOPSourceKindManagedMarkdown, doc.Source.Type)
	require.True(t, strings.HasPrefix(doc.Checksum, "sha256:"))
	require.False(t, doc.SecurityContext.SecretRefVisible)
	require.False(t, doc.SecurityContext.BrowserCredentialsUsed)
	require.True(t, doc.SecurityContext.RedactionApplied)
}

func TestNewSOPDocumentFromManagedMarkdownNormalizesAndValidates(t *testing.T) {
	source := validPilotManagedMarkdownSource()
	source.ServiceAccountProfile = " ds-sop-reader "
	source.Documents[0].SOPID = " SOP-PAY-001 "
	source.Documents[0].Title = " Payment API 5xx response "

	doc := NewSOPDocumentFromManagedMarkdown(source, source.Documents[0], " payments ", SOPApprovalStatusApproved)

	require.Equal(t, "SOP-PAY-001", doc.SOPID)
	require.Equal(t, "Payment API 5xx response", doc.Title)
	require.Equal(t, "payments", doc.OwnerTeam)
	require.Equal(t, "ds-sop-reader", doc.SecurityContext.ServiceAccountProfile)
	require.NoError(t, ValidateSOPDocument(doc))
}

func TestValidateSOPDocumentRejectsRequiredFieldAndEnumGaps(t *testing.T) {
	missingBody := validSOPDocument()
	missingBody.BodyMarkdown = ""
	require.ErrorContains(t, ValidateSOPDocument(missingBody), "bodyMarkdown: field is required")

	missingChecksum := validSOPDocument()
	missingChecksum.Checksum = ""
	require.ErrorContains(t, ValidateSOPDocument(missingChecksum), "checksum: field is required")

	unsupportedStatus := validSOPDocument()
	unsupportedStatus.ApprovalStatus = "half_approved"
	require.ErrorContains(t, ValidateSOPDocument(unsupportedStatus), `approvalStatus: unsupported value "half_approved"`)

	unsupportedSource := validSOPDocument()
	unsupportedSource.Source.Type = "wiki_export"
	require.ErrorContains(t, ValidateSOPDocument(unsupportedSource), `source.type: unsupported value "wiki_export"`)
}

func TestValidateSOPDocumentRejectsUnsafeMarkdownPayloads(t *testing.T) {
	secretBody := validSOPDocument()
	secretBody.BodyMarkdown = "Rotate with access_token=hidden"
	require.ErrorContains(t, ValidateSOPDocument(secretBody), "bodyMarkdown: contains secret-like value")

	htmlBody := validSOPDocument()
	htmlBody.BodyMarkdown = "<!DOCTYPE html><html><body>SOP</body></html>"
	require.ErrorContains(t, ValidateSOPDocument(htmlBody), "bodyMarkdown: payload does not look like markdown")

	oversizedBody := validSOPDocument()
	oversizedBody.BodyMarkdown = strings.Repeat("a", SOPDocumentBodyMarkdownMaxBytes+1)
	require.ErrorContains(t, ValidateSOPDocument(oversizedBody), "bodyMarkdown: exceeds max size")
}

func TestValidateSOPDocumentRejectsCredentialBearingDisplayURL(t *testing.T) {
	doc := validSOPDocument()
	doc.DisplayURL = "https://kb.example/sop/SOP-PAY-001?token=hidden"

	err := ValidateSOPDocument(doc)
	require.Error(t, err)
	require.Contains(t, err.Error(), "displayUrl")
	require.Contains(t, err.Error(), "contains secret-like value")
}

func TestValidateSOPDocumentRejectsCredentialFlags(t *testing.T) {
	t.Run("secretRefVisible", func(t *testing.T) {
		doc := validSOPDocument()
		doc.SecurityContext.SecretRefVisible = true
		require.ErrorContains(t, ValidateSOPDocument(doc), "securityContext.secretRefVisible")
	})

	t.Run("browserCredentialsUsed", func(t *testing.T) {
		doc := validSOPDocument()
		doc.SecurityContext.BrowserCredentialsUsed = true
		require.ErrorContains(t, ValidateSOPDocument(doc), "securityContext.browserCredentialsUsed")
	})

	t.Run("redactionRequired", func(t *testing.T) {
		doc := validSOPDocument()
		doc.SecurityContext.RedactionApplied = false
		require.ErrorContains(t, ValidateSOPDocument(doc), "securityContext.redactionApplied")
	})
}

func TestValidateSOPDocumentAcceptsAllApprovalStatuses(t *testing.T) {
	for _, status := range []string{
		SOPApprovalStatusDraft,
		SOPApprovalStatusApproved,
		SOPApprovalStatusDeprecated,
		SOPApprovalStatusDisabled,
	} {
		t.Run(status, func(t *testing.T) {
			doc := validSOPDocument()
			doc.ApprovalStatus = status
			require.NoError(t, ValidateSOPDocument(doc))
		})
	}
}

func TestNewSOPDocumentListResponseOmitsBodyMarkdown(t *testing.T) {
	resp := NewSOPDocumentListResponse([]SOPDocument{validSOPDocument()})

	require.Equal(t, SOPDocumentListContractVersion, resp.ContractVersion)
	require.Len(t, resp.Documents, 1)
	require.Equal(t, "SOP-PAY-001", resp.Documents[0].SOPID)

	raw, err := json.Marshal(resp)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "bodyMarkdown")
}

func TestPreviewSOPDocumentBindingResolvesExplicitLabel(t *testing.T) {
	resp, err := PreviewSOPDocumentBinding([]SOPDocument{validSOPDocument()}, SOPBindingPreviewRequest{
		Labels: map[string]string{"sop_id": "SOP-PAY-001"},
	})

	require.NoError(t, err)
	require.Equal(t, SOPBindingContractVersion, resp.ContractVersion)
	require.Equal(t, SOPBindingStatusBound, resp.Status)
	require.Equal(t, SOPBindingResolutionExplicitLabel, resp.Resolution)
	require.Equal(t, "SOP-PAY-001", resp.SOPID)
	require.Equal(t, "Payment API 5xx response", resp.Title)
	require.NoError(t, ValidateSOPBindingPreviewResponse(resp))
}

func TestPreviewSOPDocumentBindingReportsMissingAndDisabled(t *testing.T) {
	missing, err := PreviewSOPDocumentBinding([]SOPDocument{validSOPDocument()}, SOPBindingPreviewRequest{
		Labels: map[string]string{"sop_id": "SOP-UNKNOWN"},
	})
	require.NoError(t, err)
	require.Equal(t, SOPBindingStatusMissing, missing.Status)
	require.Equal(t, SOPBindingResolutionExplicitLabel, missing.Resolution)

	disabledDoc := validSOPDocument()
	disabledDoc.ApprovalStatus = SOPApprovalStatusDisabled
	disabled, err := PreviewSOPDocumentBinding([]SOPDocument{disabledDoc}, SOPBindingPreviewRequest{
		Labels: map[string]string{"sop_id": "SOP-PAY-001"},
	})
	require.NoError(t, err)
	require.Equal(t, SOPBindingStatusDisabled, disabled.Status)
	require.Contains(t, disabled.Warnings, "sop document is disabled")
}

func validSOPDocument() SOPDocument {
	source := validPilotManagedMarkdownSource()
	return NewSOPDocumentFromManagedMarkdown(source, source.Documents[0], "payments", SOPApprovalStatusApproved)
}
