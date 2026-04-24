package domain

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLinkedInStrategy(t *testing.T) {
	strategy := &LinkedInStrategy{}

	payload := map[string]interface{}{
		"id":        "123",
		"firstName": "John",
		"lastName":  "Doe",
		"email":     "john@example.com",
		"phone":     "555-1234",
		"jobTitle":  "Software Engineer",
	}
	data, _ := json.Marshal(payload)

	app, err := strategy.Parse(data)
	require.NoError(t, err)

	assert.Equal(t, "linkedin", app.Source)
	assert.Equal(t, "123", app.SourceRefID)
	assert.Equal(t, "John", app.FirstName)
	assert.Equal(t, "Doe", app.LastName)
	assert.Equal(t, "john@example.com", app.Email)
	assert.Equal(t, "555-1234", app.Phone)
	assert.Equal(t, "Software Engineer", app.Position)
}

func TestLinkedInStrategyMissingID(t *testing.T) {
	strategy := &LinkedInStrategy{}

	payload := map[string]interface{}{
		"firstName": "John",
	}
	data, _ := json.Marshal(payload)

	_, err := strategy.Parse(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing id")
}

func TestGoogleFormStrategy(t *testing.T) {
	strategy := &GoogleFormStrategy{}

	payload := map[string]interface{}{
		"responseId": "form-123",
		"responses": map[string]interface{}{
			"firstName": "Jane",
			"lastName":  "Smith",
			"email":     "jane@example.com",
			"phone":     "555-5678",
			"position":  "Product Manager",
		},
	}
	data, _ := json.Marshal(payload)

	app, err := strategy.Parse(data)
	require.NoError(t, err)

	assert.Equal(t, "google_forms", app.Source)
	assert.Equal(t, "form-123", app.SourceRefID)
	assert.Equal(t, "Jane", app.FirstName)
	assert.Equal(t, "Smith", app.LastName)
	assert.Equal(t, "jane@example.com", app.Email)
	assert.Equal(t, "555-5678", app.Phone)
	assert.Equal(t, "Product Manager", app.Position)
}

func TestStrategyFactory(t *testing.T) {
	linkedInStrategy, err := StrategyFactory("linkedin")
	require.NoError(t, err)
	assert.Equal(t, "linkedin", linkedInStrategy.Source())

	googleFormStrategy, err := StrategyFactory("google_forms")
	require.NoError(t, err)
	assert.Equal(t, "google_forms", googleFormStrategy.Source())

	_, err = StrategyFactory("unknown")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown source")
}
