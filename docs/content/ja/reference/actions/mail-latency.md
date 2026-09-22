# Mail Latencyアクション

`mail-latency`アクションはMaildir形式のディレクトリからメールを読み、`Received`ヘッダーから配送にかかった時間を求めてCSVファイルに書き出します。

## 基本的な構文

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

| パラメータ | 型 | 必須 | 説明 |
|---|---|---|---|
| `mail_dir` | String | 必須 | 計測対象のメールが置かれたディレクトリ |
| `output_dir` | String | 必須 | CSVを書き出すディレクトリ |

## レスポンスオブジェクト

| プロパティ | 型 | 説明 |
|---|---|---|
| `res.output_file` | String | 書き出したCSVのパス。ファイル名は`mail-latency.<timestamp>.csv` |
| `res.status` | Integer | 成功時は`0` |
| `rt` | Object | 計測に要した時間 |

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
