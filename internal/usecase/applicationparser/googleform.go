package applicationparser

import (
	"encoding/json"
	"fmt"

	"github.com/candidate-ingestion/service/internal/domain/model"
	"github.com/candidate-ingestion/service/internal/domain/service"
)

var _ service.ApplicationParser = new(GoogleFormParser)

// GoogleFormParser handles Google Forms webhook payloads
type GoogleFormParser struct{}

func (s *GoogleFormParser) Source() string {
	return "google_forms"
}

func (s *GoogleFormParser) Parse(payload []byte) (*model.CandidateApplication, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, fmt.Errorf("invalid google forms payload: %w", err)
	}

	// Google Forms structure: nested responses with field IDs
	responses := raw["responses"].(map[string]interface{})

	app := &model.CandidateApplication{
		Source:      s.Source(),
		RawPayload:  string(payload),
		SourceRefID: toString(raw["responseId"]),
		FirstName:   toString(responses["firstName"]),
		LastName:    toString(responses["lastName"]),
		Email:       toString(responses["email"]),
		Phone:       toString(responses["phone"]),
		Position:    toString(responses["position"]),
	}

	if app.SourceRefID == "" {
		return nil, fmt.Errorf("google forms payload missing responseId")
	}

	return app, nil
}
