export type Locale = 'en' | 'ja'

type Part = { name: string; description: string; code: string }

export type HomeCopy = {
  headline: string[]
  lede: string
  primaryCta: { label: string; href: string }
  secondaryCta: { label: string; href: string }
  heroWorkflowLabel: string
  exampleHeading: string
  heroDagAlt: string
  flowNote: string
  heroReportLabel: string
  channels: {
    heading: string
    body: string
    chart: {
      title: string
      runner: string
      runnerNote: string
      builtInLabel: string
      externalLabel: string
      externalNames: string[]
      transport: string
    }
    link: { label: string; href: string }
  }
  graph: {
    heading: string
    body: string
    chart: {
      title: string
      workflowLabel: string
      jobLabel: string
      stepLabel: string
      needsLabel: string
    }
    link: { label: string; href: string }
  }
  parts: { heading: string; body: string; items: Part[] }
  install: { heading: string; body: string; sourceLabel: string; docsLink: string; docsHref: string }
  copy: { idle: string; done: string }
}

export const homeCopy: Record<Locale, HomeCopy> = {
  en: {
    headline: ['Complex scenarios, as one workflow.'],
    lede: 'Probe runs workflows defined in YAML. The definition looks like GitHub Actions, and the actions come built in: HTTP, DB, Shell, SSH, gRPC, SMTP, IMAP, Browser. The result of an action can be checked, so the same file serves for testing and monitoring too.',
    primaryCta: { label: 'Install Probe', href: '/guide/introduction/installation' },
    secondaryCta: { label: 'Read the quickstart', href: '/guide/introduction/quickstart' },
    heroWorkflowLabel: 'signup-flow.yml',
    exampleHeading: 'An example sign-up scenario',
    heroDagAlt: 'Create account runs first; Database and Mailbox both follow it and run in parallel; Activate waits for both.',
    flowNote: 'Create account posts the form and publishes the id that comes back. Database and Mailbox both depend on it, so Probe runs them at the same time — one asks SQLite for the row, the other opens the mailbox over IMAP. Activate waits for both before following the confirmation link.',
    heroReportLabel: '$ probe signup-flow.yml',
    channels: {
      heading: 'Supported actions',
      body: 'A step names an action in uses, and Probe starts that action as its own process and talks to it over gRPC. The built-in actions go through exactly the same handshake an out-of-tree plugin does, so none of them is a special case. What Probe cannot reach yet, you can add.',
      chart: {
        title: 'Probe talks to every action over gRPC, whether it is built in or your own.',
        runner: 'probe',
        runnerNote: 'workflow runner',
        builtInLabel: 'Built in',
        externalLabel: 'Yours',
        externalNames: ['graphql', 'jmap'],
        transport: 'gRPC',
      },
      link: { label: 'Read the actions reference', href: '/reference/actions-reference' },
    },
    graph: {
      heading: 'What a workflow is made of',
      body: 'A workflow is a set of jobs, and a job is a sequence of steps. Steps run one after another, in the order you wrote them. Jobs run in parallel by default; only the ones with needs wait for what they depend on. Add repeat to a job and the same file runs on an interval and reports a success rate.',
      chart: {
        title: 'A workflow holds jobs; a job holds steps that run in order. Two jobs run side by side, and a third waits for both through needs.',
        workflowLabel: 'Workflow',
        jobLabel: 'Job',
        stepLabel: 'Step',
        needsLabel: 'needs',
      },
      link: { label: 'Read the jobs and steps guide', href: '/guide/concepts/jobs-and-steps' },
    },
    parts: {
      heading: 'What a step can say',
      body: 'Whatever action a step names, these fields are available to it.',
      items: [
        {
          name: 'test',
          description: 'A step passes when the expression is true. Everything about the response is in scope.',
          code: 'test: res.code == 200 && res.body.status == "ok"',
        },
        {
          name: 'outputs',
          description: 'Name a value once, then read it from any later step or job.',
          code: 'outputs:\n  token: res.body.access_token',
        },
        {
          name: 'iteration',
          description: 'Run one step once per set of variables, without copying it.',
          code: 'iteration:\n- {name: Alice, role: admin}\n- {name: Bob, role: user}',
        },
        {
          name: 'retry',
          description: 'Give a flaky step another go, after a pause.',
          code: 'retry:\n  max_attempts: 3\n  interval: 5s',
        },
        {
          name: 'skipif',
          description: 'Leave the step out when the condition holds.',
          code: 'skipif: vars.env == "local"',
        },
        {
          name: 'wait',
          description: 'Pause before the step runs, for whatever has to settle first.',
          code: 'wait: 5s',
        },
      ],
    },
    install: {
      heading: 'Install',
      body: 'Probe is a single Go binary with no runtime dependencies.',
      sourceLabel: 'Or build from source',
      docsLink: 'Read the installation guide',
      docsHref: '/guide/introduction/installation',
    },
    copy: { idle: 'Copy', done: 'Copied' },
  },
  ja: {
    headline: ['複雑なシナリオを', 'ひとつのWorkflowに'],
    lede: 'ProbeはYAMLに定義したWorkflowを実行するソフトウェアです。GitHub ActionsのようなYAML定義と、HTTP、DB、Shell、SSH、gRPC、SMTP、IMAP、Browserといった多くのアクションがビルトインされています。アクションの結果を検証できるので、テストや監視といった目的にも使えます。',
    primaryCta: { label: 'Probeをインストール', href: '/ja/guide/introduction/installation' },
    secondaryCta: { label: 'クイックスタートを読む', href: '/ja/guide/introduction/quickstart' },
    heroWorkflowLabel: 'signup-flow.yml',
    exampleHeading: 'サインアップのシナリオの例',
    heroDagAlt: 'Create accountが先に走り、DatabaseとMailboxがそれに続いて並列に走り、Activateが両方の完了を待つ。',
    flowNote: '最初にCreate accountがフォームを送信し、返ってきたidを後続へ渡します。これを待つのがDatabaseとMailboxで、この2つは互いに依存しないため同時に走ります。DatabaseはSQLiteにレコードの状態を問い合わせ、MailboxはIMAPでメールボックスを開いて確認メールを探します。Activateはその両方が終わるのを待ってから、確認リンクをたどります。',
    heroReportLabel: '$ probe signup-flow.yml',
    channels: {
      heading: 'サポートしているアクション',
      body: 'Stepのusesにアクション名を書くと、Probeはそのアクションを別プロセスとして起動し、gRPC越しにやり取りします。ビルトインのアクションも、外部のプラグインとまったく同じ手順で接続されます。特別扱いされているものは1つもないので、必要なアクションが無ければ自分で追加できます。',
      chart: {
        title: 'Probeはビルトインのアクションも自作のプラグインも、同じgRPCでつないでいる。',
        runner: 'probe',
        runnerNote: 'ワークフロー実行',
        builtInLabel: 'ビルトイン',
        externalLabel: '自作のプラグイン',
        externalNames: ['graphql', 'jmap'],
        transport: 'gRPC',
      },
      link: { label: 'アクションリファレンスを読む', href: '/ja/reference/actions/variables' },
    },
    graph: {
      heading: 'Workflowの構成要素',
      body: 'WorkflowはJobの集まりで、JobはStepの並びです。Stepは書いた順に1つずつ走ります。Jobは既定で並列に走り、needsを書いたものだけが依存先の完了を待ちます。Jobにrepeatを足せば、同じファイルが一定間隔で走り、成功率を返します。',
      chart: {
        title: 'WorkflowがJobを持ち、JobがStepを順に持つ。2つのJobは並んで走り、3つ目がneedsで両方の完了を待つ。',
        workflowLabel: 'Workflow',
        jobLabel: 'Job',
        stepLabel: 'Step',
        needsLabel: 'needs',
      },
      link: { label: 'ジョブとステップを読む', href: '/ja/guide/concepts/jobs-and-steps' },
    },
    parts: {
      heading: 'Stepで使用できる便利な機能',
      body: 'Stepにはアクションの設定以外に、以下の共通機能があります。',
      items: [
        {
          name: 'test',
          description: '式が真ならそのStepは成功です。レスポンスの中身はすべて参照できます。',
          code: 'test: res.code == 200 && res.body.status == "ok"',
        },
        {
          name: 'outputs',
          description: '値に名前をつけておくと、あとのStepやJobから読めます。',
          code: 'outputs:\n  token: res.body.access_token',
        },
        {
          name: 'iteration',
          description: '変数の組ごとに、同じStepをコピーせず繰り返します。',
          code: 'iteration:\n- {name: Alice, role: admin}\n- {name: Bob, role: user}',
        },
        {
          name: 'retry',
          description: '失敗したときに、間隔をあけてやり直します。',
          code: 'retry:\n  max_attempts: 3\n  interval: 5s',
        },
        {
          name: 'skipif',
          description: '条件に当てはまるときは、そのStepを飛ばします。',
          code: 'skipif: vars.env == "local"',
        },
        {
          name: 'wait',
          description: '先に落ち着くのを待ちたいとき、実行前に待機します。',
          code: 'wait: 5s',
        },
      ],
    },
    install: {
      heading: 'インストール',
      body: 'Probeは実行時の依存がない、単体のGoバイナリです。',
      sourceLabel: 'ソースからビルドする場合',
      docsLink: 'インストールガイドを読む',
      docsHref: '/ja/guide/introduction/installation',
    },
    copy: { idle: 'コピー', done: 'コピーしました' },
  },
}
