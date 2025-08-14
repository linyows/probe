# ブラウザアクション

`browser`アクションはChromeDPを使用してWebブラウザを自動化し、テスト、スクレイピング、Webアプリケーションとの相互作用のための包括的なWeb自動化機能を提供します。

## 基本的な構文

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

### URLへのナビゲート

```yaml
- name: "Open Website"
  uses: browser
  with:
    action: navigate
    url: "https://example.com"
    headless: true
  test: res.code == 0
  outputs:
    load_time: res.time_ms
```

### テキストコンテンツの抽出

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

### ログインフロー

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

```yaml
- name: "Browser Action with Error Handling"
  uses: browser
  with:
    action: click
    selector: "#may-not-exist"
    timeout: "5s"
  test: res.code == 0 || (res.success == "false" && res.error | contains("not found"))
  continue_on_error: true
  outputs:
    click_success: res.code == 0
    error_type: |
      {{res.code == 0 ? "none" :
        res.error | contains("timeout") ? "timeout" :
        res.error | contains("not found") ? "element_not_found" :
        "unknown"}}
```

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
