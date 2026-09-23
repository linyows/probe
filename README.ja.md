<p align="right"><a href="https://github.com/linyows/probe/blob/main/README.md">English</a> | 日本語</p>

<br><br><br><br><br><br>

<p align="center">
  <img alt="PROBE" src="https://github.com/linyows/probe/blob/main/misc/probe.svg" width="200">
</p>

<br><br><br><br><br><br>

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
  <a href="https://deepwiki.com/linyows/probe">
    <img src="http://img.shields.io/badge/deepwiki-docs-purple.svg?style=for-the-badge&labelColor=666666&color=DDDDDD" alt="Deepwiki Documentation">
  </a>
</p>

ProbeはYAMLで書いたワークフローを、HTTP API、データベース、メールサーバー、ブラウザ、シェルに対して実行し、応答を1つずつ検証して、読めるレポートを出力します。

単体のGoバイナリで、別途用意するランタイムはありません。そのため同じファイルを手元でも、CIでも、cronからでも実行できます。実行結果は終了コードに反映され、`repeat`を加えればテストとして書いたファイルがそのまま監視になります。

**ドキュメント: [probe.linyo.ws/ja](https://probe.linyo.ws/ja)**

![Architecture](/misc/probe-architecture.svg)

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

Probeでできること
-----------------

- **1つのファイルで複数のプロトコルを扱う**: HTTPのエンドポイントを呼び、その背後のデータベースに問い合わせ、送信されたメールを確認するところまでを1回の実行で行えます。
- **ジョブは並行して実行される**: 順序が必要な箇所だけ`needs`で宣言し、残りは同時に実行されます。できあがるグラフは`probe dag`で出力できます。
- **ステップ間でデータを渡せる**: ステップが公開した`outputs`を、後続のステップやジョブから参照できます。
- **すべてのステップが検証を持つ**: `test`は応答に対する式です。そのためワークフローは、たまたま成功するスクリプトではなくテストになります。
- **テストがそのまま監視になる**: `repeat`はジョブを一定の間隔で繰り返し、何回成功したかを報告します。
- **例外的な場面にも対応できる**: `retry`、`skipif`、`iteration`、`wait`、`timeout`で、一度で通らない場合、環境によっては実行しない場合、いつまでも待たせたくない場合を扱えます。
- **拡張できる**: アクションはgRPC経由で提供されるプラグインです。必要なアクションが組み込みにない場合は追加できます。

これらの関係は[Probeの理解](https://probe.linyo.ws/ja/guide/introduction/understanding-probe)で、他のツールを選んだほうがよい範囲は[比較](https://probe.linyo.ws/ja/guide/introduction/comparison)で説明しています。

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
