package candidateingestion

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoogleFormStrategy(t *testing.T) {
	strategy := &GoogleFormParser{}

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
