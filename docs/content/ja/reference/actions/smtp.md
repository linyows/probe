# SMTPアクション

`smtp`アクションはSMTPサーバーへメールを配送します。任意の本文を書いて通知を送るためのものではなく、配送そのものを計測・負荷試験するためのアクションです。本文は自動生成され、サイズは`length`で指定します。

## 基本的な構文

SMTPのステップでは、サーバー、エンベロープのアドレス、送信内容を指定します。

```yaml
steps:
  - name: Send a probe mail
    uses: smtp
    with:
      addr: "localhost:2525"
      from: "sender@example.com"
      to: "recipient@example.com"
      subject: "Delivery probe"
      session: 1
      message: 1
      length: 500
    test: res.code == 0 && res.sent > 0
```

## パラメータ

以下のフィールドで配送内容を指定します。いずれもテンプレート式を書けます。

| パラメータ | 型 | 必須 | デフォルト | 説明 |
|---|---|---|---|---|
| `addr` | String | 必須 | - | SMTPサーバー（`host:port`） |
| `from` | String | 必須 | - | エンベロープの送信者 |
| `to` | String | 必須 | - | エンベロープの受信者 |
| `subject` | String | 任意 | `""` | 件名 |
| `myhostname` | String | 任意 | - | `HELO` / `EHLO`で名乗るホスト名 |
| `session` | Integer | 任意 | `1` | 開くSMTPセッション数 |
| `message` | Integer | 任意 | `1` | 1セッションあたりの送信通数 |
| `length` | Integer | 任意 | `0` | 生成する本文のバイト数 |

認証、TLS、CC/BCC、任意の本文やHTMLを指定するパラメータはありません。実行結果にレポートを出したい場合はステップの`echo`を使います。

## レスポンスオブジェクト

実行後、`res`に配送の結果が入ります。

| フィールド | 型 | 説明 |
|---|---|---|
| `res.code` | Integer | すべて配送できたとき`0` |
| `res.sent` | Integer | 配送できた通数 |
| `res.failed` | Integer | 失敗した通数 |
| `res.total` | Integer | 試行した通数 |
| `res.error` | String | 失敗時のエラーメッセージ |
| `res.maildata` | String | 生成したメール（テキストの場合） |
| `res.filepath` | String | 生成したメールのパス（バイナリの場合） |

## 使用例

以下の例では、生成したメッセージを送信し、その件数を後続のステップから参照します。

### 複数セッション・複数通の配送

`session`と`message`は掛け合わせになり、各セッションでその通数を配送します。

```yaml
steps:
  - name: Deliver 3 messages over 2 sessions
    id: bulk
    uses: smtp
    with:
      addr: "{{vars.smtp_addr}}"
      from: "{{vars.from_addr}}"
      to: "{{vars.to_addr}}"
      subject: "Bulk delivery test"
      myhostname: probe-client.local
      session: 2
      message: 3
      length: 750
    test: res.code == 0 && res.sent == 6
    outputs:
      sent: res.sent
```

### 結果をレポートする

出力として取り出した件数は、後続のステップから出力できます。

```yaml
  - name: Delivery summary
    uses: hello
    echo: |
      Sent: {{outputs.bulk.sent}}
      Round trip: {{rt.duration}}
```

## 関連項目

- **[Mail Latency](/ja/reference/actions/mail-latency)** - 配送遅延の計測
- **[IMAP](/ja/reference/actions/imap)** - メールボックスの操作
- **[YAML設定](/ja/reference/yaml-configuration)** - ステップのプロパティ
