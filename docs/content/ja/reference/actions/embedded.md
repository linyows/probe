# Embeddedアクション

`embedded`アクションは**ジョブファイル**を1つのステップとして実行します。共通のセットアップやチェックを別ファイルに切り出し、複数のワークフローから再利用できます。

読み込むファイルはワークフローではなくジョブです。`name`、`steps`、必要なら`defaults`を持ち、`jobs`キーはありません。

## 基本的な構文

**auth.yml:**
```yaml
name: Authentication
steps:
  - name: Get token
    id: get_token
    uses: shell
    with:
      cmd: echo "0123456789"
    test: res.code == 0
    outputs:
      mytoken: replace(res.stdout, '\n', '')
```

**workflow.yml:**
```yaml
jobs:
- name: Main
  steps:
    - name: Authenticate
      id: auth
      uses: embedded
      with:
        path: "./auth.yml"
        vars:
          environment: "{{vars.environment}}"
      test: res.code == 0
      outputs:
        token: res.outputs.mytoken

    - name: Call the API
      uses: http
      with:
        method: GET
        url: "{{vars.api_url}}/me"
        headers:
          authorization: "Bearer {{outputs.auth.token}}"
      test: res.code == 200
```

## パラメータ

埋め込みのステップでは、実行するジョブファイルと、そこに渡す変数を指定します。

| パラメータ | 型 | 必須 | デフォルト | 説明 |
|---|---|---|---|---|
| `path` | String | 必須 | - | ジョブファイルのパス。ワークフローファイルからではなく、カレントディレクトリからの相対パスとして解決されます |
| `vars` | Object | 任意 | `{}` | 埋め込みジョブに渡す変数。向こう側では`vars.<name>`で参照します |

## レスポンスオブジェクト

埋め込みジョブの終了後、`res`にその結果と出力が入ります。

| プロパティ | 型 | 説明 |
|---|---|---|
| `res.code` | Integer | 埋め込みジョブのすべてのステップが成功したとき`0` |
| `res.outputs` | Object | 埋め込みジョブのステップが公開したoutputs。出力名がキーになります |
| `res.report` | String | 埋め込みジョブのレポート。親のレポートにもネストして表示されます |
| `res.error` | String | 失敗時のエラーメッセージ |
| `rt` | Object | 埋め込みジョブの実行時間 |

`res.outputs`のキーは出力名なので、`mytoken`として公開した値は`res.outputs.mytoken`で読みます。

## 関連項目

- **[変数](/ja/reference/actions/variables)** - ステップで使える変数
- **[YAML設定](/ja/reference/yaml-configuration)** - ステップのプロパティ
- **[ファイルマージ](/ja/guide/concepts/file-merging)** - 複数ファイルからワークフローを組み立てる
