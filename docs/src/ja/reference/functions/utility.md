# ユーティリティ関数

汎用ユーティリティ関数群です。

## `uuid`

ランダムなUUID（バージョン4）を生成します。

**構文:** `uuid()`  
**戻り値:** String（UUID）

```yaml
with:
  headers:
    X-Request-ID: "{{uuid()}}"
    # "f47ac10b-58cc-4372-a567-0e02b2c3d479"のような値を生成

outputs:
  correlation_id: uuid()
```

## `random`

0から指定された最大値（排他的）までのランダムな整数を生成します。

**構文:** `random(max)`  
**戻り値:** Integer

```yaml
with:
  url: "{{vars.BASE_URL}}/test?seed={{random(1000)}}"
  # 0-999のランダム数値を生成

outputs:
  random_delay: random(5000)
  # 0-4999のランダム数値（遅延用ミリ秒）
```

## `default`

入力が空またはnullの場合にデフォルト値を返します。

**構文:** `default(value, default_value)` または `value ?? default_value`  
**戻り値:** 任意の型

```yaml
vars:
  request_timeout: "{{REQUEST_TIMEOUT ?? '30s'}}"
  custom_timeout: "{{CUSTOM_TIMEOUT}}"

with:
  timeout: "{{default(vars.custom_timeout, '60s')}}"
```

## `coalesce`

リストから最初の空でない値を返します。

**構文:** `coalesce(value1, value2, value3, ...)`  
**戻り値:** 任意の型

```yaml
vars:
  custom_api_url: "{{CUSTOM_API_URL}}"
  default_api_url: "{{DEFAULT_API_URL}}"
  step_timeout: "{{STEP_TIMEOUT}}"
  job_timeout: "{{JOB_TIMEOUT}}"
  api_url: "{{coalesce(vars.custom_api_url, vars.default_api_url, 'https://api.example.com')}}"

with:
  timeout: "{{coalesce(vars.step_timeout, vars.job_timeout, '30s')}}"
```
