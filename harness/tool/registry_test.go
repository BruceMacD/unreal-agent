package tool

import (
	"reflect"
	"testing"
	"uuid"

	"github.com/unreallabsai/unreal-agent/harness/llm"
	"github.com/unreallabsai/unreal-agent/harness/operation"
)

type fixedTranslator struct {
	status CallStatus
	result llm.ToolResult
}

type recordingContext struct {
	specs []operation.Spec
}

func (ctx *recordingContext) Submit(spec operation.Spec) operation.ID {
	ctx.specs = append(ctx.specs, spec)
	return "operation-1"
}

func (translator *fixedTranslator) Translate(Context, llm.ToolCall) CallStatus {
	return translator.status
}

func (translator *fixedTranslator) TranslateResult(
	_ string,
	_ CallStatus,
	_ []operation.Operation,
) (llm.ToolResult, error) {
	return translator.result, nil
}

	translators := StaticTranslators{
	}
	definitions := registry.StaticDefinitions()
	if len(definitions) != len(wantNames) {
		t.Fatalf("static definitions = %#v", definitions)
	}
	for index, want := range wantNames {
		if definitions[index].Tool.Name != want {
			t.Fatalf("static definition %d name = %q, want %q", index, definitions[index].Tool.Name, want)
		}
	}
		got, exists := registry.Resolve(name)
		if !exists || got != want {
			t.Fatalf("resolve %q = (%#v, %t), want (%#v, true)", name, got, exists, want)
		}
	}
	if translator, exists := registry.Resolve("unknown"); exists || translator != nil {
		t.Fatalf("resolve unknown = (%#v, %t), want (nil, false)", translator, exists)
	}
}

func TestRegistryOwnsCanonicalBashDefinition(t *testing.T) {
	want := llm.Tool{
		Type:        llm.ToolFunction,
		Name:        BashName,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{
					"type":        "string",
					"description": "The shell command to execute.",
				},
			},
			"required": []any{"command"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Bash definition = %#v, want %#v", got, want)
	}
}

func TestRegistryStaticDefinitionsReturnsIndependentValues(t *testing.T) {
	definitions := registry.StaticDefinitions()
	definitions[0].Tool.Name = "changed"
	definitions[0].Tool.Parameters["changed"] = true
	got := registry.StaticDefinitions()[0].Tool
	if got.Name != BashName {
		t.Fatalf("static definition name = %q, want %q", got.Name, BashName)
	}
	if _, exists := got.Parameters["changed"]; exists {
		t.Fatal("static definition retained a caller mutation")
	}
}

func TestRegistryRegistersListsAndUnregistersSkills(t *testing.T) {
	registry := NewRegistry(StaticTranslators{})
	first := Skill{Name: "go-review", Description: "Review Go code", Path: "/skills/go/SKILL.md"}
	second := Skill{Name: "documents", Description: "Edit documents", Path: "/skills/docs/SKILL.md"}
	firstID, err := registry.RegisterSkill(first)
	if err != nil {
		t.Fatal(err)
	}
	secondID, err := registry.RegisterSkill(second)
	if err != nil {
		t.Fatal(err)
	}
	if got := registry.Skills(); !reflect.DeepEqual(got, []Skill{first, second}) {
		t.Fatalf("skills = %#v", got)
	}

	copyOfSkills := registry.Skills()
	copyOfSkills[0].Path = "changed"
	if got := registry.Skills()[0]; got != first {
		t.Fatalf("registered skill = %#v, want %#v", got, first)
	}

	registry.UnregisterSkill(firstID)
	registry.UnregisterSkill(firstID)
	registry.UnregisterSkill(uuid.Nil())
	if got := registry.Skills(); !reflect.DeepEqual(got, []Skill{second}) {
		t.Fatalf("skills after unregister = %#v", got)
	}
	if firstID == secondID {
		t.Fatal("registration IDs are equal")
	}
}

func TestRegistryRejectsInvalidAndDuplicateSkills(t *testing.T) {
	registry := NewRegistry(StaticTranslators{})
	if _, err := registry.RegisterSkill(Skill{}); err == nil || err.Error() != "skill path must be set" {
		t.Fatalf("missing path error = %v", err)
	}
	if _, err := registry.RegisterSkill(Skill{Path: "/skill"}); err == nil ||
		err.Error() != "skill name must be set" {
		t.Fatalf("missing name error = %v", err)
	}
	if _, err := registry.RegisterSkill(Skill{Name: "review", Path: "/skill"}); err == nil ||
		err.Error() != "skill description must be set" {
		t.Fatalf("missing description error = %v", err)
	}
	skill := Skill{Name: "review", Description: "Review code", Path: "/skill"}
	if _, err := registry.RegisterSkill(skill); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.RegisterSkill(skill); err == nil ||
		err.Error() != `skill path "/skill" is already registered` {
		t.Fatalf("duplicate path error = %v", err)
	}
}
