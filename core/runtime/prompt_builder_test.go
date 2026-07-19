package runtime

import (
	"strings"
	"testing"

	"danqing-teams/core/domain"
)

func TestBuildSkillMetadataIncludesCategory(t *testing.T) {
	out := buildSkillMetadata([]domain.Skill{
		{
			Name:        "writing-plans",
			Description: "File-level implementation plans",
			Metadata:    map[string]string{"category": "coding"},
		},
		{
			Name:        "brainstorming",
			Description: "Clarify requirements before building",
			Metadata:    map[string]string{"category": "work"},
		},
		{
			Name:        "skill-creator",
			Description: "Create new skills",
			Metadata:    map[string]string{"category": "general"},
			SystemHint:  "use for skill authoring",
		},
		{
			Name:        "legacy",
			Description: "No category",
		},
	})

	if !strings.Contains(out, "<category>coding</category>") {
		t.Fatalf("missing coding category:\n%s", out)
	}
	if !strings.Contains(out, "<category>work</category>") {
		t.Fatalf("missing work category:\n%s", out)
	}
	if !strings.Contains(out, "<category>general</category>") {
		t.Fatalf("missing general category:\n%s", out)
	}
	if !strings.Contains(out, "<path>legacy</path>\n    <description>") {
		t.Fatalf("legacy skill should omit category:\n%s", out)
	}
	if !strings.Contains(out, "<hint>use for skill authoring</hint>") {
		t.Fatalf("missing system hint:\n%s", out)
	}
}

func TestBuildSkillMetadataEmpty(t *testing.T) {
	if got := buildSkillMetadata(nil); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}
