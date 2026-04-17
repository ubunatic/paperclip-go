package skills_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ubunatic/paperclip-go/internal/skills"
)

func TestLoadValidSkillFile(t *testing.T) {
	tempdir := t.TempDir()

	// Create a SKILL.md with valid YAML frontmatter
	skillDir := filepath.Join(tempdir, "test-skill")
	if err := os.Mkdir(skillDir, 0755); err != nil {
		t.Fatalf("creating skill dir: %v", err)
	}

	skillPath := filepath.Join(skillDir, "SKILL.md")
	content := `---
name: test-skill
description: A test skill for unit testing
---
# Test Skill

This is the body content of the skill markdown.

It can have multiple lines and sections.
`
	if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
		t.Fatalf("writing SKILL.md: %v", err)
	}

	// Load skills from the temp directory
	skillsList, err := skills.Load(tempdir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Verify we got one skill
	if len(skillsList) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skillsList))
	}

	// Verify the skill metadata
	skill := skillsList[0]
	if skill.Name != "test-skill" {
		t.Errorf("Name = %q, want %q", skill.Name, "test-skill")
	}
	if skill.Description != "A test skill for unit testing" {
		t.Errorf("Description = %q, want %q", skill.Description, "A test skill for unit testing")
	}
	if skill.Path != skillPath {
		t.Errorf("Path = %q, want %q", skill.Path, skillPath)
	}

	// Verify the body contains the markdown
	if !strings.Contains(skill.Body, "# Test Skill") {
		t.Errorf("Body should contain '# Test Skill', got: %q", skill.Body)
	}
	if !strings.Contains(skill.Body, "This is the body content") {
		t.Errorf("Body should contain body content, got: %q", skill.Body)
	}
}

func TestLoadMissingDirectory(t *testing.T) {
	// Try to load from a directory that doesn't exist
	skillsList, err := skills.Load("/nonexistent/directory/path")

	// Should return empty slice and nil error (graceful handling)
	if err != nil {
		t.Fatalf("Load should handle missing directory gracefully, got error: %v", err)
	}
	if len(skillsList) != 0 {
		t.Errorf("expected empty slice, got %d skills", len(skillsList))
	}
}

func TestLoadUnparseableYAML(t *testing.T) {
	tempdir := t.TempDir()

	// Create a SKILL.md with invalid YAML frontmatter
	skillDir := filepath.Join(tempdir, "bad-skill")
	if err := os.Mkdir(skillDir, 0755); err != nil {
		t.Fatalf("creating skill dir: %v", err)
	}

	skillPath := filepath.Join(skillDir, "SKILL.md")
	// YAML that will fail to parse due to invalid indentation/syntax
	content := `---
name: bad-skill
description: This has bad YAML
  invalid indentation: :
---
Body content
`
	if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
		t.Fatalf("writing SKILL.md: %v", err)
	}

	// Load skills from the temp directory
	skillsList, err := skills.Load(tempdir)

	// Should handle gracefully: return empty list and nil error (logged warning)
	if err != nil {
		t.Fatalf("Load should handle unparseable YAML gracefully, got error: %v", err)
	}
	if len(skillsList) != 0 {
		t.Errorf("expected 0 skills due to parse error, got %d", len(skillsList))
	}
}

func TestLoadMissingNameField(t *testing.T) {
	tempdir := t.TempDir()

	// Create a SKILL.md missing the required 'name' field
	skillDir := filepath.Join(tempdir, "no-name-skill")
	if err := os.Mkdir(skillDir, 0755); err != nil {
		t.Fatalf("creating skill dir: %v", err)
	}

	skillPath := filepath.Join(skillDir, "SKILL.md")
	content := `---
description: A skill without a name
---
Body content
`
	if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
		t.Fatalf("writing SKILL.md: %v", err)
	}

	// Load skills from the temp directory
	skillsList, err := skills.Load(tempdir)

	// Should handle gracefully: return empty list and nil error (logged warning)
	if err != nil {
		t.Fatalf("Load should handle missing name gracefully, got error: %v", err)
	}
	if len(skillsList) != 0 {
		t.Errorf("expected 0 skills due to missing name, got %d", len(skillsList))
	}
}

func TestLoadMultipleSkills(t *testing.T) {
	tempdir := t.TempDir()

	// Create two skill directories with SKILL.md files
	for i, name := range []string{"skill-one", "skill-two"} {
		skillDir := filepath.Join(tempdir, name)
		if err := os.Mkdir(skillDir, 0755); err != nil {
			t.Fatalf("creating skill dir: %v", err)
		}

		skillPath := filepath.Join(skillDir, "SKILL.md")
		content := `---
name: ` + name + `
description: This is ` + name + `
---
Body for ` + name
		if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
			t.Fatalf("writing SKILL.md %d: %v", i, err)
		}
	}

	// Load skills
	skillsList, err := skills.Load(tempdir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(skillsList) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(skillsList))
	}

	// Verify both skills are present (order may vary)
	names := map[string]bool{
		skillsList[0].Name: true,
		skillsList[1].Name: true,
	}
	if !names["skill-one"] {
		t.Error("expected skill-one in results")
	}
	if !names["skill-two"] {
		t.Error("expected skill-two in results")
	}
}

func TestLoadIgnoresNonSkillFiles(t *testing.T) {
	tempdir := t.TempDir()

	// Create a valid skill
	skillDir := filepath.Join(tempdir, "real-skill")
	if err := os.Mkdir(skillDir, 0755); err != nil {
		t.Fatalf("creating skill dir: %v", err)
	}

	skillPath := filepath.Join(skillDir, "SKILL.md")
	content := `---
name: real-skill
description: A real skill
---
Body
`
	if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
		t.Fatalf("writing SKILL.md: %v", err)
	}

	// Create some non-SKILL.md files in the same directory
	if err := os.WriteFile(filepath.Join(skillDir, "README.md"), []byte("readme"), 0644); err != nil {
		t.Fatalf("writing README.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "other.txt"), []byte("other"), 0644); err != nil {
		t.Fatalf("writing other.txt: %v", err)
	}

	// Load skills
	skillsList, err := skills.Load(tempdir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Should only get the SKILL.md file, not README or other files
	if len(skillsList) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skillsList))
	}
	if skillsList[0].Name != "real-skill" {
		t.Errorf("expected real-skill, got %q", skillsList[0].Name)
	}
}
