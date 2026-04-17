package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/ubunatic/paperclip-go/internal/domain"
	"github.com/ubunatic/paperclip-go/internal/testutil"
)

func TestCompaniesE2E(t *testing.T) {
	srv, _ := testutil.SpawnTestServer(t) // store managed by t.Cleanup

	// POST /api/companies → 201 + id
	body, _ := json.Marshal(map[string]string{
		"name":        "Acme Corp",
		"shortname":   "acme",
		"description": "Test company",
	})
	resp, err := http.Post(srv.URL+"/api/companies", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /api/companies: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST /api/companies status = %d, want 201", resp.StatusCode)
	}

	var created map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decoding POST response: %v", err)
	}
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatalf("expected id in POST response, got %v", created)
	}

	// POST with missing fields → 422
	badBody, _ := json.Marshal(map[string]string{"name": "No Shortname"})
	resp2, err := http.Post(srv.URL+"/api/companies", "application/json", bytes.NewReader(badBody))
	if err != nil {
		t.Fatalf("POST bad body: %v", err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("POST bad body status = %d, want 422", resp2.StatusCode)
	}

	// GET /api/companies → list with 1 item
	resp3, err := http.Get(srv.URL + "/api/companies")
	if err != nil {
		t.Fatalf("GET /api/companies: %v", err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/companies status = %d, want 200", resp3.StatusCode)
	}
	var list map[string]any
	if err := json.NewDecoder(resp3.Body).Decode(&list); err != nil {
		t.Fatalf("decoding list response: %v", err)
	}
	items, _ := list["items"].([]any)
	if len(items) != 1 {
		t.Errorf("list items len = %d, want 1", len(items))
	}

	// GET /api/companies/{id} → 200
	resp4, err := http.Get(srv.URL + "/api/companies/" + id)
	if err != nil {
		t.Fatalf("GET /api/companies/%s: %v", id, err)
	}
	resp4.Body.Close()
	if resp4.StatusCode != http.StatusOK {
		t.Errorf("GET by id status = %d, want 200", resp4.StatusCode)
	}

	// GET /api/companies/nonexistent → 404
	resp5, err := http.Get(srv.URL + "/api/companies/nonexistent-id")
	if err != nil {
		t.Fatalf("GET nonexistent: %v", err)
	}
	resp5.Body.Close()
	if resp5.StatusCode != http.StatusNotFound {
		t.Errorf("GET nonexistent status = %d, want 404", resp5.StatusCode)
	}
}

func TestAgentsE2E(t *testing.T) {
	srv, _ := testutil.SpawnTestServer(t)

	// Create a company first
	companyBody, _ := json.Marshal(map[string]string{
		"name":        "Test Corp",
		"shortname":   "test",
		"description": "Test company",
	})
	respCompany, err := http.Post(srv.URL+"/api/companies", "application/json", bytes.NewReader(companyBody))
	if err != nil {
		t.Fatalf("POST /api/companies: %v", err)
	}
	var company map[string]any
	if err := json.NewDecoder(respCompany.Body).Decode(&company); err != nil {
		t.Fatalf("decoding company response: %v", err)
	}
	respCompany.Body.Close()
	companyID, _ := company["id"].(string)

	// POST /api/agents → 201
	agentBody, _ := json.Marshal(map[string]any{
		"companyId":   companyID,
		"shortname":   "alice",
		"displayName": "Alice",
		"role":        "manager",
		"adapter":     "stub",
	})
	resp, err := http.Post(srv.URL+"/api/agents", "application/json", bytes.NewReader(agentBody))
	if err != nil {
		t.Fatalf("POST /api/agents: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST /api/agents status = %d, want 201", resp.StatusCode)
	}

	var created map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decoding POST response: %v", err)
	}
	agentID, _ := created["id"].(string)
	if agentID == "" {
		t.Fatalf("expected id in POST response, got %v", created)
	}

	// GET /api/agents/{id} → 200
	resp2, err := http.Get(srv.URL + "/api/agents/" + agentID)
	if err != nil {
		t.Fatalf("GET /api/agents/%s: %v", agentID, err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("GET /api/agents/%s status = %d, want 200", agentID, resp2.StatusCode)
	}

	// GET /api/agents?companyId=... → list with 1 item
	resp3, err := http.Get(srv.URL + "/api/agents?companyId=" + companyID)
	if err != nil {
		t.Fatalf("GET /api/agents: %v", err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/agents status = %d, want 200", resp3.StatusCode)
	}
	var agents map[string]any
	if err := json.NewDecoder(resp3.Body).Decode(&agents); err != nil {
		t.Fatalf("decoding agents list: %v", err)
	}
	agentItems, _ := agents["items"].([]any)
	if len(agentItems) != 1 {
		t.Errorf("agents list len = %d, want 1", len(agentItems))
	}

	// GET /api/agents/me with X-Agent-Id header
	req, _ := http.NewRequest("GET", srv.URL+"/api/agents/me", nil)
	req.Header.Set("X-Agent-Id", agentID)
	resp4, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /api/agents/me: %v", err)
	}
	defer resp4.Body.Close()
	if resp4.StatusCode != http.StatusOK {
		t.Errorf("GET /api/agents/me status = %d, want 200", resp4.StatusCode)
	}

	// GET /api/agents/me without header → 400
	resp5, err := http.Get(srv.URL + "/api/agents/me")
	if err != nil {
		t.Fatalf("GET /api/agents/me no header: %v", err)
	}
	resp5.Body.Close()
	if resp5.StatusCode != http.StatusBadRequest {
		t.Errorf("GET /api/agents/me without header status = %d, want 400", resp5.StatusCode)
	}
}

func TestActivityE2E(t *testing.T) {
	srv, store := testutil.SpawnTestServer(t)

	// Create a company
	companyBody, _ := json.Marshal(map[string]string{
		"name":        "Test Corp",
		"shortname":   "test",
		"description": "Test company",
	})
	respCompany, err := http.Post(srv.URL+"/api/companies", "application/json", bytes.NewReader(companyBody))
	if err != nil {
		t.Fatalf("POST /api/companies: %v", err)
	}
	var company map[string]any
	if err := json.NewDecoder(respCompany.Body).Decode(&company); err != nil {
		t.Fatalf("decoding company response: %v", err)
	}
	respCompany.Body.Close()
	companyID, _ := company["id"].(string)

	// Record activities directly to the store
	ctx := respCompany.Request.Context()
	activityLog := testutil.SpawnActivityLog(store)
	for i := 0; i < 3; i++ {
		if err := activityLog.Record(ctx, companyID, "agent", "agent-123", "action", "entity", "entity-id", "{}"); err != nil {
			t.Fatalf("recording activity %d: %v", i, err)
		}
	}

	// GET /api/activity?companyId=... → list
	resp, err := http.Get(srv.URL + "/api/activity?companyId=" + companyID)
	if err != nil {
		t.Fatalf("GET /api/activity: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/activity status = %d, want 200", resp.StatusCode)
	}

	var activities map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&activities); err != nil {
		t.Fatalf("decoding activities: %v", err)
	}
	items, _ := activities["items"].([]any)
	if len(items) != 3 {
		t.Errorf("activities list len = %d, want 3", len(items))
	}

	// GET /api/activity without companyId → 400
	resp2, err := http.Get(srv.URL + "/api/activity")
	if err != nil {
		t.Fatalf("GET /api/activity: %v", err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusBadRequest {
		t.Errorf("GET /api/activity without companyId status = %d, want 400", resp2.StatusCode)
	}
}

func TestSkillsE2E(t *testing.T) {
	// Create a temporary directory with synthetic SKILL.md files
	tempdir := t.TempDir()

	// Create a test skill directory
	skillDir := filepath.Join(tempdir, "test-skill")
	if err := os.Mkdir(skillDir, 0755); err != nil {
		t.Fatalf("creating skill dir: %v", err)
	}

	// Create a SKILL.md file
	skillPath := filepath.Join(skillDir, "SKILL.md")
	skillContent := `---
name: test-skill-e2e
description: A test skill for E2E testing
---
# Test Skill

This is a test skill for E2E testing.
`
	if err := os.WriteFile(skillPath, []byte(skillContent), 0644); err != nil {
		t.Fatalf("writing SKILL.md: %v", err)
	}

	srv, _ := testutil.SpawnTestServerWithSkillsDir(t, tempdir)

	// GET /api/skills → list
	resp, err := http.Get(srv.URL + "/api/skills")
	if err != nil {
		t.Fatalf("GET /api/skills: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/skills status = %d, want 200", resp.StatusCode)
	}

	var skillsResponse map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&skillsResponse); err != nil {
		t.Fatalf("decoding skills response: %v", err)
	}

	// Verify response has items key
	items, ok := skillsResponse["items"]
	if !ok {
		t.Fatalf("expected 'items' key in response, got %v", skillsResponse)
	}

	itemsList, ok := items.([]any)
	if !ok {
		t.Fatalf("expected items to be array, got %T", items)
	}

	// Should have loaded exactly 1 skill from our temporary directory
	if len(itemsList) != 1 {
		t.Errorf("skills list len = %d, want 1", len(itemsList))
	}

	// Verify structure: item should have name and description fields
	if len(itemsList) > 0 {
		skillMap, ok := itemsList[0].(map[string]any)
		if !ok {
			t.Errorf("expected skill to be map, got %T", itemsList[0])
		} else {
			if name, ok := skillMap["name"]; !ok || name == "" {
				t.Errorf("expected 'name' field in skill, got %v", skillMap)
			}
			if name, ok := skillMap["name"].(string); ok && name != "test-skill-e2e" {
				t.Errorf("expected name 'test-skill-e2e', got %q", name)
			}
		}
	}
}

func TestIssuesE2E(t *testing.T) {
	srv, _ := testutil.SpawnTestServer(t)

	// Create a company first
	companyBody, _ := json.Marshal(map[string]string{
		"name":        "Test Corp",
		"shortname":   "test",
		"description": "Test company",
	})
	respCompany, err := http.Post(srv.URL+"/api/companies", "application/json", bytes.NewReader(companyBody))
	if err != nil {
		t.Fatalf("POST /api/companies: %v", err)
	}
	var company map[string]any
	if err := json.NewDecoder(respCompany.Body).Decode(&company); err != nil {
		t.Fatalf("decoding company response: %v", err)
	}
	respCompany.Body.Close()
	companyID, _ := company["id"].(string)

	// Create an agent
	agentBody, _ := json.Marshal(map[string]any{
		"companyId":   companyID,
		"shortname":   "alice",
		"displayName": "Alice",
		"role":        "manager",
		"adapter":     "stub",
	})
	respAgent, err := http.Post(srv.URL+"/api/agents", "application/json", bytes.NewReader(agentBody))
	if err != nil {
		t.Fatalf("POST /api/agents: %v", err)
	}
	var agent map[string]any
	if err := json.NewDecoder(respAgent.Body).Decode(&agent); err != nil {
		t.Fatalf("decoding agent response: %v", err)
	}
	respAgent.Body.Close()
	agentID, _ := agent["id"].(string)

	// POST /api/issues → 201
	issueBody, _ := json.Marshal(map[string]any{
		"companyId": companyID,
		"title":     "Test Issue",
		"body":      "This is a test issue",
	})
	resp, err := http.Post(srv.URL+"/api/issues", "application/json", bytes.NewReader(issueBody))
	if err != nil {
		t.Fatalf("POST /api/issues: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST /api/issues status = %d, want 201", resp.StatusCode)
	}

	var created map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decoding POST response: %v", err)
	}
	issueID, _ := created["id"].(string)
	if issueID == "" {
		t.Fatalf("expected id in POST response, got %v", created)
	}

	// GET /api/issues/{id} → 200
	resp2, err := http.Get(srv.URL + "/api/issues/" + issueID)
	if err != nil {
		t.Fatalf("GET /api/issues/%s: %v", issueID, err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("GET /api/issues/%s status = %d, want 200", issueID, resp2.StatusCode)
	}

	// POST /api/issues/{id}/checkout → 200
	checkoutBody, _ := json.Marshal(map[string]string{"agentId": agentID})
	resp3, err := http.Post(srv.URL+"/api/issues/"+issueID+"/checkout", "application/json", bytes.NewReader(checkoutBody))
	if err != nil {
		t.Fatalf("POST /api/issues/%s/checkout: %v", issueID, err)
	}
	resp3.Body.Close()
	if resp3.StatusCode != http.StatusOK {
		t.Errorf("POST /api/issues/%s/checkout status = %d, want 200", issueID, resp3.StatusCode)
	}

	// POST /api/issues/{id}/checkout again by same agent → 200 (idempotent)
	resp4, err := http.Post(srv.URL+"/api/issues/"+issueID+"/checkout", "application/json", bytes.NewReader(checkoutBody))
	if err != nil {
		t.Fatalf("POST /api/issues/%s/checkout (idempotent): %v", issueID, err)
	}
	resp4.Body.Close()
	if resp4.StatusCode != http.StatusOK {
		t.Errorf("POST /api/issues/%s/checkout (idempotent) status = %d, want 200", issueID, resp4.StatusCode)
	}

	// POST /api/issues/{id}/comments → 201
	commentBody, _ := json.Marshal(map[string]any{
		"body":          "Test comment",
		"authorKind":    "agent",
		"authorAgentId": agentID,
	})
	resp5, err := http.Post(srv.URL+"/api/issues/"+issueID+"/comments", "application/json", bytes.NewReader(commentBody))
	if err != nil {
		t.Fatalf("POST /api/issues/%s/comments: %v", issueID, err)
	}
	resp5.Body.Close()
	if resp5.StatusCode != http.StatusCreated {
		t.Errorf("POST /api/issues/%s/comments status = %d, want 201", issueID, resp5.StatusCode)
	}

	// GET /api/issues/{id}/comments → 200 (contains comment)
	resp6, err := http.Get(srv.URL + "/api/issues/" + issueID + "/comments")
	if err != nil {
		t.Fatalf("GET /api/issues/%s/comments: %v", issueID, err)
	}
	defer resp6.Body.Close()
	if resp6.StatusCode != http.StatusOK {
		t.Errorf("GET /api/issues/%s/comments status = %d, want 200", issueID, resp6.StatusCode)
	}
	var comments map[string]any
	if err := json.NewDecoder(resp6.Body).Decode(&comments); err != nil {
		t.Fatalf("decoding comments: %v", err)
	}
	commentItems, _ := comments["items"].([]any)
	if len(commentItems) != 1 {
		t.Errorf("comments list len = %d, want 1", len(commentItems))
	}

	// POST /api/issues/{id}/release → 200
	releaseBody, _ := json.Marshal(map[string]string{"agentId": agentID})
	resp7, err := http.Post(srv.URL+"/api/issues/"+issueID+"/release", "application/json", bytes.NewReader(releaseBody))
	if err != nil {
		t.Fatalf("POST /api/issues/%s/release: %v", issueID, err)
	}
	resp7.Body.Close()
	if resp7.StatusCode != http.StatusOK {
		t.Errorf("POST /api/issues/%s/release status = %d, want 200", issueID, resp7.StatusCode)
	}

	// POST /api/issues/{id}/checkout again → 200 (succeeds after release)
	resp8, err := http.Post(srv.URL+"/api/issues/"+issueID+"/checkout", "application/json", bytes.NewReader(checkoutBody))
	if err != nil {
		t.Fatalf("POST /api/issues/%s/checkout (after release): %v", issueID, err)
	}
	resp8.Body.Close()
	if resp8.StatusCode != http.StatusOK {
		t.Errorf("POST /api/issues/%s/checkout (after release) status = %d, want 200", issueID, resp8.StatusCode)
	}

	// Test error cases: missing required fields
	badIssueBody, _ := json.Marshal(map[string]string{"companyId": companyID})
	resp9, err := http.Post(srv.URL+"/api/issues", "application/json", bytes.NewReader(badIssueBody))
	if err != nil {
		t.Fatalf("POST /api/issues (bad): %v", err)
	}
	resp9.Body.Close()
	if resp9.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("POST /api/issues (bad) status = %d, want 422", resp9.StatusCode)
	}

	// Test 404 not found
	resp10, err := http.Get(srv.URL + "/api/issues/nonexistent-id")
	if err != nil {
		t.Fatalf("GET /api/issues/nonexistent: %v", err)
	}
	resp10.Body.Close()
	if resp10.StatusCode != http.StatusNotFound {
		t.Errorf("GET /api/issues/nonexistent status = %d, want 404", resp10.StatusCode)
	}
}

func TestSkillsE2E(t *testing.T) {
	// Create test skills
	testSkills := []domain.Skill{
		{
			Name:        "Test Skill",
			Description: "A test skill for E2E testing",
			Path:        "/test/skill/SKILL.md",
			Body:        "This is the test skill body",
		},
	}

	srv, _ := testutil.SpawnTestServerWithSkills(t, testSkills)

	// GET /api/skills → 200 with items
	resp, err := http.Get(srv.URL + "/api/skills/")
	if err != nil {
		t.Fatalf("GET /api/skills/: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/skills/ status = %d, want 200", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	items, ok := result["items"].([]any)
	if !ok {
		t.Fatalf("expected items to be an array, got %T", result["items"])
	}

	if len(items) != 1 {
		t.Errorf("GET /api/skills/ returned %d items, want 1", len(items))
	}

	// Verify the skill data
	if len(items) > 0 {
		skillItem, ok := items[0].(map[string]any)
		if !ok {
			t.Fatalf("expected skill item to be an object, got %T", items[0])
		}

		if skillItem["name"] != "Test Skill" {
			t.Errorf("skill name = %v, want 'Test Skill'", skillItem["name"])
		}
		if skillItem["description"] != "A test skill for E2E testing" {
			t.Errorf("skill description = %v, want 'A test skill for E2E testing'", skillItem["description"])
		}
	}
}
