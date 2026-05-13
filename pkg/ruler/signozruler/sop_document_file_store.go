package signozruler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

const (
	sopDocumentFileStoreContractVersion = "ds.sop_document_store.v1"
	sopDocumentFileStorePathEnv         = "DS_APM_SOP_DOCUMENT_STORE_PATH"
	sopDocumentFileStoreDefaultPath     = "var/ds-apm/sop-documents.json"
)

type sopDocumentFileStoreSnapshot struct {
	ContractVersion string                  `json:"contractVersion"`
	Documents       []ruletypes.SOPDocument `json:"documents"`
}

func defaultSOPDocumentFileStorePath() string {
	if path := strings.TrimSpace(os.Getenv(sopDocumentFileStorePathEnv)); path != "" {
		return path
	}

	return sopDocumentFileStoreDefaultPath
}

func loadSOPDocumentFileStore(path string) (map[string]ruletypes.SOPDocument, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return map[string]ruletypes.SOPDocument{}, nil
	}

	payload, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]ruletypes.SOPDocument{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("SOP document file store: read %q: %w", path, err)
	}
	if strings.TrimSpace(string(payload)) == "" {
		return nil, fmt.Errorf("SOP document file store: %q is empty", path)
	}

	var snapshot sopDocumentFileStoreSnapshot
	if err := json.Unmarshal(payload, &snapshot); err != nil {
		return nil, fmt.Errorf("SOP document file store: decode %q: %w", path, err)
	}
	if snapshot.ContractVersion != sopDocumentFileStoreContractVersion {
		return nil, fmt.Errorf("SOP document file store: unsupported contractVersion %q", snapshot.ContractVersion)
	}

	docs := make(map[string]ruletypes.SOPDocument, len(snapshot.Documents))
	for i, doc := range snapshot.Documents {
		if err := ruletypes.ValidateSOPDocument(doc); err != nil {
			return nil, fmt.Errorf("SOP document file store: documents[%d]: %w", i, err)
		}
		key := sopDocumentKey(doc.SOPID, doc.Version)
		if _, exists := docs[key]; exists {
			return nil, fmt.Errorf("SOP document file store: duplicate document %q version %q", doc.SOPID, doc.Version)
		}
		docs[key] = doc
	}

	return docs, nil
}

func saveSOPDocumentFileStore(path string, docs []ruletypes.SOPDocument) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("SOP document file store: path must not be empty")
	}

	docs = sortedSOPDocuments(docs)
	for i, doc := range docs {
		if err := ruletypes.ValidateSOPDocument(doc); err != nil {
			return fmt.Errorf("SOP document file store: documents[%d]: %w", i, err)
		}
	}

	snapshot := sopDocumentFileStoreSnapshot{
		ContractVersion: sopDocumentFileStoreContractVersion,
		Documents:       docs,
	}
	payload, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("SOP document file store: encode: %w", err)
	}
	payload = append(payload, '\n')

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("SOP document file store: mkdirall: %w", err)
	}

	tmpPath := fmt.Sprintf("%s.tmp.%d", path, time.Now().UTC().UnixNano())
	if err := os.WriteFile(tmpPath, payload, 0o600); err != nil {
		return fmt.Errorf("SOP document file store: write temp: %w", err)
	}
	defer os.Remove(tmpPath) //nolint:errcheck

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("SOP document file store: rename: %w", err)
	}

	return nil
}

func sortedSOPDocumentsFromMap(docs map[string]ruletypes.SOPDocument) []ruletypes.SOPDocument {
	result := make([]ruletypes.SOPDocument, 0, len(docs))
	for _, doc := range docs {
		result = append(result, doc)
	}

	return sortedSOPDocuments(result)
}

func sortedSOPDocuments(docs []ruletypes.SOPDocument) []ruletypes.SOPDocument {
	result := append([]ruletypes.SOPDocument(nil), docs...)
	sort.Slice(result, func(i, j int) bool {
		if result[i].SOPID == result[j].SOPID {
			return result[i].Version < result[j].Version
		}
		return result[i].SOPID < result[j].SOPID
	})

	return result
}
