# ブラウザアクション

`browser`アクションはChromeDPを使用してWebブラウザを自動化し、テスト、スクレイピング、Webアプリケーションとの相互作用のための包括的なWeb自動化機能を提供します。

## 基本的な構文

ブラウザのステップでは、行う操作を`action`で指定し、その操作に必要な情報を与えます。

```yaml
steps:
  - name: "Navigate to Website"
    uses: browser
    with:
      action: navigate
      url: "https://example.com"
      headless: true
      timeout: 30s
    test: res.code == 0
```

## パラメータ

どのブラウザ操作を行うかは`action`で決まり、残りのパラメータはその操作に必要な情報、つまり対象の要素、入力する値、ブラウザの起動方法を与えます。

### `action` (必須)

**型:** String  
**説明:** 実行するブラウザアクション  
**値:** 
- **ナビゲーション:** `navigate`
- **テキスト/コンテンツ:** `text`, `value`, `get_html`
- **属性:** `get_attribute`
- **インタラクション:** `click`, `double_click`, `right_click`, `hover`, `focus`
- **入力:** `type`, `send_keys`, `select`
- **フォーム:** `submit`
- **スクロール:** `scroll`
- **スクリーンショット:** `screenshot`, `capture_screenshot`, `full_screenshot`
- **待機:** `wait_visible`, `wait_not_visible`, `wait_ready`, `wait_text`, `wait_enabled`

### `url` (オプション)

**型:** String  
**説明:** ナビゲートするURL（navigateアクションに必須）  
**サポート:** テンプレート式

```yaml
with:
  action: navigate
  url: "https://example.com"
  url: "{{vars.base_url}}/login"
```

### `selector` (オプション)

**型:** String  
**説明:** 要素をターゲットするためのCSSセレクタ  
**サポート:** テンプレート式

```yaml
with:
  action: get_text
  selector: "h1"
  selector: "#main-title"
  selector: ".article-content p:first-child"
```

### `value` (オプション)

**型:** String  
**説明:** タイプする値または待機するテキスト  
**サポート:** テンプレート式

```yaml
with:
  action: type
  selector: "#email"
  value: "user@example.com"
  value: "{{vars.username}}"
```

### `attribute` (オプション)

**型:** String  
**説明:** 取得する属性名（get_attributeアクションに必須）

```yaml
with:
  action: get_attribute
  selector: "a"
  attribute: "href"
```

### `headless` (オプション)

**型:** Boolean  
**デフォルト:** `true`  
**説明:** ヘッドレスモードでブラウザを実行するかどうか

```yaml
with:
  action: navigate
  url: "https://example.com"
  headless: false  # ブラウザウィンドウを表示
```

### `timeout` (オプション)

**型:** Duration  
**デフォルト:** `30s`  
**説明:** アクションタイムアウト

```yaml
with:
  action: wait_visible
  selector: ".loading"
  timeout: "60s"
```

## レスポンスオブジェクト

ブラウザアクションはアクション固有のプロパティを持つ`res`オブジェクトを提供します：

### 共通プロパティ

 | プロパティ | 型      | 説明                                    |
 | ---------- | ------  | -------------                           |
 | `code`     | Integer | 結果コード (0 = 成功, 非ゼロ = エラー)  |
 | `results`  | Object  | アクション固有の結果 (テキスト、値など) |

### ナビゲーションレスポンス

 | プロパティ | 型     | 説明                         |
 | ---------- | ------ | -------------                |
 | `url`      | String | ナビゲートしたURL            |
 | `time_ms`  | String | ナビゲーション時間（ミリ秒） |

### テキスト/属性レスポンス

 | プロパティ  | 型     | 説明                                    |
 | ----------  | ------ | -------------                           |
 | `selector`  | String | 使用されたCSSセレクタ                   |
 | `text`      | String | 抽出されたテキストコンテンツ (get_text) |
 | `attribute` | String | 属性名 (get_attribute)                  |
 | `value`     | String | 属性値 (get_attribute)                  |
 | `exists`    | String | 属性が存在する場合"true"                |

### スクリーンショットレスポンス

 | プロパティ   | 型     | 説明                                     |
 | ----------   | ------ | -------------                            |
 | `screenshot` | String | Base64エンコードされたスクリーンショット |
 | `size_bytes` | String | スクリーンショットサイズ（バイト）       |

## ブラウザアクション

`action`に指定できる値ごとに、参照するパラメータと返る内容を示します。

### URLへのナビゲート

`navigate`はページを開きます。他のブラウザ操作はすべてこのステップを前提にします。

```yaml
- name: "Open Website"
  uses: browser
  with:
    action: navigate
    url: "https://example.com"
    headless: true
  test: res.code == 0
  outputs:
    load_time: rt.sec * 1000
```

### テキストコンテンツの抽出

`text`はセレクタに一致した要素のテキストを読みます。読んだ値は検証にも、`outputs`での受け渡しにも使えます。

```yaml
- name: "Get Page Title"
  uses: browser
  with:
    action: text
    selector: "h1"
  test: res.code == 0 && res.results.text != ""
  outputs:
    page_title: res.results.text

- name: "Get Input Value"
  uses: browser
  with:
    action: value
    selector: "#username"
  test: res.code == 0
  outputs:
    current_username: res.results.value

- name: "Get Element HTML"
  uses: browser
  with:
    action: get_html
    selector: ".article-content"
  test: res.code == 0
  outputs:
    article_html: res.results.get_html
```

### 要素属性の取得

`get_attribute`は要素のテキストではなく、指定した属性の値を読みます。

```yaml
- name: "Extract Links"
  uses: browser
  with:
    action: get_attribute
    selector: "a.download-link"
    attribute: "href"
  test: res.code == 0 && res.exists == "true"
  outputs:
    download_url: res.results.value
```

### フォームインタラクション

フォームの操作は1ステップにつき1つです。フィールドへの入力、コントロールのクリック、送信と分けて書きます。

```yaml
# フォームフィールドの入力
- name: "Enter Email"
  uses: browser
  with:
    action: type
    selector: "#email"
    value: "user@example.com"
  test: res.code == 0

# ボタンのクリック
- name: "Click Submit"
  uses: browser
  with:
    action: click
    selector: "#submit-btn"
  test: res.code == 0

# フォームの送信
- name: "Submit Form"
  uses: browser
  with:
    action: submit
    selector: "form"
  test: res.code == 0
```

### 要素の待機

読み込み後に描画されるページでは、次のステップが要素を指定する前に明示的に待つ必要があります。

```yaml
# 要素の表示を待機
- name: "Wait for Results"
  uses: browser
  with:
    action: wait_visible
    selector: ".search-results"
    timeout: "10s"
  test: res.code == 0

# 特定のテキストを待機
- name: "Wait for Success Message"
  uses: browser
  with:
    action: wait_text
    selector: ".status"
    value: "Success"
  test: res.code == 0
```

### スクリーンショットの撮影

`screenshot`は、その時点でページがどう表示されていたかを記録します。

```yaml
- name: "Take Screenshot"
  uses: browser
  with:
    action: screenshot
  test: res.code == 0
  outputs:
    screenshot_data: res.screenshot
    screenshot_size: res.size_bytes
```

## 高度な使用例

1つのアクションだけで完結する場面はほとんどありません。以下のワークフローでは複数のステップをつなぎ、`outputs`で状態を引き継ぎます。

### ログインフロー

ログインは一連の手順になります。フォームを開き、認証情報を入力し、送信し、結果を確認します。

```yaml
vars:
  login_url: "{{LOGIN_URL}}"
  username: "{{USERNAME}}"
  password: "{{PASSWORD}}"

steps:
  - name: "Navigate to Login"
    uses: browser
    with:
      action: navigate
      url: "{{vars.login_url}}"
    test: res.code == 0

  - name: "Enter Username"
    uses: browser
    with:
      action: type
      selector: "#username"
      value: "{{vars.username}}"
    test: res.code == 0

  - name: "Enter Password"
    uses: browser
    with:
      action: type
      selector: "#password"
      value: "{{vars.password}}"
    test: res.code == 0

  - name: "Submit Login"
    uses: browser
    with:
      action: click
      selector: "#login-button"
    test: res.code == 0

  - name: "Wait for Dashboard"
    uses: browser
    with:
      action: wait_visible
      selector: ".dashboard"
      timeout: "15s"
    test: res.code == 0
```

### データ抽出

ページを開いて複数の要素を読めば、ワークフローの他の部分で使える値になります。

```yaml
steps:
  - name: "Navigate to Data Page"
    uses: browser
    with:
      action: navigate
      url: "https://example.com/data"
    test: res.code == 0

  - name: "Wait for Table"
    uses: browser
    with:
      action: wait_visible
      selector: "table"
    test: res.code == 0

  - name: "Count Rows"
    uses: browser
    with:
      action: get_elements
      selector: "table tr"
    test: res.code == 0 && res.count != "0"
    outputs:
      row_count: res.count

  - name: "Extract First Cell"
    uses: browser
    with:
      action: get_text
      selector: "table tr:first-child td:first-child"
    test: res.code == 0
    outputs:
      first_cell: res.results.text
```

### E2Eテスト

E2Eテストでは、利用者と同じ手順でアプリケーションを操作し、画面に表示された内容を検証します。

```yaml
steps:
  - name: "Load Application"
    uses: browser
    with:
      action: navigate
      url: "https://app.example.com"
    test: res.code == 0

  - name: "Fill Contact Form"
    uses: browser
    with:
      action: type
      selector: "#contact-name"
      value: "John Doe"
    test: res.code == 0

  - name: "Fill Email"
    uses: browser
    with:
      action: type
      selector: "#contact-email"
      value: "john@example.com"
    test: res.code == 0

  - name: "Fill Message"
    uses: browser
    with:
      action: type
      selector: "#contact-message"
      value: "Hello from automated test"
    test: res.code == 0

  - name: "Submit Form"
    uses: browser
    with:
      action: submit
      selector: "#contact-form"
    test: res.code == 0

  - name: "Verify Success"
    uses: browser
    with:
      action: wait_text
      selector: ".success-message"
      value: "Thank you"
      timeout: "10s"
    test: res.code == 0

  - name: "Take Success Screenshot"
    uses: browser
    with:
      action: screenshot
    test: res.code == 0
```

## エラーハンドリング

アクションが完了できないと、ステップ全体がアクションのエラーとして失敗し、実行は終了ステータス`3`で終わります。セレクタが何にも一致しなくても空の結果にはなりません。`wait_visible`、`text`、`click`など要素を探すアクションは、ステップの`timeout`まで待ち続けます。URLに到達できないナビゲーションはすぐに失敗します。どちらの場合もテストできる`res`はなく、失敗のメッセージにブラウザのエラーが入ります。

### 失敗時のページ

アクションが失敗すると、Probeはその時点のページを保存します。ページ全体のPNGのスクリーンショットと、文書のHTMLです。2つのファイルは`probe-browser-failure-1947177907.png`と`.html`のように同じ名前を共有します。失敗のメッセージには、ページのURLとあわせて両方のファイルのパスが入ります。

```
action error in step_execute: action execution failed (caused by: ... context deadline exceeded (page at failure: url http://localhost:8090/, screenshot /tmp/probe-browser-failure-1947177907.png, html /tmp/probe-browser-failure-1947177907.html))
```

`--report`で書き出すファイルでも、このメッセージが失敗の`message`になります。テストがfalseになるのはアクションの失敗ではないため、そのときはページを保存しません。残したい場合は`capture_screenshot`アクションを加えます。

ファイルはシステムの一時ディレクトリに保存します。`evidence_dir`を指定するとそこに保存し、ディレクトリは必要に応じて作成します。CIのアーティファクトとしてアップロードする場合に便利です。

```yaml
- name: Checkout page
  uses: browser
  with:
    evidence_dir: out/browser
    actions:
    - name: navigate
      url: "{{vars.url}}/checkout"
    - name: wait_visible
      selector: "#pay"
```

ページを読み取れるように、ブラウザはタイムアウトのあとも最大10秒残します。ブラウザが起動しなかった場合など、読み取れなかったときはその旨をメッセージに書き、何も保存しません。

## パフォーマンスの考慮事項

- **ヘッドレスモード**: より高速な実行のため`headless: true`（デフォルト）を使用
- **タイムアウト**: ハングを防ぐために適切なタイムアウトを設定
- **リソース使用量**: ブラウザアクションは他のアクションよりも多くのリソースを消費
- **スクリーンショット**: 大きなスクリーンショットは大量のメモリを消費

## セキュリティ機能

ブラウザアクションはいくつかのセキュリティ対策を実装しています：

- **サンドボックス実行**: ChromeDPはサンドボックス環境で実行
- **タイムアウト保護**: 無限ハングを防止
- **URL検証**: ナビゲーション前にURLを検証
- **リソース制限**: 組み込みのリソース使用制限
