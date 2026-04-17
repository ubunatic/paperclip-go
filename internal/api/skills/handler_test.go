package skills_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ubunatic/paperclip-go/internal/api/skills"
	"github.com/ubunatic/paperclip-go/internal/domain"
)

func TestSkillsHandler(t *testing.T) {
	// Create some test skills
	testSkills := []domain.Skill{
		{
			Name:        "skill-one",
			Description: "First test skill",
			Path:        "/skills/skill-one/SKILL.md",
			Body:        "# Skill One",
		},
		{
			Name:        "skill-two",
			Description: "Second test skill",
			Path:        "/skills/skill-two/SKILL.md",
			Body:        "# Skill Two",
		},
	}

	// Create handler with test skills
	handler := skills.Handler(testSkills)

	// Make a request
	req := httptest.NewRequest("GET", "/api/skills", nil)
	w := httptest.NewRecorder()

	// Call handler
	handler(w, req)

	// Verify status code
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	// Verify response structure
	var response map[string]any
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	// Verify 'items' key exists
	items, ok := response["items"]
	if !ok {
		t.Fatalf("expected 'items' key in response, got %v", response)
	}

	// Verify items is an array
	itemsArray, ok := items.([]any)
	if !ok {
		t.Fatalf("expected items to be array, got %T", items)
	}

	// Verify we got 2 items
	if len(itemsArray) != 2 {
		t.Fatalf("expected 2 items, got %d", len(itemsArray))
	}

	// Verify structure of first item
	firstItem, ok := itemsArray[0].(map[string]any)
	if !ok {
		t.Fatalf("expected item to be map, got %T", itemsArray[0])
	}

	// Verify required fields
	if name, ok := firstItem["name"]; !ok || name == "" {
		t.Errorf("expected 'name' field in item, got %v", firstItem)
	}
	if _, ok := firstItem["description"]; !ok {
		t.Errorf("expected 'description' field in item, got %v", firstItem)
	}
	if _, ok := firstItem["path"]; !ok {
		t.Errorf("expected 'path' field in item, got %v", firstItem)
	}
	if _, ok := firstItem["body"]; !ok {
		t.Errorf("expected 'body' field in item, got %v", firstItem)
	}

	// Verify specific values
	if firstItem["name"] != "skill-one" {
		t.Errorf("expected name 'skill-one', got %v", firstItem["name"])
	}
	if firstItem["description"] != "First test skill" {
		t.Errorf("expected description 'First test skill', got %v", firstItem["description"])
	}
}

func TestSkillsHandlerEmpty(t *testing.T) {
	// Create handler with no skills
	handler := skills.Handler([]domain.Skill{})

	// Make a request
	req := httptest.NewRequest("GET", "/api/skills", nil)
	w := httptest.NewRecorder()

	// Call handler
	handler(w, req)

	// Verify status code
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	// Verify response structure
	var response map[string]any
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	// Verify 'items' key exists
	items, ok := response["items"]
	if !ok {
		t.Fatalf("expected 'items' key in response, got %v", response)
	}

	// Verify items is an empty array
	itemsArray, ok := items.([]any)
	if !ok {
		t.Fatalf("expected items to be array, got %T", items)
	}
	if len(itemsArray) != 0 {
		t.Errorf("expected 0 items, got %d", len(itemsArray))
	}
}
