// Package skills provides functions for loading skills from disk.
package skills

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ubunatic/paperclip-go/internal/domain"
	"gopkg.in/yaml.v3"
)

// Load walks the given directory and loads all skills from SKILL.md files.
// For each subdirectory, it looks for a SKILL.md file with YAML front matter.
// Malformed skills are logged but skipped (do not error out).
// Missing directory returns nil, nil without error.
func Load(dir string) ([]domain.Skill, error) {
	// Handle missing directory gracefully
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading skills directory %s: %w", dir, err)
	}

	var skills []domain.Skill

	// Walk through each entry
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillPath := filepath.Join(dir, entry.Name(), "SKILL.md")
		skill, err := parseSkillFile(skillPath, entry.Name())
		if err != nil {
			log.Printf("skills: skipping %s: %v", entry.Name(), err)
			continue
		}
		if skill != nil {
			skills = append(skills, *skill)
		}
	}

	return skills, nil
}

// parseSkillFile reads and parses a SKILL.md file with YAML front matter.
// Returns nil, nil if the file doesn't exist or has no front matter.
// Returns an error only for actual parsing failures.
func parseSkillFile(path, name string) (*domain.Skill, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	// Parse YAML front matter: ---\nYAML\n---\nBody
	content := string(data)

	// Check for front matter delimiters
	if !strings.HasPrefix(content, "---\n") {
		return nil, fmt.Errorf("no YAML front matter (missing opening ---)")
	}

	// Find the closing delimiter
	parts := strings.SplitN(content[4:], "\n---\n", 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("no closing --- delimiter for front matter")
	}

	frontMatter := strings.TrimSpace(parts[0])
	body := strings.TrimSpace(parts[1])

	// Parse YAML front matter
	var meta struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}
	if err := yaml.Unmarshal([]byte(frontMatter), &meta); err != nil {
		return nil, fmt.Errorf("parsing YAML front matter: %w", err)
	}

	// Use provided name in front matter, fallback to directory name
	skillName := meta.Name
	if skillName == "" {
		skillName = name
	}

	skill := &domain.Skill{
		Name:        skillName,
		Description: meta.Description,
		Path:        path,
		Body:        body,
	}

	return skill, nil
}
