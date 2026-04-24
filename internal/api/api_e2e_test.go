package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
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
	// Verify default status is "open" when not provided
	if status, _ := created["status"].(string); status != "open" {
		t.Errorf("POST /api/issues (no status) returned status = %q, want open", status)
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

	// POST /api/issues with invalid status → 422
	invalidStatusBody, _ := json.Marshal(map[string]any{
		"companyId": companyID,
		"title":     "Test Issue with Invalid Status",
		"body":      "This issue has an invalid status",
		"status":    "bogus",
	})
	respInvalidStatus, err := http.Post(srv.URL+"/api/issues", "application/json", bytes.NewReader(invalidStatusBody))
	if err != nil {
		t.Fatalf("POST /api/issues (invalid status): %v", err)
	}
	respInvalidStatus.Body.Close()
	if respInvalidStatus.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("POST /api/issues (invalid status) status = %d, want 422", respInvalidStatus.StatusCode)
	}

	// POST /api/issues with explicit valid status "blocked" → 201 with status persisted
	validStatusBody, _ := json.Marshal(map[string]any{
		"companyId": companyID,
		"title":     "Test Issue with Blocked Status",
		"body":      "This issue starts as blocked",
		"status":    "blocked",
	})
	respValidStatus, err := http.Post(srv.URL+"/api/issues", "application/json", bytes.NewReader(validStatusBody))
	if err != nil {
		t.Fatalf("POST /api/issues (valid status): %v", err)
	}
	defer respValidStatus.Body.Close()
	if respValidStatus.StatusCode != http.StatusCreated {
		t.Fatalf("POST /api/issues (valid status) status = %d, want 201", respValidStatus.StatusCode)
	}

	var createdWithStatus map[string]any
	if err := json.NewDecoder(respValidStatus.Body).Decode(&createdWithStatus); err != nil {
		t.Fatalf("decoding POST (valid status) response: %v", err)
	}
	createdStatus, _ := createdWithStatus["status"].(string)
	if createdStatus != "blocked" {
		t.Errorf("POST /api/issues created issue status = %q, want %q", createdStatus, "blocked")
	}

	// Verify the status persisted by fetching the issue
	createdIssueID, _ := createdWithStatus["id"].(string)
	respFetchStatus, err := http.Get(srv.URL + "/api/issues/" + createdIssueID)
	if err != nil {
		t.Fatalf("GET /api/issues/%s: %v", createdIssueID, err)
	}
	defer respFetchStatus.Body.Close()
	if respFetchStatus.StatusCode != http.StatusOK {
		t.Errorf("GET /api/issues/%s status = %d, want 200", createdIssueID, respFetchStatus.StatusCode)
	}

	var fetchedIssue map[string]any
	if err := json.NewDecoder(respFetchStatus.Body).Decode(&fetchedIssue); err != nil {
		t.Fatalf("decoding fetched issue: %v", err)
	}
	fetchedStatus, _ := fetchedIssue["status"].(string)
	if fetchedStatus != "blocked" {
		t.Errorf("fetched issue status = %q, want %q", fetchedStatus, "blocked")
	}

	// Create a new issue for documents/workProducts tests
	docTestBody, _ := json.Marshal(map[string]any{
		"companyId": companyID,
		"title":     "Test Issue for Documents",
		"body":      "Issue for testing documents and workProducts",
	})
	respDocTest, err := http.Post(srv.URL+"/api/issues", "application/json", bytes.NewReader(docTestBody))
	if err != nil {
		t.Fatalf("POST /api/issues (doc test): %v", err)
	}
	defer respDocTest.Body.Close()

	var docTestIssue map[string]any
	if err := json.NewDecoder(respDocTest.Body).Decode(&docTestIssue); err != nil {
		t.Fatalf("decoding doc test issue: %v", err)
	}
	docTestIssueID, _ := docTestIssue["id"].(string)

	// Test PATCH with documents → 200
	docPatchBody, _ := json.Marshal(map[string]any{
		"documents": []map[string]string{
			{"title": "spec", "url": "https://example.com/spec"},
			{"title": "design", "url": "https://example.com/design"},
		},
	})
	req3, _ := http.NewRequest("PATCH", srv.URL+"/api/issues/"+docTestIssueID, bytes.NewReader(docPatchBody))
	req3.Header.Set("Content-Type", "application/json")
	resp15, err := http.DefaultClient.Do(req3)
	if err != nil {
		t.Fatalf("PATCH /api/issues/%s (documents): %v", docTestIssueID, err)
	}
	resp15.Body.Close()
	if resp15.StatusCode != http.StatusOK {
		t.Errorf("PATCH /api/issues/%s (documents) status = %d, want 200", docTestIssueID, resp15.StatusCode)
	}

	// GET /api/issues/{id} to verify documents persisted
	resp16, err := http.Get(srv.URL + "/api/issues/" + docTestIssueID)
	if err != nil {
		t.Fatalf("GET /api/issues/%s (verify documents): %v", docTestIssueID, err)
	}
	defer resp16.Body.Close()

	var fetchedWithDocs map[string]any
	if err := json.NewDecoder(resp16.Body).Decode(&fetchedWithDocs); err != nil {
		t.Fatalf("decoding fetched issue with documents: %v", err)
	}

	docs, ok := fetchedWithDocs["documents"].([]any)
	if !ok || len(docs) != 2 {
		t.Errorf("fetched issue documents = %v, want slice of 2 items", fetchedWithDocs["documents"])
	}

	// Test PATCH with workProducts → 200
	wpPatchBody, _ := json.Marshal(map[string]any{
		"workProducts": []map[string]string{
			{"name": "report", "type": "pdf"},
		},
	})
	req4, _ := http.NewRequest("PATCH", srv.URL+"/api/issues/"+docTestIssueID, bytes.NewReader(wpPatchBody))
	req4.Header.Set("Content-Type", "application/json")
	resp17, err := http.DefaultClient.Do(req4)
	if err != nil {
		t.Fatalf("PATCH /api/issues/%s (workProducts): %v", docTestIssueID, err)
	}
	resp17.Body.Close()
	if resp17.StatusCode != http.StatusOK {
		t.Errorf("PATCH /api/issues/%s (workProducts) status = %d, want 200", docTestIssueID, resp17.StatusCode)
	}

	// GET /api/issues/{id} to verify workProducts persisted
	resp18, err := http.Get(srv.URL + "/api/issues/" + docTestIssueID)
	if err != nil {
		t.Fatalf("GET /api/issues/%s (verify workProducts): %v", docTestIssueID, err)
	}
	defer resp18.Body.Close()

	var fetchedWithWP map[string]any
	if err := json.NewDecoder(resp18.Body).Decode(&fetchedWithWP); err != nil {
		t.Fatalf("decoding fetched issue with workProducts: %v", err)
	}

	wps, ok := fetchedWithWP["workProducts"].([]any)
	if !ok || len(wps) != 1 {
		t.Errorf("fetched issue workProducts = %v, want slice of 1 item", fetchedWithWP["workProducts"])
	}

	// Test PATCH with empty documents to clear → 200
	clearDocsPatchBody, _ := json.Marshal(map[string]any{
		"documents": []any{},
	})
	req5, _ := http.NewRequest("PATCH", srv.URL+"/api/issues/"+docTestIssueID, bytes.NewReader(clearDocsPatchBody))
	req5.Header.Set("Content-Type", "application/json")
	resp19, err := http.DefaultClient.Do(req5)
	if err != nil {
		t.Fatalf("PATCH /api/issues/%s (clear documents): %v", docTestIssueID, err)
	}
	resp19.Body.Close()
	if resp19.StatusCode != http.StatusOK {
		t.Errorf("PATCH /api/issues/%s (clear documents) status = %d, want 200", docTestIssueID, resp19.StatusCode)
	}

	// GET /api/issues/{id} to verify documents cleared
	resp20, err := http.Get(srv.URL + "/api/issues/" + docTestIssueID)
	if err != nil {
		t.Fatalf("GET /api/issues/%s (verify cleared documents): %v", docTestIssueID, err)
	}
	defer resp20.Body.Close()

	var fetchedClearedDocs map[string]any
	if err := json.NewDecoder(resp20.Body).Decode(&fetchedClearedDocs); err != nil {
		t.Fatalf("decoding fetched issue with cleared documents: %v", err)
	}

	clearedDocs, ok := fetchedClearedDocs["documents"].([]any)
	if !ok || len(clearedDocs) != 0 {
		t.Errorf("fetched issue cleared documents = %v, want empty array", fetchedClearedDocs["documents"])
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

func TestAgentLifecycleE2E(t *testing.T) {
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
	var created map[string]any
	if err := json.NewDecoder(respAgent.Body).Decode(&created); err != nil {
		t.Fatalf("decoding POST response: %v", err)
	}
	respAgent.Body.Close()
	agentID, _ := created["id"].(string)

	// Check initial state is idle
	runtimeState, _ := created["runtimeState"].(string)
	if runtimeState != "idle" {
		t.Errorf("initial runtimeState = %q, want %q", runtimeState, "idle")
	}

	// Pause the agent → 200
	req, _ := http.NewRequest("POST", srv.URL+"/api/agents/"+agentID+"/pause", nil)
	respPause, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /api/agents/%s/pause: %v", agentID, err)
	}
	var paused map[string]any
	if err := json.NewDecoder(respPause.Body).Decode(&paused); err != nil {
		t.Fatalf("decoding pause response: %v", err)
	}
	respPause.Body.Close()
	if respPause.StatusCode != http.StatusOK {
		t.Errorf("pause status = %d, want 200", respPause.StatusCode)
	}
	pausedState, _ := paused["runtimeState"].(string)
	if pausedState != "paused" {
		t.Errorf("after pause runtimeState = %q, want %q", pausedState, "paused")
	}

	// Resume the agent → 200
	reqResume, _ := http.NewRequest("POST", srv.URL+"/api/agents/"+agentID+"/resume", nil)
	respResume, err := http.DefaultClient.Do(reqResume)
	if err != nil {
		t.Fatalf("POST /api/agents/%s/resume: %v", agentID, err)
	}
	var resumed map[string]any
	if err := json.NewDecoder(respResume.Body).Decode(&resumed); err != nil {
		t.Fatalf("decoding resume response: %v", err)
	}
	respResume.Body.Close()
	if respResume.StatusCode != http.StatusOK {
		t.Errorf("resume status = %d, want 200", respResume.StatusCode)
	}
	resumedState, _ := resumed["runtimeState"].(string)
	if resumedState != "running" {
		t.Errorf("after resume runtimeState = %q, want %q", resumedState, "running")
	}

	// Terminate the agent → 200
	reqTerminate, _ := http.NewRequest("POST", srv.URL+"/api/agents/"+agentID+"/terminate", nil)
	respTerminate, err := http.DefaultClient.Do(reqTerminate)
	if err != nil {
		t.Fatalf("POST /api/agents/%s/terminate: %v", agentID, err)
	}
	var terminated map[string]any
	if err := json.NewDecoder(respTerminate.Body).Decode(&terminated); err != nil {
		t.Fatalf("decoding terminate response: %v", err)
	}
	respTerminate.Body.Close()
	if respTerminate.StatusCode != http.StatusOK {
		t.Errorf("terminate status = %d, want 200", respTerminate.StatusCode)
	}
	terminatedState, _ := terminated["runtimeState"].(string)
	if terminatedState != "terminated" {
		t.Errorf("after terminate runtimeState = %q, want %q", terminatedState, "terminated")
	}

	// Try to terminate again → 422 (invalid transition)
	reqTerminate2, _ := http.NewRequest("POST", srv.URL+"/api/agents/"+agentID+"/terminate", nil)
	respTerminate2, err := http.DefaultClient.Do(reqTerminate2)
	if err != nil {
		t.Fatalf("POST /api/agents/%s/terminate (2nd): %v", agentID, err)
	}
	respTerminate2.Body.Close()
	if respTerminate2.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("2nd terminate status = %d, want 422", respTerminate2.StatusCode)
	}

	// PATCH to update runtime state via Update handler
	agent3Body, _ := json.Marshal(map[string]any{
		"companyId":   companyID,
		"shortname":   "bob",
		"displayName": "Bob",
		"role":        "engineer",
		"adapter":     "stub",
	})
	respAgent3, err := http.Post(srv.URL+"/api/agents", "application/json", bytes.NewReader(agent3Body))
	if err != nil {
		t.Fatalf("POST /api/agents (agent3): %v", err)
	}
	var created3 map[string]any
	if err := json.NewDecoder(respAgent3.Body).Decode(&created3); err != nil {
		t.Fatalf("decoding agent3 response: %v", err)
	}
	respAgent3.Body.Close()
	agent3ID, _ := created3["id"].(string)

	// PATCH with runtimeState update → 200
	patchBody, _ := json.Marshal(map[string]string{
		"runtimeState": "paused",
	})
	reqPatch, _ := http.NewRequest("PATCH", srv.URL+"/api/agents/"+agent3ID, bytes.NewReader(patchBody))
	reqPatch.Header.Set("Content-Type", "application/json")
	respPatch, err := http.DefaultClient.Do(reqPatch)
	if err != nil {
		t.Fatalf("PATCH /api/agents/%s: %v", agent3ID, err)
	}
	var patched map[string]any
	if err := json.NewDecoder(respPatch.Body).Decode(&patched); err != nil {
		t.Fatalf("decoding patch response: %v", err)
	}
	respPatch.Body.Close()
	if respPatch.StatusCode != http.StatusOK {
		t.Errorf("PATCH status = %d, want 200", respPatch.StatusCode)
	}
	patchedState, _ := patched["runtimeState"].(string)
	if patchedState != "paused" {
		t.Errorf("after PATCH runtimeState = %q, want %q", patchedState, "paused")
	}

	// PATCH with invalid runtimeState → 422
	badPatchBody, _ := json.Marshal(map[string]string{
		"runtimeState": "bogus",
	})
	reqBadPatch, _ := http.NewRequest("PATCH", srv.URL+"/api/agents/"+agent3ID, bytes.NewReader(badPatchBody))
	reqBadPatch.Header.Set("Content-Type", "application/json")
	respBadPatch, err := http.DefaultClient.Do(reqBadPatch)
	if err != nil {
		t.Fatalf("PATCH /api/agents/%s (bad): %v", agent3ID, err)
	}
	respBadPatch.Body.Close()
	if respBadPatch.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("PATCH with bogus state status = %d, want 422", respBadPatch.StatusCode)
	}

	// Verify persistence: GET the agent and check state is still paused
	respFetch, err := http.Get(srv.URL + "/api/agents/" + agent3ID)
	if err != nil {
		t.Fatalf("GET /api/agents/%s: %v", agent3ID, err)
	}
	var fetched map[string]any
	if err := json.NewDecoder(respFetch.Body).Decode(&fetched); err != nil {
		t.Fatalf("decoding fetched response: %v", err)
	}
	respFetch.Body.Close()
	fetchedState, _ := fetched["runtimeState"].(string)
	if fetchedState != "paused" {
		t.Errorf("after fetch runtimeState = %q, want %q", fetchedState, "paused")
	}

	// Test 404 on nonexistent agent for pause/resume/terminate
	nonexistentID := "nonexistent-" + uuid.New().String()

	// Test pause on nonexistent agent
	reqPauseNonexistent, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/agents/%s/pause", srv.URL, nonexistentID), nil)
	respPauseNonexistent, err := http.DefaultClient.Do(reqPauseNonexistent)
	if err != nil {
		t.Fatalf("pause nonexistent: %v", err)
	}
	respPauseNonexistent.Body.Close()
	if respPauseNonexistent.StatusCode != http.StatusNotFound {
		t.Errorf("pause nonexistent: expected 404, got %d", respPauseNonexistent.StatusCode)
	}

	// Test resume on nonexistent agent
	reqResumeNonexistent, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/agents/%s/resume", srv.URL, nonexistentID), nil)
	respResumeNonexistent, err := http.DefaultClient.Do(reqResumeNonexistent)
	if err != nil {
		t.Fatalf("resume nonexistent: %v", err)
	}
	respResumeNonexistent.Body.Close()
	if respResumeNonexistent.StatusCode != http.StatusNotFound {
		t.Errorf("resume nonexistent: expected 404, got %d", respResumeNonexistent.StatusCode)
	}

	// Test terminate on nonexistent agent
	reqTerminateNonexistent, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/agents/%s/terminate", srv.URL, nonexistentID), nil)
	respTerminateNonexistent, err := http.DefaultClient.Do(reqTerminateNonexistent)
	if err != nil {
		t.Fatalf("terminate nonexistent: %v", err)
	}
	respTerminateNonexistent.Body.Close()
	if respTerminateNonexistent.StatusCode != http.StatusNotFound {
		t.Errorf("terminate nonexistent: expected 404, got %d", respTerminateNonexistent.StatusCode)
	}
}

func TestAgentConfigurationE2E(t *testing.T) {
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
	var created map[string]any
	if err := json.NewDecoder(respAgent.Body).Decode(&created); err != nil {
		t.Fatalf("decoding POST response: %v", err)
	}
	respAgent.Body.Close()
	agentID, _ := created["id"].(string)

	// Check initial configuration is empty
	initialConfig, _ := created["configuration"].(map[string]any)
	if initialConfig == nil || len(initialConfig) != 0 {
		t.Errorf("initial configuration = %v, want empty map", initialConfig)
	}

	// PATCH with configuration → 200
	patchBody1, _ := json.Marshal(map[string]any{
		"configuration": map[string]any{
			"model": "claude-opus-4",
		},
	})
	reqPatch1, _ := http.NewRequest("PATCH", srv.URL+"/api/agents/"+agentID, bytes.NewReader(patchBody1))
	reqPatch1.Header.Set("Content-Type", "application/json")
	respPatch1, err := http.DefaultClient.Do(reqPatch1)
	if err != nil {
		t.Fatalf("PATCH /api/agents/%s: %v", agentID, err)
	}
	var patched1 map[string]any
	if err := json.NewDecoder(respPatch1.Body).Decode(&patched1); err != nil {
		t.Fatalf("decoding patch response: %v", err)
	}
	respPatch1.Body.Close()
	if respPatch1.StatusCode != http.StatusOK {
		t.Errorf("PATCH status = %d, want 200", respPatch1.StatusCode)
	}

	patchedConfig1, _ := patched1["configuration"].(map[string]any)
	if patchedConfig1["model"] != "claude-opus-4" {
		t.Errorf("configuration[model] = %v, want %q", patchedConfig1["model"], "claude-opus-4")
	}

	// GET to verify persistence
	respGet1, err := http.Get(srv.URL + "/api/agents/" + agentID)
	if err != nil {
		t.Fatalf("GET /api/agents/%s: %v", agentID, err)
	}
	var fetched1 map[string]any
	if err := json.NewDecoder(respGet1.Body).Decode(&fetched1); err != nil {
		t.Fatalf("decoding GET response: %v", err)
	}
	respGet1.Body.Close()

	fetchedConfig1, _ := fetched1["configuration"].(map[string]any)
	if fetchedConfig1["model"] != "claude-opus-4" {
		t.Errorf("fetched configuration[model] = %v, want %q", fetchedConfig1["model"], "claude-opus-4")
	}

	// PATCH with merge: add temperature, preserve model
	patchBody2, _ := json.Marshal(map[string]any{
		"configuration": map[string]any{
			"temperature": 0.7,
		},
	})
	reqPatch2, _ := http.NewRequest("PATCH", srv.URL+"/api/agents/"+agentID, bytes.NewReader(patchBody2))
	reqPatch2.Header.Set("Content-Type", "application/json")
	respPatch2, err := http.DefaultClient.Do(reqPatch2)
	if err != nil {
		t.Fatalf("PATCH merge /api/agents/%s: %v", agentID, err)
	}
	var patched2 map[string]any
	if err := json.NewDecoder(respPatch2.Body).Decode(&patched2); err != nil {
		t.Fatalf("decoding merge patch response: %v", err)
	}
	respPatch2.Body.Close()

	patchedConfig2, _ := patched2["configuration"].(map[string]any)
	if patchedConfig2["model"] != "claude-opus-4" {
		t.Errorf("merged configuration[model] = %v, want %q (should be preserved)", patchedConfig2["model"], "claude-opus-4")
	}
	if patchedConfig2["temperature"] != float64(0.7) {
		t.Errorf("merged configuration[temperature] = %v, want 0.7", patchedConfig2["temperature"])
	}

	// PATCH with empty body → 422
	emptyPatchBody, _ := json.Marshal(map[string]any{})
	reqEmptyPatch, _ := http.NewRequest("PATCH", srv.URL+"/api/agents/"+agentID, bytes.NewReader(emptyPatchBody))
	reqEmptyPatch.Header.Set("Content-Type", "application/json")
	respEmptyPatch, err := http.DefaultClient.Do(reqEmptyPatch)
	if err != nil {
		t.Fatalf("PATCH empty /api/agents/%s: %v", agentID, err)
	}
	respEmptyPatch.Body.Close()
	if respEmptyPatch.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("empty PATCH status = %d, want 422", respEmptyPatch.StatusCode)
	}

	// PATCH nonexistent agent → 404
	nonexistentID := "nonexistent-" + uuid.New().String()
	reqNonexistent, _ := http.NewRequest("PATCH", srv.URL+"/api/agents/"+nonexistentID, bytes.NewReader(patchBody1))
	reqNonexistent.Header.Set("Content-Type", "application/json")
	respNonexistent, err := http.DefaultClient.Do(reqNonexistent)
	if err != nil {
		t.Fatalf("PATCH nonexistent /api/agents/%s: %v", nonexistentID, err)
	}
	respNonexistent.Body.Close()
	if respNonexistent.StatusCode != http.StatusNotFound {
		t.Errorf("PATCH nonexistent status = %d, want 404", respNonexistent.StatusCode)
	}
}

func TestLabelsE2E(t *testing.T) {
	srv, _ := testutil.SpawnTestServer(t)

	// 1. Create company
	companyBody, _ := json.Marshal(map[string]string{
		"name":        "Test Corp",
		"shortname":   "test",
		"description": "Test company",
	})
	respCompany, err := http.Post(srv.URL+"/api/companies", "application/json", bytes.NewReader(companyBody))
	if err != nil {
		t.Fatalf("POST /api/companies: %v", err)
	}
	defer respCompany.Body.Close()
	if respCompany.StatusCode < http.StatusOK || respCompany.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(respCompany.Body)
		t.Fatalf("POST /api/companies returned %d: %s", respCompany.StatusCode, string(body))
	}
	var company map[string]any
	if err := json.NewDecoder(respCompany.Body).Decode(&company); err != nil {
		t.Fatalf("decoding company response: %v", err)
	}
	companyID, _ := company["id"].(string)

	// 2. Create label (bug, #ff0000)
	labelBody, _ := json.Marshal(map[string]string{
		"companyId": companyID,
		"name":      "bug",
		"color":     "#ff0000",
	})
	respLabel, err := http.Post(srv.URL+"/api/labels", "application/json", bytes.NewReader(labelBody))
	if err != nil {
		t.Fatalf("POST /api/labels: %v", err)
	}
	defer respLabel.Body.Close()
	labelRespBody, err := io.ReadAll(respLabel.Body)
	if err != nil {
		t.Fatalf("reading label response: %v", err)
	}
	if respLabel.StatusCode != http.StatusCreated {
		t.Fatalf("POST /api/labels status = %d, want 201; body = %s", respLabel.StatusCode, string(labelRespBody))
	}
	var label map[string]any
	if err := json.Unmarshal(labelRespBody, &label); err != nil {
		t.Fatalf("decoding label response: %v", err)
	}
	labelID, _ := label["id"].(string)

	// 3. List labels → 1 item
	respListLabels, err := http.Get(srv.URL + "/api/labels?companyId=" + companyID)
	if err != nil {
		t.Fatalf("GET /api/labels: %v", err)
	}
	defer respListLabels.Body.Close()
	if respListLabels.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/labels status = %d, want 200", respListLabels.StatusCode)
	}
	var labelsList map[string]any
	if err := json.NewDecoder(respListLabels.Body).Decode(&labelsList); err != nil {
		t.Fatalf("decoding labels list: %v", err)
	}
	labelItems, _ := labelsList["items"].([]any)
	if len(labelItems) != 1 {
		t.Errorf("labels list len = %d, want 1", len(labelItems))
	}

	// 4. Create duplicate label → 409
	dupLabelBody, _ := json.Marshal(map[string]string{
		"companyId": companyID,
		"name":      "bug",
		"color":     "#00ff00",
	})
	respDupLabel, err := http.Post(srv.URL+"/api/labels", "application/json", bytes.NewReader(dupLabelBody))
	if err != nil {
		t.Fatalf("POST duplicate label: %v", err)
	}
	respDupLabel.Body.Close()
	if respDupLabel.StatusCode != http.StatusConflict {
		t.Errorf("POST duplicate label status = %d, want 409", respDupLabel.StatusCode)
	}

	// 5. Create issue
	issueBody, _ := json.Marshal(map[string]any{
		"companyId": companyID,
		"title":     "Test Issue",
		"body":      "This is a test issue",
	})
	respIssue, err := http.Post(srv.URL+"/api/issues", "application/json", bytes.NewReader(issueBody))
	if err != nil {
		t.Fatalf("POST /api/issues: %v", err)
	}
	defer respIssue.Body.Close()
	if respIssue.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(respIssue.Body)
		t.Fatalf("POST /api/issues status = %d, want 201; body = %s", respIssue.StatusCode, string(body))
	}
	var issue map[string]any
	if err := json.NewDecoder(respIssue.Body).Decode(&issue); err != nil {
		t.Fatalf("decoding issue response: %v", err)
	}
	issueID, _ := issue["id"].(string)

	// 6. Get issue → labels:[] (empty)
	respGetIssue, err := http.Get(srv.URL + "/api/issues/" + issueID)
	if err != nil {
		t.Fatalf("GET /api/issues/%s: %v", issueID, err)
	}
	defer respGetIssue.Body.Close()
	if respGetIssue.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(respGetIssue.Body)
		t.Fatalf("GET /api/issues/%s status = %d, want 200; body = %s", issueID, respGetIssue.StatusCode, string(body))
	}
	var issueGet map[string]any
	if err := json.NewDecoder(respGetIssue.Body).Decode(&issueGet); err != nil {
		t.Fatalf("decoding issue response: %v", err)
	}
	labels, _ := issueGet["labels"].([]any)
	if len(labels) != 0 {
		t.Errorf("initial labels len = %d, want 0", len(labels))
	}

	// 7. Add label to issue → 200
	addLabelBody, _ := json.Marshal(map[string]string{
		"labelId": labelID,
	})
	respAddLabel, err := http.Post(srv.URL+"/api/issues/"+issueID+"/labels", "application/json", bytes.NewReader(addLabelBody))
	if err != nil {
		t.Fatalf("POST /api/issues/%s/labels: %v", issueID, err)
	}
	respAddLabel.Body.Close()
	if respAddLabel.StatusCode != http.StatusOK {
		t.Errorf("POST add label status = %d, want 200", respAddLabel.StatusCode)
	}

	// 8. Get issue → labels has 1 item
	respGetIssue2, err := http.Get(srv.URL + "/api/issues/" + issueID)
	if err != nil {
		t.Fatalf("GET /api/issues/%s after add: %v", issueID, err)
	}
	defer respGetIssue2.Body.Close()
	if respGetIssue2.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(respGetIssue2.Body)
		t.Fatalf("GET /api/issues/%s after add status = %d, want 200; body = %s", issueID, respGetIssue2.StatusCode, string(body))
	}
	var issueGet2 map[string]any
	if err := json.NewDecoder(respGetIssue2.Body).Decode(&issueGet2); err != nil {
		t.Fatalf("decoding issue response: %v", err)
	}
	labels2, _ := issueGet2["labels"].([]any)
	if len(labels2) != 1 {
		t.Errorf("labels after add len = %d, want 1", len(labels2))
	}

	// 9. Add same label again → 200 (idempotent)
	addLabelBody2, _ := json.Marshal(map[string]string{
		"labelId": labelID,
	})
	respAddLabelAgain, err := http.Post(srv.URL+"/api/issues/"+issueID+"/labels", "application/json", bytes.NewReader(addLabelBody2))
	if err != nil {
		t.Fatalf("POST add same label again: %v", err)
	}
	respAddLabelAgain.Body.Close()
	if respAddLabelAgain.StatusCode != http.StatusOK {
		t.Errorf("POST add same label again status = %d, want 200", respAddLabelAgain.StatusCode)
	}

	// 10. Get issue → still 1 label (no duplicate)
	respGetIssue3, err := http.Get(srv.URL + "/api/issues/" + issueID)
	if err != nil {
		t.Fatalf("GET /api/issues/%s after add again: %v", issueID, err)
	}
	defer respGetIssue3.Body.Close()
	if respGetIssue3.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(respGetIssue3.Body)
		t.Fatalf("GET /api/issues/%s after add again status = %d, want 200; body = %s", issueID, respGetIssue3.StatusCode, string(body))
	}
	var issueGet3 map[string]any
	if err := json.NewDecoder(respGetIssue3.Body).Decode(&issueGet3); err != nil {
		t.Fatalf("decoding issue response: %v", err)
	}
	labels3, _ := issueGet3["labels"].([]any)
	if len(labels3) != 1 {
		t.Errorf("labels after add again len = %d, want 1 (no duplicate)", len(labels3))
	}

	// 11. Remove label → 204
	reqRemoveLabel, _ := http.NewRequest("DELETE", srv.URL+"/api/issues/"+issueID+"/labels/"+labelID, nil)
	respRemoveLabel, err := http.DefaultClient.Do(reqRemoveLabel)
	if err != nil {
		t.Fatalf("DELETE label from issue: %v", err)
	}
	respRemoveLabel.Body.Close()
	if respRemoveLabel.StatusCode != http.StatusNoContent {
		t.Errorf("DELETE remove label status = %d, want 204", respRemoveLabel.StatusCode)
	}

	// 12. Get issue → labels:[] again
	respGetIssue4, err := http.Get(srv.URL + "/api/issues/" + issueID)
	if err != nil {
		t.Fatalf("GET /api/issues/%s after remove: %v", issueID, err)
	}
	defer respGetIssue4.Body.Close()
	if respGetIssue4.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(respGetIssue4.Body)
		t.Fatalf("GET /api/issues/%s after remove status = %d, want 200; body = %s", issueID, respGetIssue4.StatusCode, string(body))
	}
	var issueGet4 map[string]any
	if err := json.NewDecoder(respGetIssue4.Body).Decode(&issueGet4); err != nil {
		t.Fatalf("decoding issue response: %v", err)
	}
	labels4, _ := issueGet4["labels"].([]any)
	if len(labels4) != 0 {
		t.Errorf("labels after remove len = %d, want 0", len(labels4))
	}

	// 13. Delete label → 204
	reqDeleteLabel, _ := http.NewRequest("DELETE", srv.URL+"/api/labels/"+labelID, nil)
	respDeleteLabel, err := http.DefaultClient.Do(reqDeleteLabel)
	if err != nil {
		t.Fatalf("DELETE label: %v", err)
	}
	respDeleteLabel.Body.Close()
	if respDeleteLabel.StatusCode != http.StatusNoContent {
		t.Errorf("DELETE label status = %d, want 204", respDeleteLabel.StatusCode)
	}

	// 14. List labels → empty
	respListLabels2, err := http.Get(srv.URL + "/api/labels?companyId=" + companyID)
	if err != nil {
		t.Fatalf("GET /api/labels after delete: %v", err)
	}
	defer respListLabels2.Body.Close()
	var labelsList2 map[string]any
	if err := json.NewDecoder(respListLabels2.Body).Decode(&labelsList2); err != nil {
		t.Fatalf("decoding labels list: %v", err)
	}
	labelItems2, _ := labelsList2["items"].([]any)
	if len(labelItems2) != 0 {
		t.Errorf("labels list after delete len = %d, want 0", len(labelItems2))
	}

	// 15. POST label to issue with nonexistent labelId → 404
	badLabelAddBody, _ := json.Marshal(map[string]string{
		"labelId": "nonexistent-label-id",
	})
	respBadLabelAdd, err := http.Post(srv.URL+"/api/issues/"+issueID+"/labels", "application/json", bytes.NewReader(badLabelAddBody))
	if err != nil {
		t.Fatalf("POST with nonexistent label: %v", err)
	}
	respBadLabelAdd.Body.Close()
	if respBadLabelAdd.StatusCode != http.StatusNotFound {
		t.Errorf("POST with nonexistent label status = %d, want 404", respBadLabelAdd.StatusCode)
	}

	// 16. POST label with missing fields → 422
	missingFieldsBody, _ := json.Marshal(map[string]string{
		"companyId": companyID,
		"name":      "feature",
		// missing color
	})
	respMissingFields, err := http.Post(srv.URL+"/api/labels", "application/json", bytes.NewReader(missingFieldsBody))
	if err != nil {
		t.Fatalf("POST with missing fields: %v", err)
	}
	respMissingFields.Body.Close()
	if respMissingFields.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("POST with missing fields status = %d, want 422", respMissingFields.StatusCode)
	}

	// 17. GET labels without companyId → 400
	respBadListLabels, err := http.Get(srv.URL + "/api/labels")
	if err != nil {
		t.Fatalf("GET /api/labels without companyId: %v", err)
	}
	respBadListLabels.Body.Close()
	if respBadListLabels.StatusCode != http.StatusBadRequest {
		t.Errorf("GET /api/labels without companyId status = %d, want 400", respBadListLabels.StatusCode)
	}
}
