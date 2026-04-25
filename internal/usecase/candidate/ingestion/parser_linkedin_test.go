package candidateingestion

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLinkedInStrategy(t *testing.T) {
	strategy := &LinkedInParser{}

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
	strategy := &LinkedInParser{}

	payload := map[string]interface{}{
		"firstName": "John",
	}
	data, _ := json.Marshal(payload)

	_, err := strategy.Parse(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing id")
}
