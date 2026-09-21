# 組み込み関数

Probe の式は [expr-lang/expr](https://expr-lang.org/)（v1.17）で評価されます。
利用できる関数は、expr の組み込み関数と、Probe が独自に登録した関数の 2 種類です。

- **[利用可能なフィールドと構文](./functions/available-fields-and-syntax)** - 式が書けるフィールド、パイプ、演算子、評価の制限
- **[Probe 固有関数](./functions/probe)** - `match_json`、`parse_json`、`unixtime`、`random_str` など
- **[文字列関数](./functions/string)** - 文字列操作とフォーマット
- **[数値関数](./functions/mathematics)** - 数値演算と型変換
- **[日時関数](./functions/datetime)** - 日付と時刻のユーティリティ
- **[配列・マップ関数](./functions/array)** - コレクション操作
- **[JSON・エンコーディング関数](./functions/json)** - JSON と Base64
- **[高度な関数使用法](./functions/advanced-usage)** - 組み合わせとベストプラクティス
