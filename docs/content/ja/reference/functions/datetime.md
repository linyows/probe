# 日時関数

日付と時刻のユーティリティ関数群です。

| 関数 | 説明 |
|---|---|
| `now()` | 現在時刻を時刻値として返す |
| `date(s)` / `date(s, layout)` / `date(s, layout, tz)` | 文字列を時刻値に解析する |
| `duration(s)` | `"1h30m"` のような期間を解析する |
| `timezone(name)` | `"UTC"` や `"Asia/Tokyo"` などのロケーションを取得する |

## `now`

`now()` が返すのは数値ではなく時刻値です。
フォーマットするには Go のレイアウトを `Format` に渡し、Unix タイムスタンプが欲しい場合は `Unix()` または Probe 固有の [`unixtime()`](/ja/reference/functions/probe#unixtime) を使います。

```yaml
outputs:
  today: "{{now().Format('2006-01-02')}}"
  started_at: "{{now().Format('2006-01-02T15:04:05Z07:00')}}"
  epoch: "{{unixtime()}}"
```

よく使うレイアウトは次のとおりです。

- `2006-01-02` — 日付（YYYY-MM-DD）
- `15:04:05` — 時刻（HH:MM:SS）
- `2006-01-02 15:04:05` — 日時
- `2006-01-02T15:04:05Z07:00` — RFC 3339

## `date`

`date()` は現在時刻をフォーマットする関数ではなく、文字列を時刻値に解析する関数です。

```yaml
test: date(res.body.expires_at) > now()
```

レイアウトを指定すると、その書式で解析します。

```yaml
test: date(res.body.published_on, "02/01/2006") < now()
```

## `duration`

期間を表す文字列を解析します。
時刻値との加減算やしきい値の比較に使えます。

```yaml
test: date(res.body.expires_at) > now() + duration("24h")
```
