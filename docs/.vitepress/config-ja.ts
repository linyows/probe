const guide = [
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
      { text: 'ワークフロー', link: 'workflows' },
      { text: 'ジョブとステップ', link: 'jobs-and-steps' },
      { text: 'アクション', link: 'actions' },
      { text: 'データフロー', link: 'data-flow' },
      { text: '式とテンプレート', link: 'expressions-and-templates' },
      { text: 'ファイルマージ', link: 'file-merging' },
      { text: '実行モデル', link: 'execution-model' },
      { text: 'テストとアサーション', link: 'testing-and-assertions' },
      { text: 'エラーハンドリング', link: 'error-handling' }
    ]
  },
  {
    text: 'やりかた',
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

const reference = [
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

export const jaNavSidebar = {
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
    '/ja/guide/': guide,
    '/ja/reference/': reference,
  },
  lastUpdated: {
    text: '更新日時',
  },
  editLink: {
    pattern: 'https://github.com/linyows/probe/edit/main/docs/src/:path',
    text: 'GitHubでこのページを編集する'
  }
}
