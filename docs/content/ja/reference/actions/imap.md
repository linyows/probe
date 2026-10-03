# IMAP Action

IMAPアクションは、IMAPサーバーに接続してメールの読み取り、検索、メールボックス管理などのメール操作を実行します。

## 基本構文

IMAPのステップでは接続を確立し、`commands`に並べたコマンドを順に実行します。

```yaml
vars:
  imap_username: "{{IMAP_USERNAME}}"
  imap_password: "{{IMAP_PASSWORD}}"

steps:
  - name: "メールチェック"
    uses: imap
    with:
      host: "imap.example.com"
      port: 993
      username: "{{vars.imap_username}}"
      password: "{{vars.imap_password}}"
      tls: true
      commands:
      - name: "select"
        mailbox: "INBOX"
      - name: "search"
        criteria:
          flags: ["unseen"]
    test: res.code == 0
```

## パラメータ

パラメータでは、接続先のサーバー、提示する認証情報、接続の保護方法、そして接続後に実行するIMAPコマンドの一覧を指定します。

### `host` (必須)

**型:** String  
**説明:** IMAPサーバーのホスト名またはIPアドレス  
**テンプレート式をサポート:** はい

```yaml
with:
  host: "imap.gmail.com"
  host: "imap.example.com" 
  host: "{{vars.imap_server}}"
```

### `port` (オプション)

**型:** Integer  
**デフォルト:** `993`  
**説明:** IMAPサーバーのポート

```yaml
with:
  port: 993   # IMAPS (SSL/TLS)
  port: 143   # IMAP (プレーンまたはSTARTTLS)
```

### `username` (必須)

**型:** String  
**説明:** IMAP認証のユーザー名  
**テンプレート式をサポート:** はい

```yaml
vars:
  email_user: "{{EMAIL_USER}}"

with:
  username: "{{vars.email_user}}"
  username: "user@example.com"
```

### `password` (必須)

**型:** String  
**説明:** IMAP認証のパスワード  
**テンプレート式をサポート:** はい

```yaml
vars:
  email_password: "{{EMAIL_PASSWORD}}"
  app_password: "{{EMAIL_APP_PASSWORD}}"

with:
  password: "{{vars.email_password}}"
  password: "{{vars.app_password}}"
```

### `tls` (オプション)

**型:** Boolean
**デフォルト:** `true`
**説明:** TLS/SSLで暗号化して接続するかどうか

```yaml
with:
  host: "imap.example.com"
  port: 993
  tls: true     # TLSを使う（推奨）
```

### `insecure_skip_tls` (オプション)

**型:** Boolean
**デフォルト:** `false`
**説明:** サーバーの証明書を検証せずに受け入れる

`tls: true`では証明書をシステムが信頼する認証局で検証します。そのため、ローカルやステージングのメールサーバーのように自己署名証明書を使うサーバーでは`x509: certificate signed by unknown authority`で失敗します。`insecure_skip_tls: true`にすると、それでも接続します。通信は暗号化されたままですが、なりすましへの防御はなくなるため、自分で管理するサーバーにだけ使います。

```yaml
with:
  host: "mail.staging.internal"
  port: 993
  tls: true
  insecure_skip_tls: true
```

### `timeout` (オプション)

**型:** 期間の文字列、または秒数
**デフォルト:** `30s`
**説明:** セッション全体の制限時間。接続、TLSのハンドシェイク、ログイン、すべてのコマンド、ログアウトを含む

```yaml
with:
  timeout: "60s"   # または 60
```

時間内に応答しないサーバーとの接続は打ち切られ、ステップはアクションのエラーとして失敗し、実行は終了ステータス`3`で終わります。メッセージは`IMAP session timed out after 30s while running commands`のようになります。サーバーがコマンドを拒否した場合は`res.code`が`1`になるのと異なります。期間でも数値でもない値や、0以下の値はエラーになります。

### `commands` (必須)

**型:** コマンドオブジェクトの配列  
**説明:** 順次実行するIMAPコマンド

```yaml
with:
  commands:
  - name: "select"
    mailbox: "INBOX"
  - name: "search"
    criteria:
      since: "today"
  - name: "fetch"
    sequence: "1:5"
    dataitem: "ALL"
```

## IMAPコマンド

`commands`の各要素は1つのIMAP操作を表し、その操作に必要なフィールドを持ちます。ここでは対応しているコマンドと、検索で指定できる条件を示します。

### サポートされているコマンド

- **select**: 読み書き操作用のメールボックス選択
- **examine**: 読み取り専用でのメールボックス選択  
- **search**: 条件によるメッセージ検索
- **uid search**: UIDを使用したメッセージ検索
- **list**: 使用可能なメールボックスの一覧表示
- **fetch**: メッセージデータの取得
- **uid fetch**: UIDを使用したメッセージデータの取得
- **store**: メッセージのフラグの変更
- **uid store**: UIDを使用したメッセージのフラグの変更
- **copy**: メッセージを別のメールボックスへコピー
- **uid copy**: UIDを使用したメッセージのコピー
- **create**: メールボックスの作成
- **delete**: メールボックスの削除
- **rename**: メールボックス名の変更
- **subscribe**: メールボックスの購読
- **unsubscribe**: メールボックスの購読解除
- **noop**: 操作なし（キープアライブ）

#### storeとuid store

メッセージのフラグを設定、追加、削除します。`dataitem`は、置き換えるなら`FLAGS`、追加するなら`+FLAGS`、削除するなら`-FLAGS`です。末尾に`.SILENT`を付けると、サーバーは新しいフラグを返しません。`value`にはフラグを括弧付きか括弧なしで並べます。`FLAGS`に空の値か`()`を指定するとフラグをすべて消し、片方だけの括弧はエラーになります。`uid store`は`sequence`にUIDを取ります。`sequence`を省くと、`fetch`と同じく最新の検索の結果を対象にします。ただし種類が合っている場合に限ります。`store`には`search`、`uid store`には`uid search`が必要です。検索のたびに、種類を問わずそれ以前の検索結果は置き換わるため、古い検索が使われることはありません。

`sequence`の中で単独の`*`は最後のメッセージを指し、数値は`1`から`4294967295`でなければなりません。`0`やそれより大きい値は`*`とは解釈せずエラーにします。番号はメールボックスごとのものなので、`select`や`examine`を実行すると直前の検索結果は使われなくなります。メールボックスを切り替えたあとに`sequence`を省いたコマンドは、別のメッセージを対象にせずエラーになります。

```yaml
- name: "select"
  mailbox: "INBOX"
- name: "store"
  sequence: "1:3"
  dataitem: "+FLAGS"
  value: '\Seen \Flagged'
```

`res.data.store.count`は、サーバーが新しいフラグとともに返したメッセージの数で、`.SILENT`では`0`です。変更できるのはフラグだけで、ほかのデータ項目を指定するとコマンドは失敗します。

#### copyとuid copy

メッセージを、既にある別のメールボックスへコピーします。`uid copy`は`sequence`にUIDを取り、`sequence`を省くとどちらも直前の検索結果を対象にします。

```yaml
- name: "select"
  mailbox: "INBOX"
- name: "copy"
  sequence: "1:2"
  mailbox: "Archive"
```

`res.data.copy.count`は、サーバーが報告したときにコピーされたメッセージの数です。サーバーがUIDPLUSに対応していれば報告され、そうでなければ`0`です。

存在しないメールボックスへの`copy`のようにコマンドが失敗すると、ほかのIMAPコマンドと同じく`res.code`が`1`になり、`res.error`に理由が入ります。

### 検索条件

`search`の絞り込みは`criteria`で指定します。日付とフラグを組み合わせられます。

```yaml
criteria:
  since: "today"              # 日付ベースの検索
  flags: ["unseen"]           # フラグベースの検索
  headers:                    # ヘッダーベースの検索
    from: "sender@example.com"
    subject: "件名"
  bodies: ["重要"]            # 本文テキスト検索
  texts: ["会議"]             # 全文検索
```

## レスポンスオブジェクト

IMAPアクションは以下の構造を持つ`res`オブジェクトを提供します：

| プロパティ | 型 | 説明 |
|----------|------|-------------|
| `code` | Integer | 操作結果 (0 = 成功、非ゼロ = エラー) |
| `data` | Object | コマンドタイプ別に整理されたコマンド結果 |
| `error` | String | 操作が失敗した場合のエラーメッセージ |

## 使用例

ポート、TLS、認証情報はプロバイダによって異なります。以下ではGmailへの接続設定と、受信を監視するワークフローを示します。

### Gmail設定

Gmailではアカウントのパスワードではなくアプリパスワードを使い、993番ポートにTLSで接続します。

```yaml
vars:
  gmail_username: "{{GMAIL_USERNAME}}"
  gmail_app_password: "{{GMAIL_APP_PASSWORD}}"

steps:
  - name: "Gmailの受信箱をチェック"
    uses: imap
    with:
      host: "imap.gmail.com"
      port: 993
      username: "{{vars.gmail_username}}"
      password: "{{vars.gmail_app_password}}"  # アプリパスワードを使用
      tls: true
      commands:
      - name: "select"
        mailbox: "INBOX"
      - name: "search"
        criteria:
          flags: ["unseen"]
          since: "today"
      - name: "fetch"
        sequence: "*"
        dataitem: "ENVELOPE FLAGS"
    test: res.code == 0
    outputs:
      unread_count: res.data.search.count
      latest_sender: res.data.fetch.messages__0__from
```

### メール監視ワークフロー

定期的に受信箱を検索すれば、届くはずのメールが届いているかを監視できます。

```yaml
name: メールアラート監視
vars:
  monitor_email: "{{MONITOR_EMAIL}}"
  monitor_password: "{{MONITOR_PASSWORD}}"

jobs:
- name: 重要なアラートを監視
  repeat:
    count: 288     # 24時間、5分間隔で実行
    interval: "5m"
  steps:
  - name: "重要なアラートをチェック"
    id: check-alerts
    uses: imap
    with:
      host: "imap.example.com"
      username: "{{vars.monitor_email}}"
      password: "{{vars.monitor_password}}"
      commands:
      - name: "select"
        mailbox: "INBOX"
      - name: "search"
        criteria:
          headers:
            subject: "重要"
          flags: ["unseen"]
          since: "5分前"
    test: res.code == 0
    outputs:
      critical_count: res.data.search.count
      has_critical: res.data.search.count > 0
```

## セキュリティに関する考慮事項

IMAPアクションは以下のセキュリティ対策を実装しています：

- **TLS暗号化**: データ保護のためデフォルトでTLSを使用
- **証明書検証**: デフォルトでサーバー証明書を検証
- **認証情報の保護**: パスワードはログと出力でマスク化
- **接続タイムアウト**: ハング状態の接続を防止
- **認証**: 標準的なIMAP認証方式をサポート
