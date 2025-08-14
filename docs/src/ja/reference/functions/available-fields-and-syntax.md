# 利用可能なフィールドと構文

特定のフィールドでは、中カッコ2つで囲むテンプレート式（例: <span v-pre>{{ your_variable }}</span>）やブール式を使用でき、テンプレート式中では、組み込み関数が利用できます。
組み込み関数は文字列操作、データフォーマット、数学的操作などのユーティリティを提供します。組み込み関数は以下のフィールドで利用できます：

レベル   | フィールド名 | テンプレート式 | ブール式 | 補足
---:     | ---:         | :---:          | :---:    | ---
workflow | name         | -              | -        | ワークフロー名
workflow | description  | -              | -        | ワークフロー説明
workflow | vars         | ✅             | -        | グローバル変数
job      | name         | ✅             | -        | ジョブ名
job      | skipif       | -              | ✅       | ジョブスルーの条件式
step     | name         | ✅             | -        | ステップ名
step     | with         | ✅             | -        | アクション引数
step     | echo         | ✅             | -        | レポート出力
step     | vars         | ✅             | -        | ステップ変数
step     | outputs      | ✅             | -        | ワークフロー間共有変数
step     | skipif       | -              | ✅       | ステップスルーの条件式
step     | iter         | -              | -        | イテレーション変数

## 関数の構文

関数はパイプ演算子（`|`）を使用してテンプレート式内で呼び出すか、直接的な関数呼び出しとして使用します：

```yaml
# パイプ構文（チェーンに推奨）
vars:
  user_name: "{{USER_NAME}}"
  base_url: "{{BASE_URL}}"
  path: "{{PATH}}"

value: "{{vars.user_name | upper | trim}}"

# 直接関数呼び出し
value: "{{upper(vars.user_name)}}"

# 混合使用
value: "{{vars.base_url}}/{{vars.path | lower | replace(' ', '-')}}"
```
## 関数カテゴリ

- **[文字列関数](./string)** - 文字列操作とフォーマット
- **[日時関数](./datetime)** - 日付と時刻のユーティリティ
- **[エンコーディング関数](./encoding)** - Base64、URLエンコーディングなど
- **[数学関数](./mathematics)** - 数値演算
- **[ユーティリティ関数](./utility)** - 汎用ユーティリティ
- **[JSON関数](./json)** - JSON操作とクエリ
