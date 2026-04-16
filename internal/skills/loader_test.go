package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_HappyPath(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Create first skill
	skill1Dir := filepath.Join(tmpDir, "skill1")
	if err := os.Mkdir(skill1Dir, 0o755); err != nil {
		t.Fatalf("creating skill1 dir: %v", err)
	}
	skill1Content := `---
name: Test Skill 1
description: This is the first test skill
---
This is the body of skill 1.
It has multiple lines.`
	if err := os.WriteFile(filepath.Join(skill1Dir, "SKILL.md"), []byte(skill1Content), 0o644); err != nil {
		t.Fatalf("writing skill1: %v", err)
	}

	// Create second skill
	skill2Dir := filepath.Join(tmpDir, "skill2")
	if err := os.Mkdir(skill2Dir, 0o755); err != nil {
		t.Fatalf("creating skill2 dir: %v", err)
	}
	skill2Content := `---
name: Test Skill 2
description: This is the second test skill
---
This is the body of skill 2.`
	if err := os.WriteFile(filepath.Join(skill2Dir, "SKILL.md"), []byte(skill2Content), 0o644); err != nil {
		t.Fatalf("writing skill2: %v", err)
	}

	// Load skills
	skills, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Verify we got 2 skills
	if len(skills) != 2 {
		t.Errorf("Load returned %d skills, want 2", len(skills))
	}

	// Verify skill 1
	found1 := false
	for _, s := range skills {
		if s.Name == "Test Skill 1" {
			found1 = true
			if s.Description != "This is the first test skill" {
				t.Errorf("skill1 description = %q, want %q", s.Description, "This is the first test skill")
			}
			if !strings.Contains(s.Body, "body of skill 1") {
				t.Errorf("skill1 body doesn't contain expected content")
			}
		}
	}
	if !found1 {
		t.Errorf("skill1 not found in results")
	}

	// Verify skill 2
	found2 := false
	for _, s := range skills {
		if s.Name == "Test Skill 2" {
			found2 = true
			if s.Description != "This is the second test skill" {
				t.Errorf("skill2 description = %q, want %q", s.Description, "This is the second test skill")
			}
			if !strings.Contains(s.Body, "body of skill 2") {
				t.Errorf("skill2 body doesn't contain expected content")
			}
		}
	}
	if !found2 {
		t.Errorf("skill2 not found in results")
	}
}

func TestLoad_MissingDir(t *testing.T) {
	// Load from non-existent directory
	skills, err := Load(filepath.Join(t.TempDir(), "nonexistent"))

	// Should return nil, nil (no error)
	if err != nil {
		t.Errorf("Load: got error %v, want nil", err)
	}
	if skills != nil {
		t.Errorf("Load returned %v, want nil", skills)
	}
}

func TestLoad_NoFrontMatter(t *testing.T) {
	tmpDir := t.TempDir()

	// Create skill without front matter
	skillDir := filepath.Join(tmpDir, "skill_no_meta")
	if err := os.Mkdir(skillDir, 0o755); err != nil {
		t.Fatalf("creating skill dir: %v", err)
	}
	skillContent := "Just a body with no front matter"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0o644); err != nil {
		t.Fatalf("writing skill: %v", err)
	}

	// Load skills (should skip this one with a warning)
	skills, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Should return empty list (skill was skipped)
	if len(skills) != 0 {
		t.Errorf("Load returned %d skills, want 0 (skill should have been skipped)", len(skills))
	}
}

func TestLoad_MalformedYAML(t *testing.T) {
	tmpDir := t.TempDir()

	// Create one good skill
	goodDir := filepath.Join(tmpDir, "good_skill")
	if err := os.Mkdir(goodDir, 0o755); err != nil {
		t.Fatalf("creating good_skill dir: %v", err)
	}
	goodContent := `---
name: Good Skill
description: This one works
---
Good body`
	if err := os.WriteFile(filepath.Join(goodDir, "SKILL.md"), []byte(goodContent), 0o644); err != nil {
		t.Fatalf("writing good skill: %v", err)
	}

	// Create one skill with malformed YAML
	badDir := filepath.Join(tmpDir, "bad_skill")
	if err := os.Mkdir(badDir, 0o755); err != nil {
		t.Fatalf("creating bad_skill dir: %v", err)
	}
	badContent := `---
name: Bad Skill
description: [invalid yaml: {
---
Bad body`
	if err := os.WriteFile(filepath.Join(badDir, "SKILL.md"), []byte(badContent), 0o644); err != nil {
		t.Fatalf("writing bad skill: %v", err)
	}

	// Load skills (should skip bad one, keep good one)
	skills, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Should have only 1 skill (the good one)
	if len(skills) != 1 {
		t.Errorf("Load returned %d skills, want 1", len(skills))
	}

	// Verify it's the good skill
	if len(skills) > 0 && skills[0].Name != "Good Skill" {
		t.Errorf("skill name = %q, want %q", skills[0].Name, "Good Skill")
	}
}

func TestLoad_FallbackToDirectoryName(t *testing.T) {
	tmpDir := t.TempDir()

	// Create skill without explicit name in front matter
	skillDir := filepath.Join(tmpDir, "my_skill_dir")
	if err := os.Mkdir(skillDir, 0o755); err != nil {
		t.Fatalf("creating skill dir: %v", err)
	}
	skillContent := `---
description: Skill without explicit name
---
Body content`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0o644); err != nil {
		t.Fatalf("writing skill: %v", err)
	}

	// Load skills
	skills, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Should have 1 skill with directory name as fallback
	if len(skills) != 1 {
		t.Errorf("Load returned %d skills, want 1", len(skills))
	}
	if len(skills) > 0 && skills[0].Name != "my_skill_dir" {
		t.Errorf("skill name = %q, want %q (directory name fallback)", skills[0].Name, "my_skill_dir")
	}
}