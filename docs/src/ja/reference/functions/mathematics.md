# 数学関数

数値演算の関数群です。

## `add`

2つの数値を加算します。

**構文:** `add(a, b)`  
**戻り値:** Number

```yaml
outputs:
  total_time: add(res.time, 100)
  # レスポンス時間に100msを追加

test: add(res.body.json.count, res.body.json.pending) > 50
```

## `sub`

最初の数値から2番目の数値を減算します。

**構文:** `sub(a, b)`  
**戻り値:** Number

```yaml
outputs:
  time_diff: sub(unixtime(), res.body.json.created_at)
  # 秒単位の経過時間を計算

test: sub(res.time, outputs.baseline.time) < 500
```

## `mul`

2つの数値を乗算します。

**構文:** `mul(a, b)`  
**戻り値:** Number

```yaml
outputs:
  time_in_seconds: mul(res.time, 0.001)
  # ミリ秒を秒に変換

test: mul(res.body.json.price, res.body.json.quantity) <= 1000
```

## `div`

最初の数値を2番目の数値で除算します。

**構文:** `div(a, b)`  
**戻り値:** Number

```yaml
outputs:
  average_time: div(res.body.json.total_time, res.body.json.request_count)
  # 平均を計算

test: div(res.body.json.success_count, res.body.json.total_count) > 0.95
```

## `mod`

除算の余りを返します。

**構文:** `mod(a, b)`  
**戻り値:** Number

```yaml
test: mod(res.body.json.id, 2) == 0
# IDが偶数かチェック

if: mod(unixtime(), 3600) < 60
# 毎時の最初の1分間のみ実行
```

## `round`

数値を最も近い整数に四捨五入します。

**構文:** `round(number)`  
**戻り値:** Integer

```yaml
outputs:
  rounded_time: round(div(res.time, 1000))
  # 秒に変換して四捨五入

test: round(res.body.json.score) >= 8
```

## `floor`

数値を最も近い整数に切り下げます。

**構文:** `floor(number)`  
**戻り値:** Integer

```yaml
outputs:
  time_seconds: floor(div(res.time, 1000))
  # ミリ秒を秒に変換（切り下げ）
```

## `ceil`

数値を最も近い整数に切り上げます。

**構文:** `ceil(number)`  
**戻り値:** Integer

```yaml
outputs:
  min_requests: ceil(mul(res.body.json.users, 1.5))
  # 必要な最小リクエスト数を計算（切り上げ）
```
