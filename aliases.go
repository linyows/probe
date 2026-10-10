package probe

import (
	"fmt"
	"maps"
	"regexp"
	"slices"

	"github.com/linyows/probe/actionref"
	"github.com/linyows/probe/expr"
)

// aliasName is a name a workflow can give an external action: one that
// cannot be taken for a path or a repository, or hold a template.
var aliasName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)

// aliasProblem is what is wrong with a name the workflow's actions give,
// or with a use of one.
type aliasProblem struct {
	// name is the name under actions, or empty when job is set.
	name string
	// job is the index of the job whose defaults are wrong, with action the
	// key of them that is.
	job    int
	action string
	msg    string
}

func (a aliasProblem) Error() string {
	return a.msg
}

// checkAliases returns what is wrong with the names the workflow's actions
// give, in the order of the names. builtin are the names of the actions of
// Probe, which a name must not be; when it is empty, that is not checked.
func (w *Workflow) checkAliases(builtin []string) []aliasProblem {
	var problems []aliasProblem
	for _, name := range slices.Sorted(maps.Keys(w.Actions)) {
		ref := w.Actions[name]
		var msg string
		switch {
		case !aliasName.MatchString(name):
			msg = fmt.Sprintf("actions: the name %q must be letters, digits, '_' and '-', starting with a letter or a digit", name)
		case slices.Contains(builtin, name):
			msg = fmt.Sprintf("actions: the name %q is that of an action of Probe, which a workflow cannot give to another", name)
		case !actionref.IsExternal(ref):
			msg = fmt.Sprintf("actions: %s must name an external action, as github.com/<owner>/<repo>@<commit SHA> or a local path starting with ./, not %q", name, ref)
		// A local path is taken as it is written, so a template in one would
		// name a directory with braces in its name.
		case len(expr.TemplateExprs(ref)) > 0:
			msg = fmt.Sprintf("actions: %s must name an external action as it is, not by a template: %q", name, ref)
		default:
			if _, err := actionref.Parse(ref); err != nil {
				msg = fmt.Sprintf("actions: %s: %s", name, err)
			}
		}
		if msg != "" {
			problems = append(problems, aliasProblem{name: name, msg: msg})
		}
	}
	return problems
}

// resolveAliases writes, in place of each name the workflow's actions give,
// the external action it names: in the uses of the steps and in the keys of
// the defaults of the jobs. What runs the workflow, the guard included, then
// sees the action as if the workflow had named it in full, so that a name
// lets nothing run that the action it stands for would not.
//
// A name that is wrong, as checkAliases tells, is left as it is written. It
// returns the defaults that are written for an action twice, once by its
// name and once in full.
func (w *Workflow) resolveAliases(wrong []aliasProblem) []aliasProblem {
	names := maps.Clone(w.Actions)
	for _, p := range wrong {
		delete(names, p.name)
	}
	if len(names) == 0 {
		return nil
	}

	var problems []aliasProblem
	for i := range w.Jobs {
		job := &w.Jobs[i]
		for _, st := range job.Steps {
			if st == nil {
				continue
			}
			if ref, ok := names[st.Uses]; ok {
				st.Uses = ref
			}
		}
		defaults, ok := job.Defaults.(map[string]any)
		if !ok {
			continue
		}
		for _, action := range slices.Sorted(maps.Keys(defaults)) {
			ref, ok := names[action]
			if !ok {
				continue
			}
			if _, twice := defaults[ref]; twice {
				problems = append(problems, aliasProblem{job: i, action: action, msg: fmt.Sprintf("defaults: %s and %s are the same action, whose defaults can be written once", action, ref)})
				continue
			}
			defaults[ref] = defaults[action]
			delete(defaults, action)
		}
	}
	return problems
}
