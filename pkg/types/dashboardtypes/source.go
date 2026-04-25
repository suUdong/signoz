package dashboardtypes

import (
	"encoding/json"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/valuer"
)

type Source struct {
	valuer.String
}

var (
	SourceAIO11yOverview = Source{valuer.NewString("ai-o11y-overview")}
)

var AllSources = []Source{
	SourceAIO11yOverview,
}

func NewSource(s string) (Source, error) {
	switch s {
	case SourceAIO11yOverview.StringValue():
		return SourceAIO11yOverview, nil
	default:
		return Source{}, errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "invalid system dashboard source: %s", s)
	}
}

func (source *Source) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	s, err := NewSource(str)
	if err != nil {
		return err
	}

	*source = s
	return nil
}
