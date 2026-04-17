// Package skills loads skill definitions from the filesystem.
package skills

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/ubunatic/paperclip-go/internal/domain"
)

// Load walks the skillsDir looking for SKILL.md files and returns a slice of parsed Skills.
// Returns an empty slice and nil error if the directory doesn't exist.
// Logs warnings for unparseable files but doesn't fail the entire load.
func Load(skillsDir string) ([]domain.Skill, error) {
	// Handle missing directory gracefully
	if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
		return []domain.Skill{}, nil
	}

	var skills []domain.Skill

	err := filepath.WalkDir(skillsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Look for SKILL.md files
		if !d.IsDir() && d.Name() == "SKILL.md" {
			skill, parseErr := parseSkillFile(path)
			if parseErr != nil {
				log.Printf("skills: warning parsing %s: %v", path, parseErr)
				return nil // Log warning but continue
			}
			skills = append(skills, skill)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walking skills directory: %w", err)
	}

	return skills, nil
}

// frontmatter holds the YAML frontmatter fields from a SKILL.md file.
type frontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// parseSkillFile reads a SKILL.md file, extracts YAML frontmatter (between --- delimiters),
// and returns a Skill with the parsed metadata and body content.
func parseSkillFile(path string) (domain.Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return domain.Skill{}, fmt.Errorf("reading file: %w", err)
	}

	content := string(data)

	// Extract YAML frontmatter (between --- delimiters)
	var name, description, body string

	if strings.HasPrefix(content, "---\n") {
		// Find the closing ---
		rest := content[4:] // Skip opening "---\n"
		endIdx := strings.Index(rest, "\n---\n")
		if endIdx != -1 {
			yamlBlock := rest[:endIdx]
			body = strings.TrimLeft(rest[endIdx+5:], "\n") // Skip "\n---\n"

			// Parse YAML frontmatter
			var fm frontmatter
			if err := yaml.Unmarshal([]byte(yamlBlock), &fm); err != nil {
				return domain.Skill{}, fmt.Errorf("parsing YAML frontmatter: %w", err)
			}

			name = fm.Name
			description = fm.Description
		}
	}

	// name is required
	if name == "" {
		return domain.Skill{}, fmt.Errorf("missing required 'name' field in frontmatter")
	}

	return domain.Skill{
		Name:        name,
		Description: description,
		Path:        path,
		Body:        body,
	}, nil
}
