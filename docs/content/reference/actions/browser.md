# Browser Action

The `browser` action automates web browsers using ChromeDP, providing comprehensive web automation capabilities for testing, scraping, and interaction with web applications.

## Basic Syntax

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

## Parameters

### `action` (required)

**Type:** String  
**Description:** The browser action to perform  
**Values:** 
- **Navigation:** `navigate`
- **Text/Content:** `text`, `value`, `get_html`
- **Attributes:** `get_attribute`
- **Interactions:** `click`, `double_click`, `right_click`, `hover`, `focus`
- **Input:** `type`, `send_keys`, `select`
- **Forms:** `submit`
- **Scrolling:** `scroll`
- **Screenshots:** `screenshot`, `capture_screenshot`, `full_screenshot`
- **Waiting:** `wait_visible`, `wait_not_visible`, `wait_ready`, `wait_text`, `wait_enabled`

### `url` (optional)

**Type:** String  
**Description:** URL to navigate to (required for navigate action)  
**Supports:** Template expressions

```yaml
with:
  action: navigate
  url: "https://example.com"
  url: "{{vars.base_url}}/login"
```

### `selector` (optional)

**Type:** String  
**Description:** CSS selector for targeting elements  
**Supports:** Template expressions

```yaml
with:
  action: get_text
  selector: "h1"
  selector: "#main-title"
  selector: ".article-content p:first-child"
```

### `value` (optional)

**Type:** String  
**Description:** Value to type or text to wait for  
**Supports:** Template expressions

```yaml
with:
  action: type
  selector: "#email"
  value: "user@example.com"
  value: "{{vars.username}}"
```

### `attribute` (optional)

**Type:** String  
**Description:** Attribute name to retrieve (required for get_attribute action)

```yaml
with:
  action: get_attribute
  selector: "a"
  attribute: "href"
```

### `headless` (optional)

**Type:** Boolean  
**Default:** `true`  
**Description:** Whether to run browser in headless mode

```yaml
with:
  action: navigate
  url: "https://example.com"
  headless: false  # Show browser window
```

### `timeout` (optional)

**Type:** Duration  
**Default:** `30s`  
**Description:** Action timeout

```yaml
with:
  action: wait_visible
  selector: ".loading"
  timeout: "60s"
```

## Response Object

The browser action provides a `res` object with action-specific properties:

### Common Properties

| Property | Type | Description |
|----------|------|-------------|
| `code` | Integer | Result code (0 = success, non-zero = error) |
| `results` | Object | Action-specific results (text, values, etc.) |

### Navigation Response

| Property | Type | Description |
|----------|------|-------------|
| `url` | String | URL that was navigated to |
| `time_ms` | String | Navigation time in milliseconds |

### Text/Attribute Response

| Property | Type | Description |
|----------|------|-------------|
| `selector` | String | CSS selector used |
| `text` | String | Extracted text content (get_text) |
| `attribute` | String | Attribute name (get_attribute) |
| `value` | String | Attribute value (get_attribute) |
| `exists` | String | "true" if attribute exists |

### Screenshot Response

| Property | Type | Description |
|----------|------|-------------|
| `screenshot` | String | Base64-encoded screenshot |
| `size_bytes` | String | Screenshot size in bytes |

## Browser Actions

### Navigate to URL

```yaml
- name: "Open Website"
  uses: browser
  with:
    action: navigate
    url: "https://example.com"
    headless: true
  test: res.code == 0
  outputs:
    load_time: rt.sec * 1000
```

### Extract Text Content

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

### Get Element Attributes

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

### Form Interactions

```yaml
# Fill form fields
- name: "Enter Email"
  uses: browser
  with:
    action: type
    selector: "#email"
    value: "user@example.com"
  test: res.code == 0

# Click buttons
- name: "Click Submit"
  uses: browser
  with:
    action: click
    selector: "#submit-btn"
  test: res.code == 0

# Submit forms
- name: "Submit Form"
  uses: browser
  with:
    action: submit
    selector: "form"
  test: res.code == 0
```

### Wait for Elements

```yaml
# Wait for element to appear
- name: "Wait for Results"
  uses: browser
  with:
    action: wait_visible
    selector: ".search-results"
    timeout: "10s"
  test: res.code == 0

# Wait for specific text
- name: "Wait for Success Message"
  uses: browser
  with:
    action: wait_text
    selector: ".status"
    value: "Success"
  test: res.code == 0
```

### Capture Screenshots

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

## Advanced Usage Examples

### Login Flow

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

### Data Extraction

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

### E2E Testing

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

## Error Handling

```yaml
- name: "Browser Action with Error Handling"
  uses: browser
  with:
    action: click
    selector: "#may-not-exist"
    timeout: "5s"
  test: res.code == 0 || (res.success == "false" && res.error | contains("not found"))
  outputs:
    click_success: res.code == 0
    error_type: |
      {{res.code == 0 ? "none" :
        res.error | contains("timeout") ? "timeout" :
        res.error | contains("not found") ? "element_not_found" :
        "unknown"}}
```

## Performance Considerations

- **Headless Mode**: Use `headless: true` (default) for faster execution
- **Timeouts**: Set appropriate timeouts to prevent hanging
- **Resource Usage**: Browser actions consume more resources than other actions
- **Screenshots**: Large screenshots consume significant memory

## Security Features

The browser action implements several security measures:

- **Sandboxed Execution**: ChromeDP runs in a sandboxed environment
- **Timeout Protection**: Prevents indefinite hanging
- **URL Validation**: Validates URLs before navigation
- **Resource Limits**: Built-in resource usage limits
