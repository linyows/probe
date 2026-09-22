/*
 * The job graph of `heroWorkflow`, drawn from the `flowchart LR` that
 * `probe dag --mermaid` emits for it: one box per job, the step inside it,
 * and an edge per `needs`. The action name on each box is read from the
 * workflow and is the one addition to what Mermaid would draw.
 *
 * Inline SVG rather than mermaid.js: the graph is fixed, and a megabyte of
 * client-side layout engine to draw four boxes is not a trade worth making.
 */

type Node = {
  id: string
  job: string
  step: string
  action: string
  x: number
  y: number
}

const W = 196
const H = 76

const nodes: Node[] = [
  { id: 'signup', job: 'Create account', step: 'Post the form', action: 'http', x: 4, y: 80 },
  { id: 'db', job: 'Database', step: 'Row is pending', action: 'db', x: 306, y: 4 },
  { id: 'mailbox', job: 'Mailbox', step: 'Confirmation arrived', action: 'imap', x: 306, y: 156 },
  { id: 'activate', job: 'Activate', step: 'Follow the link', action: 'http', x: 608, y: 80 },
]

const edges = [
  { from: [200, 118], to: [306, 42] },
  { from: [200, 118], to: [306, 194] },
  { from: [502, 42], to: [608, 118] },
  { from: [502, 194], to: [608, 118] },
] as const

export function Flowchart({ title }: { title: string }) {
  return (
    <svg
      className="lp-flow"
      viewBox="0 0 808 236"
      role="img"
      aria-label={title}
      preserveAspectRatio="xMidYMid meet"
    >
      <defs>
        <marker
          id="lp-flow-arrow"
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

      {edges.map(({ from, to }, i) => {
        const [x1, y1] = from
        const [x2, y2] = to
        const mid = (x1 + x2) / 2
        return (
          <path
            key={i}
            className="lp-flow__edge"
            d={`M${x1} ${y1} C${mid} ${y1}, ${mid} ${y2}, ${x2} ${y2}`}
            markerEnd="url(#lp-flow-arrow)"
          />
        )
      })}

      {nodes.map(node => (
        <g key={node.id}>
          <rect className="lp-flow__job" x={node.x} y={node.y} width={W} height={H} rx="4" />
          <text className="lp-flow__jobName" x={node.x + 12} y={node.y + 21}>
            {node.job}
          </text>
          <text className="lp-flow__action" x={node.x + W - 12} y={node.y + 21} textAnchor="end">
            {node.action}
          </text>
          <rect
            className="lp-flow__step"
            x={node.x + 10}
            y={node.y + 34}
            width={W - 20}
            height={30}
            rx="2"
          />
          <text className="lp-flow__stepName" x={node.x + 22} y={node.y + 54}>
            {node.step}
          </text>
        </g>
      ))}
    </svg>
  )
}
