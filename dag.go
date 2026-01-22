package probe

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
)

// DagRenderer is the interface for DAG rendering
type DagRenderer interface {
	Render() string
}

// DagRendererBase provides common functionality for DAG renderers
type DagRendererBase struct {
	workflow      *Workflow
	evaluatedVars map[string]any // cached evaluated vars (lazy loaded)
}

// NewDagRendererBase creates a new DagRendererBase
func NewDagRendererBase(w *Workflow) DagRendererBase {
	return DagRendererBase{
		workflow: w,
	}
}

// GetEvaluatedVars returns evaluated vars with lazy loading
func (b *DagRendererBase) GetEvaluatedVars() map[string]any {
	if b.evaluatedVars == nil {
		vars, err := b.workflow.evalVars()
		if err == nil {
			b.evaluatedVars = vars
		}
	}
	return b.evaluatedVars
}

// ExpandPath expands template variables in the path using evaluated workflow vars
func (b *DagRendererBase) ExpandPath(path string) string {
	vars := b.GetEvaluatedVars()
	if vars == nil {
		return path
	}

	// Build environment with evaluated vars
	env := map[string]any{
		"vars": vars,
	}

	expr := &Expr{}
	expanded, err := expr.EvalTemplate(path, env)
	if err != nil {
		return path // Return original path if expansion fails
	}
	return expanded
}

// ResolvePath resolves a (potentially relative) path using a fixed priority order:
//  1. If the path is absolute, it is returned as-is.
//  2. If workflow.basePath is set, first try the path relative to the workflow
//     directory (workflow.basePath/path).
//  3. If not found, try the path relative to the parent of the workflow directory
//     (for project-root relative paths; parentDir/path).
//  4. If still not found, fall back to resolving the path from the current working
//     directory using filepath.Abs, which matches the runtime's default behavior.
func (b *DagRendererBase) ResolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}

	// Try workflow directory first
	if b.workflow.basePath != "" {
		workflowRelPath := filepath.Join(b.workflow.basePath, path)
		if _, err := os.Stat(workflowRelPath); err == nil {
			return workflowRelPath
		}

		// Try parent of workflow directory (for project-root relative paths)
		parentDir := filepath.Dir(b.workflow.basePath)
		if parentDir != b.workflow.basePath {
			parentRelPath := filepath.Join(parentDir, path)
			if _, err := os.Stat(parentRelPath); err == nil {
				return parentRelPath
			}
		}
	}

	// Fall back to current directory (matches runtime behavior)
	absPath, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return absPath
}

// LoadEmbeddedJob loads a job definition from an embedded YAML file
func LoadEmbeddedJob(path string) (*Job, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}

	job := &Job{}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	if err = dec.Decode(job); err != nil {
		return nil, err
	}

	return job, nil
}
