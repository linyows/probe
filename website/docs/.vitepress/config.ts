import { defineConfig } from 'vitepress'

const defaultThemeConfig = {
  logo: {
    light: '/probe-logo.svg',
    dark: '/probe-logo-dark.svg',
  },
  footer: {
    message: 'Released under the MIT License.',
    copyright: 'Copyright © 2025-present linyows',
  },
  search: {
    provider: 'local',
  },
  socialLinks: [
    { icon: 'github', link: 'https://github.com/linyows/probe' }
  ],
}

const enGuide = [
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

const enReference = [
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

const jaGuide = [
  {
    text: 'はじめに',
    base: '/ja/guide/introduction/',
    collapsed: false,
    items: [
      { text: 'Probeは何ですか？', link: 'what-is-probe' },
      { text: 'インストール', link: 'installation' },
      { text: 'Probeを理解する', link: 'understanding-probe' },
      { text: 'CLIの基本', link: 'cli-basics' },
      { text: 'クイックスタート', link: 'quickstart' },
      { text: '最初のワークフロー', link: 'your-first-workflow' }
    ]
  },
  {
    text: 'コンセプト',
    base: '/ja/guide/concepts/',
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
    base: '/ja/guide/how-tos/',
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
    text: 'チュートリアル',
    base: '/ja/guide/tutorials/',
    collapsed: false,
    items: [
      { text: 'API Testing Pipeline', link: 'api-testing-pipeline' },
      { text: 'First Monitoring System', link: 'first-monitoring-system' },
      { text: 'Multi-Environment Testing', link: 'multi-environment-testing' }
    ]
  },
]

const jaReference = [
  {
    text: 'リファレンス',
    base: '/ja/reference/',
    collapsed: false,
    items: [
      { text: 'CLI', link: 'cli-reference' },
      { text: 'アクション', link: 'actions-reference' },
      { text: '組込み関数', link: 'built-in-functions' },
      { text: '環境変数', link: 'environment-variables' },
      { text: 'YAMLフィールド', link: 'yaml-configuration' }
    ]
  },
]

const enNavSidebar = {
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
    '/guide/': { items: enGuide },
    '/reference/': { items: enReference },
  },
  editLink: {
    pattern: 'https://github.com/linyows/probe/edit/main/website/docs/:path',
    text: 'Edit this page on GitHub'
  },
  lastUpdated: {
    text: 'Updated at',
  },
}

const jaNavSidebar = {
  nav: [
    { text: 'ガイド', link: '/ja/guide/introduction/what-is-probe' },
    { text: 'リファレンス', link: '/ja/reference/yaml-configuration' },
    { text: '外部ドキュメント',
      items: [
        { text: 'Github Release', link: 'https://github.com/linyows/probe/releases' },
        { text: 'Go Docs', link: 'http://godoc.org/github.com/linyows/probe' },
        { text: 'Deep Wiki', link: 'https://deepwiki.com/linyows/probe' },
      ],
    },
  ],
  sidebar: {
    '/ja/guide/': { items: jaGuide },
    '/ja/reference/': { items: jaReference },
  },
  lastUpdated: {
    text: '更新日時',
  },
  editLink: {
    pattern: 'https://github.com/linyows/probe/edit/main/website/docs/:path',
    text: 'GitHubでこのページを編集する'
  }
}

export default defineConfig({
  title: "Probe",
  description: "A powerful YAML-based workflow automation tool designed for testing, monitoring, and automation tasks. Probe uses plugin-based actions to execute workflows, making it highly flexible and extensible.",
  rewrites: {
    'en/:rest*': ':rest*'
  },
  ignoreDeadLinks: true,
  themeConfig: {
    ...defaultThemeConfig,
    ...enNavSidebar,
  },
  locales: {
    root: {
      label: 'English',
    },
    ja: {
      label: '日本語',
      themeConfig: {
        ...jaNavSidebar,
      },
    },
  },
})
