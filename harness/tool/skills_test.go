package tool

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

	directory := filepath.Join(t.TempDir(), "skills")
	if err := os.MkdirAll(filepath.Join(directory, "alpha"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(directory, "beta"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "alpha", "SKILL.md"), []byte(`---
name: alpha
description: Handle alpha tasks.
---

Alpha instructions.
`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "beta", "SKILL.md"), []byte(`---
name: beta
description: Handle beta tasks.
metadata:
  ignored: true
---

Beta instructions.
`), 0o600); err != nil {
		t.Fatal(err)
	}

		t.Fatalf("skill errors = %v", skillErrors)
	}
	want := []Skill{
		{
			Name:        "alpha",
			Description: "Handle alpha tasks.",
			Path:        filepath.Join(directory, "alpha", "SKILL.md"),
		},
		{
			Name:        "beta",
			Description: "Handle beta tasks.",
			Path:        filepath.Join(directory, "beta", "SKILL.md"),
		},
	}
		t.Fatalf("skills = %#v, want %#v", got, want)
	}
}

		t.Fatalf("skill errors = %v", skillErrors)
	}
		t.Fatalf("skills = %#v, want empty", got)
	}
}

	directory := filepath.Join(t.TempDir(), "skills")
	invalidPath := filepath.Join(directory, "invalid", "SKILL.md")
	validPath := filepath.Join(directory, "valid", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(invalidPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(validPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(invalidPath, []byte("name: invalid\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(validPath, []byte("---\nname: valid\ndescription: Valid skill.\n---\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	want := []Skill{{Name: "valid", Description: "Valid skill.", Path: validPath}}
		t.Fatalf("skills = %#v, want %#v", got, want)
	}
	if len(skillErrors) != 1 ||
		!strings.Contains(skillErrors[0].Error(), "missing opening YAML frontmatter delimiter") {
		t.Fatalf("skill errors = %v", skillErrors)
	}
}
