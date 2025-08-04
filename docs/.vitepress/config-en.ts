const guide = [
  {
    text: 'Introduction',
    base: '/guide/introduction/',
    collapsed: false,
    items: [
      { text: 'What is Probe?', link: 'what-is-probe' },
      { text: 'Installation', link: 'installation' },
      { text: 'Understanding Probe', link: 'understanding-probe' },
      { text: 'CLI Basics', link: 'cli-basics' },
      { text: 'Quickstart', link: 'quickstart' },
      { text: 'Your First Workflow', link: 'your-first-workflow' }
    ]
  },
  {
    text: 'Concepts',
    base: '/guide/concepts/',
    collapsed: false,
    items: [
      { text: 'Workflows', link: 'workflows' },
      { text: 'Jobs and Steps', link: 'jobs-and-steps' },
      { text: 'Actions', link: 'actions' },
      { text: 'Data Flow', link: 'data-flow' },
      { text: 'Expressions and Templates', link: 'expressions-and-templates' },
      { text: 'File Merging', link: 'file-merging' },
      { text: 'Execution Model', link: 'execution-model' },
      { text: 'Testing and Assertions', link: 'testing-and-assertions' },
      { text: 'Error Handling', link: 'error-handling' }
    ]
  },
  {
    text: 'How-tos',
    base: '/guide/how-tos/',
    collapsed: false,
    items: [
      { text: 'API Testing', link: 'api-testing' },
      { text: 'Environment Management', link: 'environment-management' },
      { text: 'Error Handling Strategies', link: 'error-handling-strategies' },
      { text: 'Monitoring Workflows', link: 'monitoring-workflows' },
      { text: 'Performance Testing', link: 'performance-testing' }
    ]
  },
  {
    text: 'Tutorials',
    base: '/guide/tutorials/',
    collapsed: false,
    items: [
      { text: 'API Testing Pipeline', link: 'api-testing-pipeline' },
      { text: 'First Monitoring System', link: 'first-monitoring-system' },
      { text: 'Multi-Environment Testing', link: 'multi-environment-testing' }
    ]
  },
]

const reference = [
  {
    text: 'Reference',
    base: '/reference/',
    collapsed: false,
    items: [
      { text: 'CLI Reference', link: 'cli-reference' },
      { text: 'Actions Reference', link: 'actions-reference' },
      { text: 'Built-in Functions', link: 'built-in-functions' },
      { text: 'Environment Variables', link: 'environment-variables' },
      { text: 'YAML Configuration', link: 'yaml-configuration' }
    ]
  },
]

export const enNavSidebar = {
  nav: [
    { text: 'Guide', link: '/guide/introduction/what-is-probe' },
    { text: 'Reference', link: '/reference/yaml-configuration' },
    { text: 'External',
      items: [
        { text: 'Github Release', link: 'https://github.com/linyows/probe/releases' },
        { text: 'Go Docs', link: 'http://godoc.org/github.com/linyows/probe' },
        { text: 'Deep Wiki', link: 'https://deepwiki.com/linyows/probe' },
      ],
    },
  ],
  sidebar: {
    '/guide/': guide,
    '/reference/': reference,
  },
  editLink: {
    pattern: 'https://github.com/linyows/probe/edit/main/docs/src/:path',
    text: 'Edit this page on GitHub'
  },
  lastUpdated: {
    text: 'Updated at',
  },
}
