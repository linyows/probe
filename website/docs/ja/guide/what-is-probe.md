Probeは何ですか？
==

Probeは、YAMLでワークフローを宣言的に定義する無料のオープンソースソフトウェアです。
情報処理における実験やREST APIのEnd-to-Endテスト、ウェブサイトの監視、繰り返し行うタスクの自動化など、様々なワークフローのために設計されています。
ワークフローは、プラグインベースのアクションを使用して実行され、高い柔軟性と拡張性を提供します。
もし、ビルトインのアクションでは不十分な場合、プラグインを書くことで要件を満たすことができます。

ワークフローの例
--

例えば、次のワークフローのYAMLファイルを作成します。`hello world !`を出力するス
テップだけのジョブです。 このステップは、`wait: 1s` により、1秒待って出力されます。

```sh
cat <<EOF > example.yml
name: Workflow Example

jobs:
- name: My first job
  steps:
  - name: Hello
    uses: hello
    wait: 1s
    echo: hello world!
EOF
```

Probeの引数にファイル名を渡して実行すると次のようになります。

```sh
$ probe example.yml
Workflow Example

⏺ My first job (Completed in 1.04s)
  ⎿  0. ▲  🕐︎1s → Hello
           hello world!

Total workflow time: 1.07s ✔︎ All jobs succeeded
```

Probeは、簡単で迅速かつ楽しくワークフローを作って実行することができます。

ビルトインアクション
--

ビルトインのアクションは次の通りです。未実装は今後実装予定です。

- HTTP Action
- SMTP Action
- Hello Action
- ~~Shell Action~~ 未実装
- ~~SSH Action~~ 未実装
- ~~GRPC Action~~ 未実装
- ~~FTP Action~~ 未実装
- ~~IMAP Action~~ 未実装
- ~~Database Action~~ 未実装

困ったら？
--

フィードバック、アイデアの議論、質問への回答には[Github Discussions](https://github.com/linyows/probe/discussions)を利用しています。
ぜひGithub Discussionsにアクセスして、お気軽にディスカッションを始めてください。

サポートしてください
--

このプロジェクトに「いいね！」をお願いします！Githubでスターを付けて、Xでツイートしてください！
みなさんのご支援は開発者にとって大きな力となります。

コントリビュータ
--

オープンソースや素晴らしい大義に貢献したいと思っていたなら、今がチャンスです!

<a href="https://github.com/linyows/probe/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=linyows/probe" />
</a>
