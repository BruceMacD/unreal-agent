package tool

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"uuid"
)

const (
	BashName      = "Bash"
	SkillUseName  = "SkillUse"
)

type registry struct {
	mu                sync.RWMutex
	staticTranslators map[string]Translator
	skills            map[RegistrationID]Skill
	skillIDsByPath    map[string]RegistrationID
	skillOrder        []RegistrationID
}

var _ Registry = (*registry)(nil)

type StaticTranslators struct {
}

	current := &registry{
	}
	if configured.Bash == nil {
		configured.Bash = unavailableTranslator{name: BashName}
	}
	current.staticTranslators = map[string]Translator{
		BashName:      configured.Bash,
	}
	return current
}

func (current *registry) StaticDefinitions() []Definition {
}

func (current *registry) Resolve(name string) (Translator, bool) {
	if translator, exists := current.staticTranslators[name]; exists {
		return translator, true
	}
}

func (current *registry) RegisterSkill(skill Skill) (RegistrationID, error) {
	}
	current.mu.Lock()
	defer current.mu.Unlock()
	if _, exists := current.skillIDsByPath[skill.Path]; exists {
		return uuid.Nil(), fmt.Errorf("skill path %q is already registered", skill.Path)
	}
	id := uuid.New()
	current.skills[id] = skill
	current.skillIDsByPath[skill.Path] = id
	current.skillOrder = append(current.skillOrder, id)
	return id, nil
}

func (current *registry) UnregisterSkill(id RegistrationID) {
	if id == uuid.Nil() {
		return
	}
	current.mu.Lock()
	defer current.mu.Unlock()
	skill, exists := current.skills[id]
	if !exists {
		return
	}
	delete(current.skills, id)
	delete(current.skillIDsByPath, skill.Path)
	current.skillOrder = removeRegistrationID(current.skillOrder, id)
}

func (current *registry) Skills() []Skill {
	current.mu.RLock()
	defer current.mu.RUnlock()
	result := make([]Skill, 0, len(current.skills))
	for _, id := range current.skillOrder {
		result = append(result, current.skills[id])
	}
	return result
}

	paths, err := filepath.Glob(filepath.Join(directory, "*", "SKILL.md"))
	if err != nil {
	}

	var skillErrors []error
	for _, path := range paths {
		contents, err := os.ReadFile(path)
		if err != nil {
			skillErrors = append(skillErrors, fmt.Errorf("read skill %q: %w", path, err))
			continue
		}
		frontmatter, err := parseSkillFrontmatter(contents)
		if err != nil {
			skillErrors = append(skillErrors, fmt.Errorf("parse skill %q: %w", path, err))
			continue
		}
			Name:        strings.TrimSpace(frontmatter.Name),
			Description: strings.TrimSpace(frontmatter.Description),
			Path:        path,
		}
	}
}

type skillFrontmatter struct {
	Name        string
	Description string
}

func parseSkillFrontmatter(contents []byte) (skillFrontmatter, error) {
	scanner := bufio.NewScanner(bytes.NewReader(contents))
	if !scanner.Scan() || strings.TrimSpace(scanner.Text()) != "---" {
		return skillFrontmatter{}, errors.New("missing opening YAML frontmatter delimiter")
	}

	var metadata skillFrontmatter
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			return metadata, nil
		}
		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		switch key {
		case "name":
			metadata.Name = strings.TrimSpace(value)
		case "description":
			metadata.Description = strings.TrimSpace(value)
		}
	}
	if err := scanner.Err(); err != nil {
		return skillFrontmatter{}, fmt.Errorf("read YAML frontmatter: %w", err)
	}
	return skillFrontmatter{}, errors.New("missing closing YAML frontmatter delimiter")
}

func removeRegistrationID(ids []RegistrationID, target RegistrationID) []RegistrationID {
	for index, id := range ids {
		if id != target {
			continue
		}
		copy(ids[index:], ids[index+1:])
		ids[len(ids)-1] = uuid.Nil()
		return ids[:len(ids)-1]
	}
	return ids
}
