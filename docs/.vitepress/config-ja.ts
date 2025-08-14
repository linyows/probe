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
    text: 'やり方',
    base: '/ja/guide/how-tos/',
    collapsed: false,
    items: [
      { text: 'APIテスト', link: 'api-testing' },
      { text: '環境管理', link: 'environment-management' },
      { text: 'エラーハンドリング戦略', link: 'error-handling-strategies' },
      { text: 'モニタリングワークフロー', link: 'monitoring-workflows' },
      { text: 'パフォーマンステスト', link: 'performance-testing' }
    ]
  },
  {
    text: 'チュートリアル',
    base: '/ja/guide/tutorials/',
    collapsed: false,
    items: [
      { text: 'APIテストパイプライン', link: 'api-testing-pipeline' },
      { text: '初めての監視システム構築', link: 'first-monitoring-system' },
      { text: 'マルチ環境デプロイテスト', link: 'multi-environment-testing' }
    ]
  },
]

const reference = [
  {
    text: 'アクション',
    base: '/ja/reference/actions/',
    collapsed: false,
    items: [
      { text: '変数', link: 'variables' },
      { text: 'HTTP', link: 'http' },
      { text: 'SMTP', link: 'smtp' },
      { text: 'DB', link: 'db' },
      { text: 'SHELL', link: 'shell' },
      { text: 'BROWSER', link: 'browser' },
      { text: 'EMBEDDED', link: 'embedded' },
    ]
  },
  {
    text: '組み込み関数',
    base: '/ja/reference/functions/',
    collapsed: false,
    items: [
      { text: '利用可能なフィールドと構文', link: 'available-fields-and-syntax' },
      { text: '文字列関数', link: 'string' },
      { text: '日時関数', link: 'datetime' },
      { text: 'エンコーディング関数', link: 'encoding' },
      { text: '数学関数', link: 'mathematics' },
      { text: 'ユーティリティ関数', link: 'utility' },
      { text: 'JSON関数', link: 'json' },
      { text: '高度な関数使用法', link: 'advanced-usage' },
    ]
  },
  {
    text: 'その他',
    base: '/ja/reference/',
    collapsed: false,
    items: [
      { text: '環境変数', link: 'environment-variables' },
      { text: 'CLI', link: 'cli-reference' },
      { text: 'YAMLフィールド', link: 'yaml-configuration' }
    ]
  },
]

export const jaNavSidebar = {
  nav: [
    { text: 'ガイド', link: '/ja/guide/introduction/what-is-probe' },
    { text: 'リファレンス', link: '/ja/reference/actions/variables' },
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
