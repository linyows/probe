# Mail Latencyアクション

`mail-latency`アクションはMaildir形式のディレクトリからメールを読み、`Received`ヘッダーから配送にかかった時間を求めてCSVファイルに書き出します。

## 基本的な構文

このアクションはディレクトリ内のメールを読み、計測結果をCSVとして書き出します。

```yaml
steps:
  - name: Measure delivery latency
    uses: mail-latency
    with:
      mail_dir: "/var/mail/probe/new"
      output_dir: "./reports"
    test: res.status == 0
```

## パラメータ

読み込み元と書き出し先の2つのディレクトリが必要です。

| パラメータ | 型 | 必須 | 説明 |
|---|---|---|---|
| `mail_dir` | String | 必須 | 計測対象のメールが置かれたディレクトリ |
| `output_dir` | String | 必須 | CSVを書き出すディレクトリ |

`mail_dir`はサブディレクトリも含めて再帰的に読みます。計測するのは、配送されたメールの多くがそうであるように、`Return-Path:`か`Delivered-To:`のヘッダーで始まるファイルです。それ以外のファイルは無視します。読むのは最初の空行までのヘッダーだけで、ヘッダーは複数行に折り返されていてもいなくても構いません。計測するメールに`Date`ヘッダーか`Received`ヘッダーがないと、アクションは失敗し、CSVを書き出しません。

CSVにはメール1通につき1行を書き出します。列は以下のとおりです。

- `Sent Time Offset (sec)`と`Received Time Offset (sec)`: ディレクトリ内で最も早い`Date`を起点とした、送信と受信の時刻
- `Sent Time`: `Date`ヘッダーの時刻
- `Last Received Time`と`First Received Time`: 一番上と一番下の`Received`ヘッダーの時刻
- `End-to-End Latency (sec)`: `Date`から一番上の`Received`までの時間
- `Relay Latency (sec)`: 一番下の`Received`から一番上の`Received`までの時間
- `Return Path`と`File Path`: エンベロープの送信者とファイル名

時刻は、probeを実行しているマシンのローカルタイムゾーンで`2006-01-02 15:04:05`の形式で書き出します。

## レスポンスオブジェクト

結果には、書き出したファイルの場所が入ります。

| プロパティ | 型 | 説明 |
|---|---|---|
| `res.output_file` | String | 書き出したCSVのパス。ファイル名は`mail-latency.<timestamp>.csv`。前の実行と同じ秒に実行した場合は、上書きせずに`mail-latency.<timestamp>-2.csv`のように番号を付ける |
| `res.status` | Integer | 成功時は`0` |
| `rt.duration` | String | 計測に要した時間（例: `"1.2ms"`） |
| `rt.sec` | Float | 計測に要した時間（秒） |
| `status` | Integer | 成功時は`0` |

## 使用例

SMTPで送ったメールが届くのを待ってから計測する例です。

```yaml
jobs:
  - name: Mail delivery latency
    steps:
      - name: Send a probe mail
        uses: smtp
        with:
          addr: "{{vars.smtp_addr}}"
          from: "probe@example.com"
          to: "probe@example.com"
          subject: "probe {{vars.run_id}}"
          session: 1
          message: 1
          length: 500
        test: res.code == 0 && res.sent > 0

      - name: Measure
        id: latency
        uses: mail-latency
        wait: 30s
        with:
          mail_dir: "{{vars.maildir}}/new"
          output_dir: "./reports"
        test: res.status == 0
        outputs:
          report: res.output_file

      - name: Report
        uses: hello
        echo: "CSV: {{outputs.latency.report}}"
```

## 関連項目

- **[SMTP](/ja/reference/actions/smtp)** - メールの送信
- **[IMAP](/ja/reference/actions/imap)** - メールボックスの操作
- **[YAML設定](/ja/reference/yaml-configuration)** - ステップのプロパティ
