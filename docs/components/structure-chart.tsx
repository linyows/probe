/*
 * What a workflow is made of: jobs inside the workflow, steps inside a job.
 * Steps carry arrows because they run in order; the two jobs on the top row
 * carry none between them because nothing makes one wait for the other.
 */

const JOB_W = 360
const JOB_H = 84
const CHIP_W = 72
const CHIP_H = 26
const CHIP_GAP = 30

const jobs = [
  { id: 'a', x: 30, y: 48, steps: 3 },
  { id: 'b', x: 414, y: 48, steps: 2 },
  { id: 'c', x: 222, y: 168, steps: 1 },
]

export function StructureChart({
  title,
  workflowLabel,
  jobLabel,
  stepLabel,
  needsLabel,
}: {
  title: string
  workflowLabel: string
  jobLabel: string
  stepLabel: string
  needsLabel: string
}) {
  return (
    <svg
      className="lp-flow"
      viewBox="0 0 808 268"
      role="img"
      aria-label={title}
      preserveAspectRatio="xMidYMid meet"
    >
      <defs>
        <marker
          id="lp-struct-arrow"
          viewBox="0 0 10 10"
          refX="9"
          refY="5"
          markerWidth="4"
          markerHeight="4"
          orient="auto-start-reverse"
        >
          <path d="M0 0 L10 5 L0 10 z" className="lp-flow__head" />
        </marker>
      </defs>

      <rect className="lp-flow__frame" x="4" y="4" width="800" height="260" rx="4" />
      <text className="lp-flow__jobName" x="24" y="32">
        {workflowLabel}
      </text>

      <path
        className="lp-flow__edge"
        d="M210 132 C210 152, 402 148, 402 168"
        markerEnd="url(#lp-struct-arrow)"
      />
      <path
        className="lp-flow__edge"
        d="M594 132 C594 152, 402 148, 402 168"
        markerEnd="url(#lp-struct-arrow)"
      />
      <text className="lp-flow__action" x="402" y="156" textAnchor="middle">
        {needsLabel}
      </text>

      {jobs.map(job => (
        <g key={job.id}>
          <rect className="lp-flow__job" x={job.x} y={job.y} width={JOB_W} height={JOB_H} rx="4" />
          <text className="lp-flow__stepName" x={job.x + 16} y={job.y + 22}>
            {jobLabel}
          </text>
          {Array.from({ length: job.steps }, (_, i) => {
            const x = job.x + 16 + i * (CHIP_W + CHIP_GAP)
            return (
              <g key={i}>
                {i > 0 && (
                  <path
                    className="lp-flow__edge"
                    d={`M${x - CHIP_GAP + 6} ${job.y + 49} H${x - 6}`}
                    markerEnd="url(#lp-struct-arrow)"
                  />
                )}
                <rect
                  className="lp-flow__step"
                  x={x}
                  y={job.y + 36}
                  width={CHIP_W}
                  height={CHIP_H}
                  rx="2"
                />
                <text
                  className="lp-flow__chip"
                  x={x + CHIP_W / 2}
                  y={job.y + 53}
                  textAnchor="middle"
                >
                  {stepLabel}
                </text>
              </g>
            )
          })}
        </g>
      ))}
    </svg>
  )
}
