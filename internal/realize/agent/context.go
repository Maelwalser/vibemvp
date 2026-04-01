package agent

import (
	"github.com/vibe-mvp/internal/realize/dag"
	"github.com/vibe-mvp/internal/realize/skills"
)

// Context bundles everything an agent needs for one task invocation.
type Context struct {
	Task           *dag.Task
	SkillDocs      []skills.Doc
	PreviousErrors string // non-empty on retry attempts
}
