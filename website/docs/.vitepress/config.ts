import { defineConfig } from 'vitepress'

// https://vitepress.dev/reference/site-config
export default defineConfig({
  title: "Probe",
  description: "A powerful YAML-based workflow automation tool designed for testing, monitoring, and automation tasks. Probe uses plugin-based actions to execute workflows, making it highly flexible and extensible.",
  ignoreDeadLinks: true,
  themeConfig: {
    // https://vitepress.dev/reference/default-theme-config
    nav: [
      { text: 'Quickstart', link: '/get-quickstart/quickstart' },
      { text: 'Reference', link: '/reference/yaml-configuration' },
      { text: 'External',
        items: [
          { text: 'Github Release', link: 'https://github.com/linyows/probe/releases' },
          { text: 'Go Docs', link: 'http://godoc.org/github.com/linyows/probe' },
          { text: 'Deep Wiki', link: 'https://deepwiki.com/linyows/probe' },
        ],
      },
    ],
    sidebar: [
      {
        text: 'Get Started',
        items: [
          { text: 'Installation', link: '/get-started/installation' },
          { text: 'Understanding Probe', link: '/get-started/understanding-probe' },
          { text: 'CLI Basics', link: '/get-started/cli-basics' },
          { text: 'Quickstart', link: '/get-started/quickstart' },
          { text: 'Your First Workflow', link: '/get-started/your-first-workflow' }
        ]
      },
      {
        text: 'Concepts',
        items: [
          { text: 'Workflows', link: '/concepts/workflows' },
          { text: 'Jobs and Steps', link: '/concepts/jobs-and-steps' },
          { text: 'Actions', link: '/concepts/actions' },
          { text: 'Data Flow', link: '/concepts/data-flow' },
          { text: 'Expressions and Templates', link: '/concepts/expressions-and-templates' },
          { text: 'File Merging', link: '/concepts/file-merging' },
          { text: 'Execution Model', link: '/concepts/execution-model' },
          { text: 'Testing and Assertions', link: '/concepts/testing-and-assertions' },
          { text: 'Error Handling', link: '/concepts/error-handling' }
        ]
      },
      {
        text: 'How-tos',
        items: [
          { text: 'API Testing', link: '/how-tos/api-testing' },
          { text: 'Environment Management', link: '/how-tos/environment-management' },
          { text: 'Error Handling Strategies', link: '/how-tos/error-handling-strategies' },
          { text: 'Monitoring Workflows', link: '/how-tos/monitoring-workflows' },
          { text: 'Performance Testing', link: '/how-tos/performance-testing' }
        ]
      },
      {
        text: 'Tutorials',
        items: [
          { text: 'API Testing Pipeline', link: '/tutorials/api-testing-pipeline' },
          { text: 'First Monitoring System', link: '/tutorials/first-monitoring-system' },
          { text: 'Multi-Environment Testing', link: '/tutorials/multi-environment-testing' }
        ]
      },
      {
        text: 'Reference',
        items: [
          { text: 'CLI Reference', link: '/reference/cli-reference' },
          { text: 'Actions Reference', link: '/reference/actions-reference' },
          { text: 'Built-in Functions', link: '/reference/built-in-functions' },
          { text: 'Environment Variables', link: '/reference/environment-variables' },
          { text: 'YAML Configuration', link: '/reference/yaml-configuration' }
        ]
      },
    ],
    socialLinks: [
      { icon: 'github', link: 'https://github.com/linyows/probe' }
    ]
  }
})
