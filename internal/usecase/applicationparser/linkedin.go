package applicationparser

import (
	"encoding/json"
	"fmt"

	"github.com/candidate-ingestion/service/internal/domain/model"
	"github.com/candidate-ingestion/service/internal/domain/service"
)

var _ service.ApplicationParser = new(LinkedInParser)

// LinkedInParser handles LinkedIn webhook payloads
type LinkedInParser struct{}

func (s *LinkedInParser) Source() string {
	return "linkedin"
}

func (s *LinkedInParser) Parse(payload []byte) (*model.CandidateApplication, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, fmt.Errorf("invalid linkedin payload: %w", err)
	}

	app := &model.CandidateApplication{
		Source:      s.Source(),
		RawPayload:  string(payload),
		SourceRefID: toString(raw["id"]),
		FirstName:   toString(raw["firstName"]),
		LastName:    toString(raw["lastName"]),
		Email:       toString(raw["email"]),
		Phone:       toString(raw["phone"]),
		Position:    toString(raw["jobTitle"]),
	}

	if app.SourceRefID == "" {
		return nil, fmt.Errorf("linkedin payload missing id")
	}

	return app, nil
}
