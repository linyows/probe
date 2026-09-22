# 数値関数

数値演算と型変換の関数群です。

## 演算子

四則演算は演算子で書きます。
`add()` `sub()` `mul()` `div()` `mod()` といった関数はありません。

```yaml
outputs:
  millis: "{{rt.sec * 1000}}"

test: res.body.id % 2 == 0
```

利用できる演算子は `+` `-` `*` `/` `%`（剰余）`**`（べき乗）です。

## 数値関数

| 関数 | 説明 |
|---|---|
| `abs(n)` | 絶対値 |
| `ceil(n)` | 切り上げ |
| `floor(n)` | 切り下げ |
| `round(n)` | 四捨五入 |
| `max(a, b, ...)` | 最大値 |
| `min(a, b, ...)` | 最小値 |
| `sum(array)` | 配列の合計 |
| `mean(array)` | 平均 |
| `median(array)` | 中央値 |
| `bitnot(n)` | ビット反転 |

```yaml
outputs:
  rounded: "{{round(rt.sec * 1000)}}"
  slowest: "{{max(rt.sec, outputs.baseline.sec)}}"

test: mean(outputs.load.times) < 500
```

## 型変換関数

| 関数 | 説明 |
|---|---|
| `int(v)` | 整数に変換 |
| `float(v)` | 浮動小数点数に変換 |
| `string(v)` | 文字列に変換 |
| `type(v)` | 値の型名（`string`、`int` など）を返す |

Probe 固有の [`parse_int`](/ja/reference/functions/probe#parse-int) と [`parse_float`](/ja/reference/functions/probe#parse-float) は、10 進表記の文字列を厳密に解析し、64 ビット値を返す点が `int` / `float` と異なります。
