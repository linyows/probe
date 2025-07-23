package probe



type Workflow struct {
	Name        string         `yaml:"name",validate:"required"`
	Description string         `yaml:"description,omitempty"`
	Jobs        []Job          `yaml:"jobs",validate:"required"`
	Vars        map[string]any `yaml:"vars"`
	exitStatus  int
	env         map[string]string
}

func (w *Workflow) SetExitStatus(isErr bool) {
	if isErr {
		w.exitStatus = 1
	}
}

func (w *Workflow) Env() map[string]string {
	if len(w.env) == 0 {
		w.env = EnvMap()
	}
	return w.env
}

func (w *Workflow) evalVars() (map[string]any, error) {
	env := StrmapToAnymap(w.Env())
	vars := make(map[string]any)

	expr := &Expr{}
	for k, v := range w.Vars {
		if mapV, ok := v.(map[string]any); ok {
			vars[k] = expr.EvalTemplateMap(mapV, env)
		} else if strV, ok2 := v.(string); ok2 {
			output, err := expr.EvalTemplate(strV, env)
			if err != nil {
				return vars, err
			}
			vars[k] = output
		}
	}

	return vars, nil
}

func (w *Workflow) newJobContext(c Config, vars map[string]any) JobContext {
	return JobContext{
		Vars:   vars,
		Logs:   []map[string]any{},
		Config: c,
		Output: NewOutput(c.Verbose),
	}
}