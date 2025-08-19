# IMAP Action

IMAPアクションは、IMAPサーバーに接続してメールの読み取り、検索、メールボックス管理などのメール操作を実行します。

## 基本構文

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

### サポートされているコマンド

- **select**: 読み書き操作用のメールボックス選択
- **examine**: 読み取り専用でのメールボックス選択  
- **search**: 条件によるメッセージ検索
- **uid search**: UIDを使用したメッセージ検索
- **list**: 使用可能なメールボックスの一覧表示
- **fetch**: メッセージデータの取得
- **uid fetch**: UIDを使用したメッセージデータの取得
- **create**: メールボックスの作成
- **delete**: メールボックスの削除
- **rename**: メールボックス名の変更
- **subscribe**: メールボックスの購読
- **unsubscribe**: メールボックスの購読解除
- **noop**: 操作なし（キープアライブ）

### 検索条件

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

### Gmail設定

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
