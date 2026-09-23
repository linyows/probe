<p align="right"><a href="https://github.com/linyows/probe/blob/main/README.md">English</a> | 日本語</p>

<br><br><br><br>

<p align="center">
  <img alt="PROBE" src="https://github.com/linyows/probe/blob/main/misc/probe.svg" width="200">
</p>

<br><br><br><br>

<p align="center">
  <strong>Probe</strong>は、テスト、監視、自動化タスクのために設計された強力なYAMLベースのワークフロー自動化ツールです。
</p>

<p align="center">
  <a href="https://github.com/linyows/probe/actions/workflows/build.yml">
    <img alt="GitHub Workflow Status" src="https://img.shields.io/github/actions/workflow/status/linyows/probe/build.yml?branch=main&style=for-the-badge&labelColor=666666">
  </a>
  <a href="https://github.com/linyows/probe/releases">
    <img src="http://img.shields.io/github/release/linyows/probe.svg?style=for-the-badge&labelColor=666666&color=DDDDDD" alt="GitHub Release">
  </a>
  <a href="http://godoc.org/github.com/linyows/probe">
    <img src="http://img.shields.io/badge/go-docs-blue.svg?style=for-the-badge&labelColor=666666&color=DDDDDD" alt="Go Documentation">
  </a>
</p>

単体のGoバイナリで、別途用意するランタイムはありません。そのため同じファイルを手元でも、CIでも、cronからでも実行できます。実行結果は終了コードに反映され、`repeat`を加えればテストとして書いたファイルがそのまま監視になります。Probeはプラグインベースのアクションでワークフローを実行するため、柔軟に拡張できます。ドキュメント: [probe.linyo.ws/ja](https://probe.linyo.ws/ja)

![Architecture](/misc/probe-architecture.svg)

特徴
----

類似のソフトウェアは、テストを実行するものか、監視のために繰り返し確認するもののどちらかであることがほとんどです。扱えるプロトコルも1つに限られる場合が多くあります。Probeはその両方を、Webシステムが実際に使っているプロトコルの範囲で扱います。

- **1つのファイルでシステム全体を対象にできる**: HTTP、gRPC、MySQL、PostgreSQL、SQLite、SMTP、IMAP、SSH、シェル、実際のブラウザが組み込みです。エンドポイントを呼び、書き込まれた行を問い合わせ、送信されたメールを読むところまでを、3つのツールをつなぎ合わせずに1回の実行で行えます。
- **共通処理を別のファイルに置ける**: シナリオの多くはログインから始まり、2つ目以降のシナリオでも同じ手順を書くことになります。その手順を、接続先を`defaults`に書いたジョブファイルに置けば、どのワークフローからも`uses: embedded`で実行できます。呼び出し側は認証情報を`vars`で渡し、トークンを`outputs`から受け取ります。ジョブのステップは、レポートでも呼び出し元のステップの下に入れ子で表示されます。
- **同じファイルがテストにも監視にもなる**: ジョブに`repeat`を加えると一定の間隔で繰り返し、`3/3 success (100.0%)`のように結果を報告します。監視用に書き直した2つ目のワークフローは要りません。
- **ジョブは一覧ではなくグラフ**: `needs`で宣言するのは順序が必要な箇所だけで、残りは並行して実行されます。そのグラフは`probe dag`でASCIIまたはMermaidとして出力できます。シナリオを実行するツールはファイルを上から下へ順にたどります。
- **どこで動かしても同じ**: 単体のGoバイナリで、併せて入れるものも、常駐させるサービスもありません。手元の端末でも、CIのステップでも、cronからでも同じ動作になります。
- **アクションはプラグイン**: 各アクションは[go-plugin](https://github.com/hashicorp/go-plugin)越しの別プロセスです。組み込みにないプロトコルは、バイナリをフォークせずに追加できます。

このほかワークフローには、ステップやジョブの間でデータを渡す`outputs`、任意の応答を検証する`test`、`retry`、`skipif`、`iteration`、`wait`、`timeout`、ステップ間で共通する設定をまとめる`defaults`、そしてコマンドラインで複数のYAMLをマージする機能があります。

これらの関係は[Probeの理解](https://probe.linyo.ws/ja/guide/introduction/understanding-probe)で、k6、Hurl、Venom、runn、Blackbox exporter、GitHub Actionsのいずれを選んだほうがよいかは[比較](https://probe.linyo.ws/ja/guide/introduction/comparison)で説明しています。

クイックスタート
----------------

バイナリをインストールします。

```bash
go install github.com/linyows/probe/cmd/probe@latest
```

ワークフローを書きます。

```yaml
# health-check.yml
name: API Health Check
jobs:
- name: Check API Status
  steps:
  - name: Ping API
    uses: http
    with:
      url: https://api.example.com
      get: /health
    test: res.code == 200
```

実行します。

```bash
probe health-check.yml
```

```
API Health Check

⏺ Check API Status (Completed in 0.02s)
  ⎿ 0. ✓  Ping API

Total workflow time: 0.02s ✓ All jobs succeeded
```

[クイックスタート](https://probe.linyo.ws/ja/guide/introduction/quickstart)では同じ手順を順に追い、[最初のワークフロー](https://probe.linyo.ws/ja/guide/introduction/your-first-workflow)では動かし続けられる形まで広げます。

組み込みアクション
------------------

| アクション | 用途 |
|---|---|
| [`http`](https://probe.linyo.ws/ja/reference/actions/http) | HTTPリクエストとその応答 |
| [`db`](https://probe.linyo.ws/ja/reference/actions/db) | MySQL、PostgreSQL、SQLiteへのクエリ |
| [`smtp`](https://probe.linyo.ws/ja/reference/actions/smtp) | メールの送信 |
| [`imap`](https://probe.linyo.ws/ja/reference/actions/imap) | メールボックスの読み取り |
| [`mail-latency`](https://probe.linyo.ws/ja/reference/actions/mail-latency) | 受信したメールからの配送遅延の計測 |
| [`ssh`](https://probe.linyo.ws/ja/reference/actions/ssh) | リモートホストでのコマンド実行 |
| [`shell`](https://probe.linyo.ws/ja/reference/actions/shell) | Probeを実行しているマシンでのコマンド実行 |
| [`browser`](https://probe.linyo.ws/ja/reference/actions/browser) | 実際のブラウザの操作 |
| [`grpc`](https://probe.linyo.ws/ja/reference/actions/grpc) | gRPCの呼び出し |
| [`embedded`](https://probe.linyo.ws/ja/reference/actions/embedded) | ステップから別のジョブファイルを実行 |
| [`hello`](https://probe.linyo.ws/ja/reference/actions/hello) | レポートへの出力 |

ドキュメント
------------

- [ガイド](https://probe.linyo.ws/ja/guide): インストール、CLI、そしてワークフロー、ジョブ、ステップ、データフローの考え方
- [How-to](https://probe.linyo.ws/ja/guide/how-tos/api-testing): APIテスト、監視、パフォーマンス、環境管理、エラーハンドリング
- [チュートリアル](https://probe.linyo.ws/ja/guide/tutorials/first-monitoring-system): 監視システム、APIテストパイプライン、マルチ環境テストの構築
- [リファレンス](https://probe.linyo.ws/ja/reference): YAMLの設定、CLIのオプション、組み込み関数、環境変数

実行できるワークフローは[`examples/`](./examples/)にあります。

コントリビューション
--------------------

Issue、機能の要望、プルリクエストのいずれも歓迎します。

ライセンス
----------

MITです。[LICENSE](./LICENSE)を参照してください。

作者
----

[linyows](https://github.com/linyows)
