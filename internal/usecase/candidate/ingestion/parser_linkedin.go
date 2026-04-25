package candidateingestion

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/candidate-ingestion/service/internal/domain/model"
	"github.com/candidate-ingestion/service/internal/domain/service"
)

var _ service.CandidateParser = new(LinkedInParser)

// LinkedInParser handles LinkedIn webhook payloads
type LinkedInParser struct{}

func (s *LinkedInParser) Source() string {
	return "linkedin"
}

func (s *LinkedInParser) Parse(payload []byte) (*model.Candidate, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, fmt.Errorf("invalid linkedin payload: %w", err)
	}

	currentTime := time.Now().UTC()

	app := &model.Candidate{
		ID:          fmt.Sprintf("%s-%s", s.Source(), raw["id"]),
		Source:      s.Source(),
		RawPayload:  string(payload),
		SourceRefID: toString(raw["id"]),
		FirstName:   toString(raw["firstName"]),
		LastName:    toString(raw["lastName"]),
		Email:       toString(raw["email"]),
		Phone:       toString(raw["phone"]),
		Position:    toString(raw["jobTitle"]),
		CreatedAt:   currentTime,
		UpdatedAt:   currentTime,
	}

	if app.SourceRefID == "" {
		return nil, fmt.Errorf("linkedin payload missing id")
	}

	return app, nil
}
