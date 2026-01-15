package asciidag

// RenderMode controls the rendering direction of the DAG.
type RenderMode int

const (
	// RenderModeAuto automatically selects the best mode based on graph structure.
	// Simple chains use horizontal, complex graphs use vertical.
	RenderModeAuto RenderMode = iota

	// RenderModeVertical renders the DAG top-to-bottom (Sugiyama layout).
	RenderModeVertical

	// RenderModeHorizontal renders the DAG left-to-right (for simple chains).
	RenderModeHorizontal
)

// Option is a functional option for configuring a DAG.
type Option func(*DAG)

// WithRenderMode sets the rendering mode.
func WithRenderMode(mode RenderMode) Option {
	return func(d *DAG) {
		d.renderMode = mode
	}
}

// WithCrossingReductionPasses sets the number of crossing reduction passes.
// Default is 4. Higher values may produce better layouts but take longer.
// Values > 20 have diminishing returns.
func WithCrossingReductionPasses(passes int) Option {
	return func(d *DAG) {
		if passes < 0 {
			passes = 0
		}
		if passes > 1000 {
			passes = 0 // Probably a mistake
		}
		d.crossingReductionPasses = passes
	}
}
