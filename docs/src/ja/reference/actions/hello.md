# Helloアクション

`hello`アクションは開発、デバッグ、ワークフロー検証に使用される簡単なテストアクションです。

## 基本的な構文

```yaml
steps:
  - name: "Test Hello"
    uses: hello
    with:
      message: "Hello, World!"
```

## パラメータ

### `message` (オプション)

**型:** String  
**デフォルト:** `"Hello from Probe!"`  
**説明:** 表示するメッセージ  
**サポート:** テンプレート式

```yaml
with:
  message: "Hello, World!"
  message: "Current time: {{unixtime()}}"
vars:
  user_name: "{{USER_NAME}}"

  message: "Hello {{vars.user_name}}"
```

### `delay` (オプション)

**型:** Duration  
**デフォルト:** `0s`  
**説明:** 完了前の人為的な遅延

```yaml
with:
  message: "Delayed hello"
  delay: "2s"
```

## レスポンスオブジェクト

helloアクションは次のプロパティを持つ`res`オブジェクトを提供します：

 | プロパティ  | 型      | 説明                                     | 
 | ----------  | ------  | -------------                            | 
 | `message`   | String  | 表示されたメッセージ                     | 
 | `time`      | Integer | かかった時間（ミリ秒）（遅延を含む）     | 
 | `timestamp` | String  | アクション完了時のISO 8601タイムスタンプ | 

## Hello例

#### 基本テスト

```yaml
steps:
  - name: "Simple Test"
    uses: hello
    test: res.message != ""
    outputs:
      test_time: res.time
```

### タイミングテスト

```yaml
steps:
  - name: "Timing Test"
    uses: hello
    with:
      message: "Testing timing"
      delay: "1s"
    test: res.time >= 1000 && res.time < 1100
```

### テンプレートテスト

```yaml
steps:
  - name: "Template Test"
    uses: hello
    with:
vars:
  user: "{{USER}}"

      message: "User: {{vars.user}}, Time: {{unixtime()}}"
    test: res.message | contains(vars.user)
    outputs:
      rendered_message: res.message
```

