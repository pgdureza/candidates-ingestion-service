package applicationparser

import (
	"fmt"

	"github.com/candidate-ingestion/service/internal/domain/service"
)

// NewCandidateApplicationParser creates appropriate strategy based on source
func NewCandidateApplicationParser(source string) (service.ApplicationParser, error) {
	switch source {
	case "linkedin":
		return &LinkedInParser{}, nil
	case "google_forms":
		return &GoogleFormParser{}, nil
	default:
		return nil, fmt.Errorf("unknown source: %s", source)
	}
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}
