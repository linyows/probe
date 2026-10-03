# Browser Action

The `browser` action drives a real Chrome through [chromedp](https://github.com/chromedp/chromedp): it opens pages, reads and types into them, waits for elements, and takes screenshots. Chrome or Chromium has to be installed where Probe runs.

## Basic Syntax

A browser step lists what to do under `actions`, in order. Every step starts a fresh browser, runs its actions, and closes it, so cookies and a login do not carry over to the next step: put a whole flow in one step.

```yaml
steps:
  - name: Read the heading
    uses: browser
    with:
      actions:
        - name: navigate
          url: "{{vars.url}}/"
        - name: text
          id: heading
          selector: h1
    test: res.code == 0 && res.results.heading == "Welcome"
```

## Parameters

These go directly under `with`.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `actions` | List | required | The actions to run, in order. See [Actions](#actions) |
| `timeout` | Duration string | `5s` | Limit for the whole step: starting the browser and every action. Write it as a string such as `30s`; a plain number is ignored and the default applies |
| `headless` | Boolean | `true` | Run Chrome without a window |
| `window_w` | Integer | `1920` | Window width in pixels |
| `window_h` | Integer | `1080` | Window height in pixels |
| `evidence_dir` | String | the temporary directory | Where the page is saved when the actions fail. See [Page at failure](#page-at-failure) |

Starting Chrome takes part of `timeout`, often a second or more, so a short limit can run out before the first page opens.

## Actions

Each entry of `actions` names its action in `name` and takes these fields:

| Field | Used by | Description |
|-------|---------|-------------|
| `name` | all | The action, from the tables below |
| `id` | actions that return something | The key for the result in `res.results` or `res.filepaths`. Without it the action's name is the key, so a later action of the same kind replaces the earlier result |
| `url` | `navigate` | The address to open |
| `selector` | actions on an element | A CSS selector |
| `value` | `type`, `send_keys`, `select` | The text to use |
| `attribute` | `get_attribute` | The attribute to read, as a list: `[href]`. Only the first entry is read, and a plain string is rejected |
| `quality` | `full_screenshot` | `1` to `99` saves JPEG at that quality; unset or `100` saves PNG |
| `path` | screenshots | Deprecated: also write the image to this path. Use `res.filepaths` instead |

An action that looks for an element waits until the element appears, and the step fails when `timeout` runs out first. Most of them also wait for it to be visible.

### Navigation and waiting

These open a page or hold the step until the page is in the state the next action needs.

| Action | What it does |
|--------|--------------|
| `navigate` | Opens `url` and waits for the page to load. An address that cannot be reached fails at once |
| `wait_visible` | Waits until `selector` is visible |
| `wait_not_visible` | Waits until `selector` is not visible |
| `wait_enabled` | Waits until `selector` is visible and enabled |
| `wait_ready` | Waits until the page's `body` is ready. `selector` is ignored |

### Reading the page

These put a string in `res.results`, under `id` or the action's name.

| Action | Result |
|--------|--------|
| `text` | The text content of `selector` |
| `wait_text` | Waits until `selector` is visible, then its text content |
| `value` | The value of a form field, such as an `input` |
| `get_attribute` | The first attribute in `attribute` of `selector`, or an empty string when the element does not have it |
| `get_html` | The outer HTML of `selector`, once it is visible |

### Interacting

These act on the page the way a user would. They return nothing.

| Action | What it does |
|--------|--------------|
| `click` | Clicks `selector` once it is visible |
| `double_click` | Double-clicks `selector` once it is visible |
| `type`, `send_keys` | Clears `selector`, then types `value` into it |
| `submit` | Submits the form `selector` belongs to |
| `focus` | Focuses `selector` |
| `scroll` | Scrolls `selector` into view |
| `select` | Sets the `value` attribute of `selector` to `value`. It does not choose an option of a `<select>`; click the option instead |
| `hover` | Sends a `mouseover` event to `selector`. It does not wait, and does nothing when nothing matches |
| `right_click` | Sends a `contextmenu` event to `selector`. It does not wait, and does nothing when nothing matches |

### Screenshots

These save an image file and put its path in `res.filepaths`, under `id` or the action's name. The file is named for its format, `.png` or `.jpg`.

| Action | Image |
|--------|-------|
| `capture_screenshot` | The visible part of the window, as PNG |
| `full_screenshot` | The whole page, as PNG, or as JPEG when `quality` is `1` to `99` |
| `screenshot` | Only `selector`, once it is visible, as PNG |

## Response Object

When every action succeeds, the step sees:

| Field | Type | Description |
|-------|------|-------------|
| `res.code` | Integer | `0` |
| `res.results` | Object | What the reading actions returned, by `id` or action name |
| `res.filepaths` | Object | Where the screenshots were saved, by `id` or action name |
| `rt.duration` | String | How long the step took, such as `"663.268333ms"` |
| `status` | Integer | `0` |

When an action fails there is no response to test: the step fails with an action error, as described in [Error Handling](#error-handling).

## Examples

Each example is one step, since a step's actions share one browser and the next step starts a new one.

### Logging in

A login and the check that it worked have to share one browser, so they are one step.

```yaml
secrets:
  - PASSWORD
vars:
  url: "{{APP_URL}}"
  user: "{{USERNAME}}"
  password: "{{PASSWORD}}"

jobs:
- name: Sign in
  steps:
  - name: Log in and reach the dashboard
    uses: browser
    with:
      timeout: 30s
      actions:
        - name: navigate
          url: "{{vars.url}}/login"
        - name: type
          selector: "#username"
          value: "{{vars.user}}"
        - name: type
          selector: "#password"
          value: "{{vars.password}}"
        - name: click
          selector: "button[type='submit']"
        - name: wait_visible
          selector: "#dashboard"
        - name: text
          id: greeting
          selector: "#dashboard h1"
    test: res.code == 0 && res.results.greeting contains vars.user
```

### Reading several values

Give each reading action an `id` so the results do not replace each other.

```yaml
  - name: Product page
    uses: browser
    with:
      actions:
        - name: navigate
          url: "{{vars.url}}/products/42"
        - name: text
          id: title
          selector: "h1"
        - name: text
          id: price
          selector: ".price"
        - name: get_attribute
          id: image
          selector: "img.product"
          attribute: [src]
    test: |
      res.code == 0 &&
      res.results.title != "" &&
      res.results.price startsWith "$"
    outputs:
      image_url: res.results.image
```

### Keeping a screenshot

The path of the saved image is in `res.filepaths`, ready to print or pass on.

```yaml
  - name: Checkout page
    uses: browser
    with:
      window_w: 1280
      window_h: 800
      actions:
        - name: navigate
          url: "{{vars.url}}/checkout"
        - name: full_screenshot
          id: page
    test: res.code == 0
    echo: "Saved {{res.filepaths.page}}"
```

## Error Handling

An action that cannot finish makes the whole step fail with an action error, which exits the run with status `3`. A selector that matches nothing is not an empty result: `wait_visible`, `text`, `click` and the other actions that look for an element keep waiting until the step's `timeout`. A navigation that cannot reach its URL fails at once. In both cases there is no `res` to test, and the failure message carries the browser's error.

### Page at failure

When the actions fail, Probe saves the page as it was at that moment: a full-page PNG screenshot and the HTML of the document, under one shared name such as `probe-browser-failure-1947177907.png` and `.html`. The failure message names both files along with the page's URL:

```
action error in step_execute: action execution failed (caused by: ... context deadline exceeded (page at failure: url http://localhost:8090/, screenshot /tmp/probe-browser-failure-1947177907.png, html /tmp/probe-browser-failure-1947177907.html))
```

The same message is the failure's `message` in the files `--report` writes. A test that evaluates to false is not an action failure, so the page is not saved then; add a `capture_screenshot` action to keep it.

The files go to the system's temporary directory unless `evidence_dir` names another. The directory is created when needed, which suits uploading it as a CI artifact. The page can show private data, so the files are readable only by the user who ran Probe:

```yaml
- name: Checkout page
  uses: browser
  with:
    evidence_dir: out/browser
    actions:
    - name: navigate
      url: "{{vars.url}}/checkout"
    - name: wait_visible
      selector: "#pay"
```

The browser is kept for up to 10 seconds after the timeout so that the page can still be read. If it cannot be, for example because the browser never started, the message says so and nothing is saved.

## Running in CI

A CI runner needs a browser and a little more time than a laptop.

- Install Chrome or Chromium on the runner.
- Keep `headless: true`, or provide a display such as Xvfb when a window is needed.
- Chrome is started with `--no-sandbox`, as containers usually require, so point it only at sites you trust.
- Give `timeout` room for Chrome to start, which is slower on a cold runner.
