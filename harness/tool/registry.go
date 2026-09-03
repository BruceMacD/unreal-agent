package tool

import (
	"errors"
	"fmt"
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
func (current *registry) RegisterSkill(skill Skill) (RegistrationID, error) {
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
