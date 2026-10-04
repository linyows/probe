package probe

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
	"github.com/linyows/probe/dag"
	"github.com/linyows/probe/expr"
)

// graphBuilder resolves the files that embedded steps run, for Workflow.Graph.
type graphBuilder struct {
	workflow      *Workflow
	evaluatedVars map[string]any // cached evaluated vars (lazy loaded)
}

// vars returns the workflow's evaluated vars, evaluating them on first use.
func (b *graphBuilder) vars() map[string]any {
	if b.evaluatedVars == nil {
		vars, err := b.workflow.evalVars()
		if err == nil {
			b.evaluatedVars = vars
		}
	}
	return b.evaluatedVars
}

// expandPath expands template variables in the path using evaluated workflow vars
func (b *graphBuilder) expandPath(path string) string {
	vars := b.vars()
	if vars == nil {
		return path
	}

	// Build environment with evaluated vars
	env := map[string]any{
		"vars": vars,
	}

	ev := &expr.Expr{}
	expanded, err := ev.EvalTemplate(path, env)
	if err != nil {
		return path // Return original path if expansion fails
	}
	return expanded
}

// resolvePath resolves a (potentially relative) path using a fixed priority order:
//  1. If the path is absolute, it is returned as-is.
//  2. If workflow.basePath is set, first try the path relative to the workflow
//     directory (workflow.basePath/path).
//  3. If not found, try the path relative to the parent of the workflow directory
//     (for project-root relative paths; parentDir/path).
//  4. If still not found, fall back to resolving the path from the current working
//     directory using filepath.Abs, which matches the runtime's default behavior.
func (b *graphBuilder) resolvePath(path string) string {
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

// Graph returns the jobs and steps of the workflow in the form the DAG
// renderers draw. The steps of an embedded job are read from its file, at the
// path its `with.path` names after the workflow's vars are expanded.
func (w *Workflow) Graph() dag.Graph {
	b := &graphBuilder{workflow: w}
	g := dag.Graph{Jobs: make([]dag.Job, 0, len(w.Jobs))}
	for _, job := range w.Jobs {
		id := job.ID
		if id == "" {
			id = job.Name
		}
		gj := dag.Job{ID: id, Name: job.Name, Needs: job.Needs, Steps: make([]dag.Step, 0, len(job.Steps))}
		for _, st := range job.Steps {
			gj.Steps = append(gj.Steps, b.step(st))
		}
		g.Jobs = append(g.Jobs, gj)
	}
	return g
}

// step returns the graph step for st, with the steps of the job it embeds.
func (b *graphBuilder) step(st *Step) dag.Step {
	s := dag.Step{Name: st.Name, Embedded: st.Uses == "embedded"}
	if s.Name == "" {
		s.Name = st.Uses
	}
	if !s.Embedded {
		return s
	}
	path, ok := st.With["path"].(string)
	if !ok {
		return s
	}
	expanded := b.expandPath(path)
	job, err := LoadEmbeddedJob(b.resolvePath(expanded))
	if err != nil || len(job.Steps) == 0 {
		return s
	}
	s.EmbeddedFile = filepath.Base(expanded)
	for _, es := range job.Steps {
		name := es.Name
		if name == "" {
			name = es.Uses
		}
		s.EmbeddedSteps = append(s.EmbeddedSteps, name)
	}
	return s
}
