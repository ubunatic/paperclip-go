package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

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

	// DELETE /api/companies/{id} with agents → 409
	agentBody, _ := json.Marshal(map[string]any{
		"companyId":   id,
		"shortname":   "alice",
		"displayName": "Alice",
		"role":        "manager",
		"adapter":     "stub",
	})
	respAgent, err := http.Post(srv.URL+"/api/agents", "application/json", bytes.NewReader(agentBody))
	if err != nil {
		t.Fatalf("POST /api/agents: %v", err)
	}
	respAgent.Body.Close()

	req, _ := http.NewRequest("DELETE", srv.URL+"/api/companies/"+id, nil)
	resp6, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE /api/companies with agents: %v", err)
	}
	resp6.Body.Close()
	if resp6.StatusCode != http.StatusConflict {
		t.Errorf("DELETE with agents status = %d, want 409", resp6.StatusCode)
	}

	// Create a separate company with no dependents
	body2, _ := json.Marshal(map[string]string{
		"name":        "Empty Corp",
		"shortname":   "empty",
		"description": "Empty company",
	})
	resp7, err := http.Post(srv.URL+"/api/companies", "application/json", bytes.NewReader(body2))
	if err != nil {
		t.Fatalf("POST /api/companies (empty): %v", err)
	}
	var emptyCompany map[string]any
	if err := json.NewDecoder(resp7.Body).Decode(&emptyCompany); err != nil {
		t.Fatalf("decoding empty company: %v", err)
	}
	resp7.Body.Close()
	emptyID, _ := emptyCompany["id"].(string)

	// DELETE /api/companies/{id} (empty) → 204
	req2, _ := http.NewRequest("DELETE", srv.URL+"/api/companies/"+emptyID, nil)
	resp8, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("DELETE /api/companies (empty): %v", err)
	}
	resp8.Body.Close()
	if resp8.StatusCode != http.StatusNoContent {
		t.Errorf("DELETE empty company status = %d, want 204", resp8.StatusCode)
	}

	// Verify it's gone
	resp9, err := http.Get(srv.URL + "/api/companies/" + emptyID)
	if err != nil {
		t.Fatalf("GET after delete: %v", err)
	}
	resp9.Body.Close()
	if resp9.StatusCode != http.StatusNotFound {
		t.Errorf("GET deleted company status = %d, want 404", resp9.StatusCode)
	}

	// PATCH /api/companies/{id} → 200
	patchBody, _ := json.Marshal(map[string]string{
		"name": "Acme Corp Updated",
	})
	reqPatch, _ := http.NewRequest("PATCH", srv.URL+"/api/companies/"+id, bytes.NewReader(patchBody))
	reqPatch.Header.Set("Content-Type", "application/json")
	resp10, err := http.DefaultClient.Do(reqPatch)
	if err != nil {
		t.Fatalf("PATCH /api/companies/%s: %v", id, err)
	}
	defer resp10.Body.Close()
	if resp10.StatusCode != http.StatusOK {
		t.Errorf("PATCH /api/companies/%s status = %d, want 200", id, resp10.StatusCode)
	}
	var updated map[string]any
	if err := json.NewDecoder(resp10.Body).Decode(&updated); err != nil {
		t.Fatalf("decoding PATCH response: %v", err)
	}
	updatedName, _ := updated["name"].(string)
	if updatedName != "Acme Corp Updated" {
		t.Errorf("PATCH company name = %q, want 'Acme Corp Updated'", updatedName)
	}

	// PATCH /api/companies/{id} with empty name → 422
	badPatchBody, _ := json.Marshal(map[string]string{
		"name": "",
	})
	reqPatchBad, _ := http.NewRequest("PATCH", srv.URL+"/api/companies/"+id, bytes.NewReader(badPatchBody))
	reqPatchBad.Header.Set("Content-Type", "application/json")
	resp11, err := http.DefaultClient.Do(reqPatchBad)
	if err != nil {
		t.Fatalf("PATCH /api/companies/%s (empty name): %v", id, err)
	}
	resp11.Body.Close()
	if resp11.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("PATCH with empty name status = %d, want 422", resp11.StatusCode)
	}

	// PATCH /api/companies/nonexistent → 404
	reqPatchNotFound, _ := http.NewRequest("PATCH", srv.URL+"/api/companies/nonexistent-id", bytes.NewReader(patchBody))
	reqPatchNotFound.Header.Set("Content-Type", "application/json")
	resp12, err := http.DefaultClient.Do(reqPatchNotFound)
	if err != nil {
		t.Fatalf("PATCH /api/companies/nonexistent-id: %v", err)
	}
	resp12.Body.Close()
	if resp12.StatusCode != http.StatusNotFound {
		t.Errorf("PATCH nonexistent company status = %d, want 404", resp12.StatusCode)
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

	// DELETE /api/agents/{id} (no checkouts) → 204
	req2, _ := http.NewRequest("DELETE", srv.URL+"/api/agents/"+agentID, nil)
	resp6, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("DELETE /api/agents: %v", err)
	}
	resp6.Body.Close()
	if resp6.StatusCode != http.StatusNoContent {
		t.Errorf("DELETE agent status = %d, want 204", resp6.StatusCode)
	}

	// Verify it's gone
	resp7, err := http.Get(srv.URL + "/api/agents/" + agentID)
	if err != nil {
		t.Fatalf("GET after delete: %v", err)
	}
	resp7.Body.Close()
	if resp7.StatusCode != http.StatusNotFound {
		t.Errorf("GET deleted agent status = %d, want 404", resp7.StatusCode)
	}

	// DELETE /api/agents/{id} with active checkout → 409
	agent2Body, _ := json.Marshal(map[string]any{
		"companyId":   companyID,
		"shortname":   "bob",
		"displayName": "Bob",
		"role":        "engineer",
		"adapter":     "stub",
	})
	respAgent2, err := http.Post(srv.URL+"/api/agents", "application/json", bytes.NewReader(agent2Body))
	if err != nil {
		t.Fatalf("POST /api/agents (agent2): %v", err)
	}
	var agent2 map[string]any
	if err := json.NewDecoder(respAgent2.Body).Decode(&agent2); err != nil {
		t.Fatalf("decoding agent2 response: %v", err)
	}
	respAgent2.Body.Close()
	agent2ID, _ := agent2["id"].(string)

	// Create an issue
	issueBody, _ := json.Marshal(map[string]any{
		"companyId": companyID,
		"title":     "Test Issue",
		"body":      "This is a test issue",
	})
	respIssue, err := http.Post(srv.URL+"/api/issues", "application/json", bytes.NewReader(issueBody))
	if err != nil {
		t.Fatalf("POST /api/issues: %v", err)
	}
	var issue map[string]any
	if err := json.NewDecoder(respIssue.Body).Decode(&issue); err != nil {
		t.Fatalf("decoding issue response: %v", err)
	}
	respIssue.Body.Close()
	issueID, _ := issue["id"].(string)

	// Checkout the issue to agent2 (sets status='in_progress' and checked_out_by)
	checkoutBody, _ := json.Marshal(map[string]string{"agentId": agent2ID})
	respCheckout, err := http.Post(srv.URL+"/api/issues/"+issueID+"/checkout", "application/json", bytes.NewReader(checkoutBody))
	if err != nil {
		t.Fatalf("POST /api/issues/%s/checkout: %v", issueID, err)
	}
	respCheckout.Body.Close()
	if respCheckout.StatusCode != http.StatusOK {
		t.Errorf("POST /api/issues/%s/checkout status = %d, want 200", issueID, respCheckout.StatusCode)
	}

	// Try to delete agent2 with active checkout → should return 409
	req3, _ := http.NewRequest("DELETE", srv.URL+"/api/agents/"+agent2ID, nil)
	resp8, err := http.DefaultClient.Do(req3)
	if err != nil {
		t.Fatalf("DELETE /api/agents with checkout: %v", err)
	}
	resp8.Body.Close()
	if resp8.StatusCode != http.StatusConflict {
		t.Errorf("DELETE agent with checkout status = %d, want 409", resp8.StatusCode)
	}

	// Verify agent2 still exists
	resp9, err := http.Get(srv.URL + "/api/agents/" + agent2ID)
	if err != nil {
		t.Fatalf("GET agent2 after failed delete: %v", err)
	}
	resp9.Body.Close()
	if resp9.StatusCode != http.StatusOK {
		t.Errorf("GET agent2 after failed delete status = %d, want 200", resp9.StatusCode)
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

	// DELETE /api/issues/{id} while checked out → 409
	// Issue is still checked out by agent, try to delete
	req, _ := http.NewRequest("DELETE", srv.URL+"/api/issues/"+issueID, nil)
	resp11, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE /api/issues while checked out: %v", err)
	}
	resp11.Body.Close()
	if resp11.StatusCode != http.StatusConflict {
		t.Errorf("DELETE while checked out status = %d, want 409", resp11.StatusCode)
	}

	// Release the issue first
	releaseBody2, _ := json.Marshal(map[string]string{"agentId": agentID})
	resp12, err := http.Post(srv.URL+"/api/issues/"+issueID+"/release", "application/json", bytes.NewReader(releaseBody2))
	if err != nil {
		t.Fatalf("POST /api/issues/%s/release: %v", issueID, err)
	}
	resp12.Body.Close()

	// DELETE /api/issues/{id} after release → 204
	req2, _ := http.NewRequest("DELETE", srv.URL+"/api/issues/"+issueID, nil)
	resp13, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("DELETE /api/issues after release: %v", err)
	}
	resp13.Body.Close()
	if resp13.StatusCode != http.StatusNoContent {
		t.Errorf("DELETE after release status = %d, want 204", resp13.StatusCode)
	}

	// Verify issue is gone
	resp14, err := http.Get(srv.URL + "/api/issues/" + issueID)
	if err != nil {
		t.Fatalf("GET after delete: %v", err)
	}
	resp14.Body.Close()
	if resp14.StatusCode != http.StatusNotFound {
		t.Errorf("GET deleted issue status = %d, want 404", resp14.StatusCode)
	}
}

func TestStubEndpointsE2E(t *testing.T) {
	srv, _ := testutil.SpawnTestServer(t)

	// Test each stub endpoint
	endpoints := []string{
		"/api/approvals",
		"/api/costs",
		"/api/goals",
		"/api/projects",
		"/api/routines",
		"/api/plugins",
	}

	for _, endpoint := range endpoints {
		resp, err := http.Get(srv.URL + endpoint)
		if err != nil {
			t.Fatalf("GET %s: %v", endpoint, err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET %s status = %d, want 200", endpoint, resp.StatusCode)
		}

		var result map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decoding %s response: %v", endpoint, err)
		}

		items, ok := result["items"].([]any)
		if !ok {
			t.Errorf("GET %s: expected 'items' array, got %T", endpoint, result["items"])
		}
		if len(items) != 0 {
			t.Errorf("GET %s: expected empty items array, got %d items", endpoint, len(items))
		}
		resp.Body.Close()
	}
}

func TestUIServingE2E(t *testing.T) {
	srv, _ := testutil.SpawnTestServer(t)

	// GET / → 200 with landing page HTML
	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET / status = %d, want 200", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("GET / Content-Type = %s, want 'text/html; charset=utf-8'", contentType)
	}

	var bodyStr strings.Builder
	io.Copy(&bodyStr, resp.Body)
	if !strings.Contains(bodyStr.String(), "paperclip-go") {
		t.Errorf("GET / response doesn't contain 'paperclip-go'")
	}

	// GET /dashboard (non-existent route) → 200 (SPA fallback)
	resp2, err := http.Get(srv.URL + "/dashboard")
	if err != nil {
		t.Fatalf("GET /dashboard: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("GET /dashboard status = %d, want 200", resp2.StatusCode)
	}

	// GET /api/health → 200 (not intercepted by UI handler)
	resp3, err := http.Get(srv.URL + "/api/health")
	if err != nil {
		t.Fatalf("GET /api/health: %v", err)
	}
	defer resp3.Body.Close()

	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/health status = %d, want 200", resp3.StatusCode)
	}

	var health map[string]any
	if err := json.NewDecoder(resp3.Body).Decode(&health); err != nil {
		t.Fatalf("decoding /api/health response: %v", err)
	}

	if health["status"] != "ok" {
		t.Errorf("GET /api/health status field = %v, want 'ok'", health["status"])
	}
}

func TestHeartbeatE2E(t *testing.T) {
	srv, _ := testutil.SpawnTestServer(t)

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

	// Create an issue for the heartbeat to work on
	issueBody, _ := json.Marshal(map[string]any{
		"companyId": companyID,
		"title":     "Test Issue for Heartbeat",
		"body":      "This issue will be worked on by heartbeat",
	})
	respIssue, err := http.Post(srv.URL+"/api/issues", "application/json", bytes.NewReader(issueBody))
	if err != nil {
		t.Fatalf("POST /api/issues: %v", err)
	}
	var issue map[string]any
	if err := json.NewDecoder(respIssue.Body).Decode(&issue); err != nil {
		t.Fatalf("decoding issue response: %v", err)
	}
	respIssue.Body.Close()
	issueID, _ := issue["id"].(string)

	// POST /api/heartbeat/runs with agentId → 201
	runBody, _ := json.Marshal(map[string]string{
		"agentId": agentID,
	})
	resp, err := http.Post(srv.URL+"/api/heartbeat/runs", "application/json", bytes.NewReader(runBody))
	if err != nil {
		t.Fatalf("POST /api/heartbeat/runs: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST /api/heartbeat/runs status = %d, want 201", resp.StatusCode)
	}

	var created map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decoding POST response: %v", err)
	}
	runID, _ := created["id"].(string)
	if runID == "" {
		t.Fatalf("expected id in POST response, got %v", created)
	}
	status, _ := created["status"].(string)
	if status != "success" {
		t.Errorf("expected status=success, got %q", status)
	}

	// Verify the full loop: heartbeat run creates a comment on the issue
	// GET /api/issues/{id}/comments to verify stub adapter posted a comment
	respComments, err := http.Get(srv.URL + "/api/issues/" + issueID + "/comments")
	if err != nil {
		t.Fatalf("GET /api/issues/%s/comments after heartbeat: %v", issueID, err)
	}
	defer respComments.Body.Close()
	if respComments.StatusCode != http.StatusOK {
		t.Errorf("GET /api/issues/%s/comments after heartbeat status = %d, want 200", issueID, respComments.StatusCode)
	}

	var commentsResp map[string]any
	if err := json.NewDecoder(respComments.Body).Decode(&commentsResp); err != nil {
		t.Fatalf("decoding comments response: %v", err)
	}
	commentItems, _ := commentsResp["items"].([]any)
	if len(commentItems) == 0 {
		t.Errorf("expected at least one comment from stub adapter, got %d comments", len(commentItems))
	} else {
		// Verify the comment body contains the stub adapter's message
		commentMap, ok := commentItems[0].(map[string]any)
		if !ok {
			t.Errorf("expected comment to be map, got %T", commentItems[0])
		} else {
			body, _ := commentMap["body"].(string)
			if body == "" {
				t.Errorf("expected comment body to be non-empty, got %q", body)
			}
		}
	}

	// POST /api/heartbeat/runs without agentId → 422
	badBody, _ := json.Marshal(map[string]string{})
	resp2, err := http.Post(srv.URL+"/api/heartbeat/runs", "application/json", bytes.NewReader(badBody))
	if err != nil {
		t.Fatalf("POST /api/heartbeat/runs (bad): %v", err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("POST /api/heartbeat/runs (bad) status = %d, want 422", resp2.StatusCode)
	}

	// GET /api/heartbeat/runs?agentId=... → 200 with list
	resp3, err := http.Get(srv.URL + "/api/heartbeat/runs?agentId=" + agentID)
	if err != nil {
		t.Fatalf("GET /api/heartbeat/runs: %v", err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/heartbeat/runs status = %d, want 200", resp3.StatusCode)
	}

	var list map[string]any
	if err := json.NewDecoder(resp3.Body).Decode(&list); err != nil {
		t.Fatalf("decoding list response: %v", err)
	}
	items, _ := list["items"].([]any)
	if len(items) != 1 {
		t.Errorf("list items len = %d, want 1", len(items))
	}

	// GET /api/heartbeat/runs without agentId → 400
	resp4, err := http.Get(srv.URL + "/api/heartbeat/runs")
	if err != nil {
		t.Fatalf("GET /api/heartbeat/runs (no agentId): %v", err)
	}
	resp4.Body.Close()
	if resp4.StatusCode != http.StatusBadRequest {
		t.Errorf("GET /api/heartbeat/runs (no agentId) status = %d, want 400", resp4.StatusCode)
	}

	// POST /api/heartbeat/runs with non-existent agent → 404
	notFoundBody, _ := json.Marshal(map[string]string{
		"agentId": "nonexistent-agent-id",
	})
	resp5, err := http.Post(srv.URL+"/api/heartbeat/runs", "application/json", bytes.NewReader(notFoundBody))
	if err != nil {
		t.Fatalf("POST /api/heartbeat/runs (not found): %v", err)
	}
	resp5.Body.Close()
	if resp5.StatusCode != http.StatusNotFound {
		t.Errorf("POST /api/heartbeat/runs (not found) status = %d, want 404", resp5.StatusCode)
	}
}
