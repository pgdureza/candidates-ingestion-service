package domain

import (
	"encoding/json"
	"fmt"
)

// WebhookStrategy interface for different sources
type WebhookStrategy interface {
	Parse(payload []byte) (*CandidateApplication, error)
	Source() string
}

// LinkedInStrategy handles LinkedIn webhook payloads
type LinkedInStrategy struct{}

func (s *LinkedInStrategy) Source() string {
	return "linkedin"
}

func (s *LinkedInStrategy) Parse(payload []byte) (*CandidateApplication, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, fmt.Errorf("invalid linkedin payload: %w", err)
	}

	app := &CandidateApplication{
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

// GoogleFormStrategy handles Google Forms webhook payloads
type GoogleFormStrategy struct{}

func (s *GoogleFormStrategy) Source() string {
	return "google_forms"
}

func (s *GoogleFormStrategy) Parse(payload []byte) (*CandidateApplication, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, fmt.Errorf("invalid google forms payload: %w", err)
	}

	// Google Forms structure: nested responses with field IDs
	responses := raw["responses"].(map[string]interface{})

	app := &CandidateApplication{
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

// StrategyFactory creates appropriate strategy
func StrategyFactory(source string) (WebhookStrategy, error) {
	switch source {
	case "linkedin":
		return &LinkedInStrategy{}, nil
	case "google_forms":
		return &GoogleFormStrategy{}, nil
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
