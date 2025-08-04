---
title: Built-in Functions Reference
description: Complete reference for all built-in expression functions
weight: 40
---

# Built-in Functions Reference

This page provides comprehensive documentation for all built-in functions available in Probe expressions and templates. These functions can be used in template expressions (`{{}}`) and test conditions throughout your workflows.

## Overview

Built-in functions provide utilities for string manipulation, data formatting, mathematical operations, and more. They are available in all expression contexts including:

- Environment variable values
- HTTP request URLs, headers, and bodies  
- Test conditions and assertions
- Output expressions
- Conditional statements (`if` expressions)

### Function Categories

- **[String Functions](#string-functions)** - String manipulation and formatting
- **[Date/Time Functions](#datetime-functions)** - Date and time utilities
- **[Encoding Functions](#encoding-functions)** - Base64, URL encoding, and more
- **[Mathematical Functions](#mathematical-functions)** - Numeric operations
- **[Utility Functions](#utility-functions)** - General-purpose utilities
- **[JSON Functions](#json-functions)** - JSON manipulation and queries

## Function Syntax

Functions are called within template expressions using the pipe operator (`|`) or as direct function calls:

```yaml
# Pipe syntax (preferred for chaining)
vars:
  user_name: "{{USER_NAME}}"
  base_url: "{{BASE_URL}}"
  path: "{{PATH}}"

value: "{{vars.user_name | upper | trim}}"

# Direct function call
value: "{{upper(vars.user_name)}}"

# Mixed usage
value: "{{vars.base_url}}/{{vars.path | lower | replace(' ', '-')}}"
```

## String Functions

### `upper`

Converts a string to uppercase.

**Syntax:** `string | upper` or `upper(string)`  
**Returns:** String

```yaml
env:
  SERVICE_NAME: "{{env.SERVICE | upper}}"
  # If SERVICE="api-service", result is "API-SERVICE"

test: res.json.status | upper == "SUCCESS"
```

### `lower`

Converts a string to lowercase.

**Syntax:** `string | lower` or `lower(string)`  
**Returns:** String

```yaml
with:
  url: "{{env.BASE_URL}}/{{env.ENDPOINT | lower}}"
  # If ENDPOINT="USERS", result is "/users"

test: res.json.type | lower == "error"
```

### `trim`

Removes whitespace from both ends of a string.

**Syntax:** `string | trim` or `trim(string)`  
**Returns:** String

```yaml
env:
  API_KEY: "{{env.RAW_API_KEY | trim}}"
  # Removes leading/trailing spaces

with:
  headers:
    Authorization: "Bearer {{env.TOKEN | trim}}"
```

### `trimPrefix`

Removes a prefix from the beginning of a string.

**Syntax:** `trimPrefix(string, prefix)`  
**Returns:** String

```yaml
outputs:
  clean_url: trimPrefix(res.headers.Location, "https://")
  # "https://api.example.com/users" becomes "api.example.com/users"

env:
  CLEAN_PATH: "{{trimPrefix(env.FULL_PATH, '/api/v1')}}"
```

### `trimSuffix`

Removes a suffix from the end of a string.

**Syntax:** `trimSuffix(string, suffix)`  
**Returns:** String

```yaml
outputs:
  base_name: trimSuffix(res.json.filename, ".json")
  # "data.json" becomes "data"

env:
  SERVICE_NAME: "{{trimSuffix(env.CONTAINER_NAME, '-container')}}"
```

### `replace`

Replaces all occurrences of a substring with another string.

**Syntax:** `replace(string, old, new)` or `string | replace(old, new)`  
**Returns:** String

```yaml
with:
  url: "{{env.TEMPLATE_URL | replace('{id}', outputs.user.id)}}"
  # "/users/{id}/profile" becomes "/users/123/profile"

env:
  SAFE_NAME: "{{env.USER_INPUT | replace(' ', '_') | replace('-', '_')}}"
```

### `split`

Splits a string by a delimiter and returns an array.

**Syntax:** `split(string, delimiter)`  
**Returns:** Array of strings

```yaml
outputs:
  url_parts: split(res.headers.Location, "/")
  # "https://api.example.com/v1/users" becomes ["https:", "", "api.example.com", "v1", "users"]
  
  first_part: split(env.COMMA_LIST, ",")[0]
  # "apple,banana,cherry" -> first element is "apple"
```

### `join`

Joins an array of strings with a delimiter.

**Syntax:** `join(array, delimiter)`  
**Returns:** String

```yaml
env:
  # Assume we have an array from a previous step
  COMBINED: "{{join(outputs.data.items, ', ')}}"
  # ["apple", "banana", "cherry"] becomes "apple, banana, cherry"
```

### `contains`

Checks if a string contains a substring.

**Syntax:** `contains(string, substring)` or `string | contains(substring)`  
**Returns:** Boolean

```yaml
test: |
  res.json.message | contains("success") &&
  res.headers["Content-Type"] | contains("application/json")

if: env.ENVIRONMENT | contains("prod")
```

### `hasPrefix`

Checks if a string starts with a prefix.

**Syntax:** `hasPrefix(string, prefix)`  
**Returns:** Boolean

```yaml
test: hasPrefix(res.headers.Location, "https://")

if: hasPrefix(env.API_URL, "https://secure")
```

### `hasSuffix`

Checks if a string ends with a suffix.

**Syntax:** `hasSuffix(string, suffix)`  
**Returns:** Boolean

```yaml
test: hasSuffix(res.json.filename, ".json")

if: hasSuffix(env.IMAGE_NAME, ":latest")
```

### `len`

Returns the length of a string or array.

**Syntax:** `len(value)`  
**Returns:** Integer

```yaml
test: |
  len(res.json.items) > 0 &&
  len(res.json.message) > 10

outputs:
  item_count: len(res.json.data)
  name_length: len(res.json.user.name)
```

## Date/Time Functions

### `now`

Returns the current Unix timestamp.

**Syntax:** `now()`  
**Returns:** Integer (Unix timestamp)

```yaml
outputs:
  timestamp: now()
  # Returns something like 1693574400

env:
  REQUEST_TIME: "{{now()}}"
```

### `unixtime`

Alias for `now()` - returns current Unix timestamp.

**Syntax:** `unixtime()`  
**Returns:** Integer (Unix timestamp)

```yaml
with:
  headers:
    X-Timestamp: "{{unixtime()}}"
```

### `iso8601`

Returns the current time in ISO 8601 format.

**Syntax:** `iso8601()`  
**Returns:** String (ISO 8601 formatted)

```yaml
outputs:
  created_at: iso8601()
  # Returns something like "2023-09-01T12:30:00Z"

with:
  body: |
    {
      "timestamp": "{{iso8601()}}",
      "event": "test_execution"
    }
```

### `date`

Formats the current time using Go's time format layout.

**Syntax:** `date(layout)`  
**Returns:** String (formatted date)

**Common layouts:**
- `2006-01-02` - Date (YYYY-MM-DD)
- `15:04:05` - Time (HH:MM:SS)
- `2006-01-02 15:04:05` - DateTime
- `Mon Jan 2 15:04:05 2006` - Full format

```yaml
outputs:
  date_only: date("2006-01-02")
  # Returns "2023-09-01"
  
  time_only: date("15:04:05")
  # Returns "12:30:45"
  
  full_datetime: date("2006-01-02 15:04:05")
  # Returns "2023-09-01 12:30:45"

with:
  headers:
    X-Date: "{{date('Mon Jan 2 15:04:05 2006')}}"
    # Returns "Fri Sep 1 12:30:45 2023"
```

## Encoding Functions

### `base64`

Encodes a string to base64.

**Syntax:** `base64(string)` or `string | base64`  
**Returns:** String (base64 encoded)

```yaml
with:
  headers:
    Authorization: "Basic {{base64(env.USERNAME + ':' + env.PASSWORD)}}"
    # Encodes "user:pass" to "dXNlcjpwYXNz"

outputs:
  encoded_data: base64(res.text)
```

### `base64decode`

Decodes a base64 string.

**Syntax:** `base64decode(string)` or `string | base64decode`  
**Returns:** String (decoded)

```yaml
outputs:
  decoded_token: base64decode(res.json.token)
  # Decodes base64 token to plain text

test: base64decode(res.json.data) | contains("expected_value")
```

### `urlEncode`

URL encodes a string (percent encoding).

**Syntax:** `urlEncode(string)` or `string | urlEncode`  
**Returns:** String (URL encoded)

```yaml
with:
  url: "{{env.BASE_URL}}/search?q={{env.SEARCH_TERM | urlEncode}}"
  # Encodes "hello world" to "hello%20world"

outputs:
  encoded_param: urlEncode(res.json.user_input)
```

### `urlDecode`

Decodes a URL encoded string.

**Syntax:** `urlDecode(string)` or `string | urlDecode`  
**Returns:** String (decoded)

```yaml
outputs:
  original_query: urlDecode(res.json.encoded_query)
  # Decodes "hello%20world" to "hello world"
```

## Mathematical Functions

### `add`

Adds two numbers.

**Syntax:** `add(a, b)`  
**Returns:** Number

```yaml
outputs:
  total_time: add(res.time, 100)
  # Adds 100ms to response time

test: add(res.json.count, res.json.pending) > 50
```

### `sub`

Subtracts the second number from the first.

**Syntax:** `sub(a, b)`  
**Returns:** Number

```yaml
outputs:
  time_diff: sub(unixtime(), res.json.created_at)
  # Calculate age in seconds

test: sub(res.time, outputs.baseline.time) < 500
```

### `mul`

Multiplies two numbers.

**Syntax:** `mul(a, b)`  
**Returns:** Number

```yaml
outputs:
  time_in_seconds: mul(res.time, 0.001)
  # Convert milliseconds to seconds

test: mul(res.json.price, res.json.quantity) <= 1000
```

### `div`

Divides the first number by the second.

**Syntax:** `div(a, b)`  
**Returns:** Number

```yaml
outputs:
  average_time: div(res.json.total_time, res.json.request_count)
  # Calculate average

test: div(res.json.success_count, res.json.total_count) > 0.95
```

### `mod`

Returns the remainder of division.

**Syntax:** `mod(a, b)`  
**Returns:** Number

```yaml
test: mod(res.json.id, 2) == 0
# Check if ID is even

if: mod(unixtime(), 3600) < 60
# Execute only in the first minute of each hour
```

### `round`

Rounds a number to the nearest integer.

**Syntax:** `round(number)`  
**Returns:** Integer

```yaml
outputs:
  rounded_time: round(div(res.time, 1000))
  # Convert to seconds and round

test: round(res.json.score) >= 8
```

### `floor`

Rounds a number down to the nearest integer.

**Syntax:** `floor(number)`  
**Returns:** Integer

```yaml
outputs:
  time_seconds: floor(div(res.time, 1000))
  # Convert milliseconds to seconds (rounded down)
```

### `ceil`

Rounds a number up to the nearest integer.

**Syntax:** `ceil(number)`  
**Returns:** Integer

```yaml
outputs:
  min_requests: ceil(mul(res.json.users, 1.5))
  # Calculate minimum requests needed (rounded up)
```

## Utility Functions

### `uuid`

Generates a random UUID (version 4).

**Syntax:** `uuid()`  
**Returns:** String (UUID)

```yaml
with:
  headers:
    X-Request-ID: "{{uuid()}}"
    # Generates something like "f47ac10b-58cc-4372-a567-0e02b2c3d479"

outputs:
  correlation_id: uuid()
```

### `random`

Generates a random integer between 0 and the specified maximum (exclusive).

**Syntax:** `random(max)`  
**Returns:** Integer

```yaml
with:
  url: "{{env.BASE_URL}}/test?seed={{random(1000)}}"
  # Generates random number 0-999

outputs:
  random_delay: random(5000)
  # Random number 0-4999 (for delay in ms)
```

### `env`

Accesses environment variables (same as `env.VARIABLE_NAME`).

**Syntax:** `env(variable_name)`  
**Returns:** String

```yaml
# These are equivalent:
with:
  url: "{{env.API_URL}}"
  url: "{{env('API_URL')}}"
```

### `default`

Returns a default value if the input is empty or null.

**Syntax:** `default(value, default_value)` or `value || default_value`  
**Returns:** Any type

```yaml
env:
  TIMEOUT: "{{env.REQUEST_TIMEOUT || '30s'}}"
  # Use 30s if REQUEST_TIMEOUT is not set

with:
  timeout: "{{default(env.CUSTOM_TIMEOUT, '60s')}}"
```

### `coalesce`

Returns the first non-empty value from a list.

**Syntax:** `coalesce(value1, value2, value3, ...)`  
**Returns:** Any type

```yaml
env:
  API_URL: "{{coalesce(env.CUSTOM_API_URL, env.DEFAULT_API_URL, 'https://api.example.com')}}"
  # Uses first available URL

with:
  timeout: "{{coalesce(env.STEP_TIMEOUT, env.JOB_TIMEOUT, '30s')}}"
```

## JSON Functions

### `tojson`

Converts a value to JSON string.

**Syntax:** `tojson(value)` or `value | tojson`  
**Returns:** String (JSON)

```yaml
with:
  body: "{{outputs.user_data | tojson}}"
  # Converts object to JSON string

outputs:
  json_response: tojson(res.json)
```

### `fromjson`

Parses a JSON string to an object.

**Syntax:** `fromjson(json_string)` or `json_string | fromjson`  
**Returns:** Object

```yaml
outputs:
  parsed_data: fromjson(res.text)
  # Parse JSON string to object

test: fromjson(res.json.metadata).version == "1.0"
```

### `jsonpath`

Extracts values from JSON using JSONPath expressions.

**Syntax:** `jsonpath(json_object, path)`  
**Returns:** Any type

```yaml
outputs:
  user_names: jsonpath(res.json, "$.users[*].name")
  # Extract all user names from array
  
  first_email: jsonpath(res.json, "$.users[0].email")
  # Get first user's email

test: jsonpath(res.json, "$.status.code") == 200
```

### `keys`

Returns the keys of an object as an array.

**Syntax:** `keys(object)`  
**Returns:** Array of strings

```yaml
outputs:
  header_names: keys(res.headers)
  # Get all response header names
  
  json_fields: keys(res.json)
  # Get all JSON object keys

test: len(keys(res.json)) > 5
# Ensure response has more than 5 fields
```

### `values`

Returns the values of an object as an array.

**Syntax:** `values(object)`  
**Returns:** Array

```yaml
outputs:
  header_values: values(res.headers)
  # Get all response header values
  
  all_user_names: values(res.json.users)
  # Get all values from users object
```

## Advanced Function Usage

### Function Chaining

Functions can be chained using the pipe operator:

```yaml
env:
  CLEAN_NAME: "{{env.RAW_NAME | trim | lower | replace(' ', '-')}}"
  # Chain: trim whitespace → lowercase → replace spaces with hyphens

with:
  url: "{{env.BASE_URL | trimSuffix('/') | replace('http://', 'https://')}}/api"
  # Chain: remove trailing slash → force HTTPS → add path
```

### Conditional Function Usage

Functions can be used in conditional expressions:

```yaml
test: |
  res.status == 200 &&
  len(res.json.items) > 0 &&
  contains(res.json.status | upper, "SUCCESS")

if: |
  env.ENVIRONMENT == "production" ||
  (env.ENVIRONMENT == "staging" && contains(env.BRANCH_NAME, "release"))
```

### Complex Data Manipulation

```yaml
outputs:
  # Extract and format user data
  formatted_users: |
    {{range res.json.users}}
      {{.name | upper}}: {{.email | lower}}
    {{end}}
  
  # Calculate metrics
  success_rate: |
    {{div(mul(res.json.successful_requests, 100), res.json.total_requests)}}%
  
  # Generate URLs
  api_endpoints: |
    {{env.BASE_URL | trimSuffix('/')}}/{{env.API_VERSION}}/{{env.RESOURCE | lower}}
```

### Error-Safe Function Usage

Use default values and null checks to make functions more robust:

```yaml
outputs:
  safe_length: "{{len(res.json.items || []))}}"
  # Use empty array if items is null
  
  safe_name: "{{res.json.user.name | default('Unknown') | upper}}"  
  # Provide default if name is missing
  
  safe_url: "{{coalesce(env.CUSTOM_URL, env.DEFAULT_URL, 'https://fallback.com')}}"
  # Multiple fallback options
```

## Performance Considerations

### Function Performance

- **String functions:** Generally fast, but avoid excessive chaining
- **Date functions:** `now()` and `iso8601()` have minimal overhead
- **JSON functions:** `jsonpath()` can be slow on large objects
- **Mathematical functions:** Very fast for simple operations

### Best Practices

```yaml
# Good: Compute once, reuse
env:
  CURRENT_TIME: "{{iso8601()}}"
  API_URL: "{{env.BASE_URL | trimSuffix('/')}}"

jobs:
  test:
    steps:
      - name: "Use precomputed values"
        with:
          url: "{{env.API_URL}}/health"
          headers:
            X-Timestamp: "{{env.CURRENT_TIME}}"

# Avoid: Recomputing in every step
      - name: "Inefficient"
        with:
          url: "{{env.BASE_URL | trimSuffix('/')}}/health"  # Recomputed
          headers:
            X-Timestamp: "{{iso8601()}}"  # Different timestamp
```

## See Also

- **[YAML Configuration](../yaml-configuration/)** - Using functions in YAML config
- **[Actions Reference](../actions-reference/)** - Functions in action parameters
- **[Concepts: Expressions and Templates](../../concepts/expressions-and-templates/)** - Expression language guide
- **[How-tos: Dynamic Configuration](../../how-tos/environment-management/)** - Practical function usage