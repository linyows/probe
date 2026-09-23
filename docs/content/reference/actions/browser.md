# Browser Action

The `browser` action automates web browsers using ChromeDP, providing comprehensive web automation capabilities for testing, scraping, and interaction with web applications.

## Basic Syntax

A browser step names the operation in `action` and gives it whatever that operation needs.

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

`action` decides which browser operation runs, and the rest of the parameters supply what that operation needs: the target, the value to type, and how the browser is launched.

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

Every browser action returns these two fields, whatever it did.

| Property | Type | Description |
|----------|------|-------------|
| `code` | Integer | Result code (0 = success, non-zero = error) |
| `results` | Object | Action-specific results (text, values, etc.) |

### Navigation Response

`navigate` reports where it ended up and how long the page took.

| Property | Type | Description |
|----------|------|-------------|
| `url` | String | URL that was navigated to |
| `time_ms` | String | Navigation time in milliseconds |

### Text/Attribute Response

`text` and `get_attribute` report the selector they used along with what they read.

| Property | Type | Description |
|----------|------|-------------|
| `selector` | String | CSS selector used |
| `text` | String | Extracted text content (get_text) |
| `attribute` | String | Attribute name (get_attribute) |
| `value` | String | Attribute value (get_attribute) |
| `exists` | String | "true" if attribute exists |

### Screenshot Response

`screenshot` returns the image itself, encoded as Base64.

| Property | Type | Description |
|----------|------|-------------|
| `screenshot` | String | Base64-encoded screenshot |
| `size_bytes` | String | Screenshot size in bytes |

## Browser Actions

Each `action` value is shown below with the parameters it reads and what it returns.

### Navigate to URL

`navigate` opens a page and is the step every other browser action depends on.

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

`text` reads the text of the element the selector matches, which can then be asserted on or passed along in `outputs`.

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

`get_attribute` reads one named attribute rather than the element's text.

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

Filling a form takes one step per action: typing into a field, clicking a control, and submitting.

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

A page that renders after loading needs an explicit wait before the next step can address it.

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

`screenshot` records what the page looked like at that point in the run.

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

A single action rarely stands alone. The workflows below chain several steps together and carry state between them through `outputs`.

### Login Flow

Logging in is a sequence: open the form, type the credentials, submit, and confirm the result.

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

Navigating and then reading several elements turns a page into values the rest of the workflow can use.

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

An end-to-end test drives the application the way a user would and asserts on what the page shows.

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

A selector that matches nothing returns a non-zero `code`, which the step's test and `continue_on_error` decide what to do with.

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
