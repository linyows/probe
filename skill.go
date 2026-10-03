package probe

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

// skill is the agent skill in skills/probe/SKILL.md. It lives there so that
// skill installers that read a repository find it, and it is built into the
// binary so that probe skill installs the copy that matches this version.
//
//go:embed skills/probe/SKILL.md
var skill string

// DefaultSkillDir is where probe skill install puts the skill when no
// directory is given: the project-level skills directory of Claude Code.
const DefaultSkillDir = ".claude/skills/probe"

// Skill returns the agent skill as Markdown.
func Skill() string {
	return skill
}

// InstallSkill writes SKILL.md into dir, creating it, and returns the path it
// wrote. An existing SKILL.md is replaced, so running it again after an
// upgrade brings the skill up to date.
//
// The file is written next to its destination and renamed over it. A rename
// replaces the directory entry itself, so when a checked-out project carries
// SKILL.md as a symlink to a file elsewhere, that file is left alone instead
// of being overwritten through the link.
func InstallSkill(dir string) (string, error) {
	if dir == "" {
		dir = DefaultSkillDir
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create the skill directory: %w", err)
	}
	path := filepath.Join(dir, "SKILL.md")

	tmp, err := os.CreateTemp(dir, ".SKILL.md.*")
	if err != nil {
		return "", fmt.Errorf("failed to write the skill: %w", err)
	}
	_, err = tmp.WriteString(skill)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Chmod(tmp.Name(), 0o644)
	}
	if err == nil {
		err = os.Rename(tmp.Name(), path)
	}
	if err != nil {
		_ = os.Remove(tmp.Name())
		return "", fmt.Errorf("failed to write the skill: %w", err)
	}
	return path, nil
}
