# Hello Action

The `hello` action does nothing but succeed. It is useful as a placeholder, for a step whose only job is an `echo`, and for trying out expressions.

## Basic Syntax

A hello step takes `echo`, and anything passed in `with` comes back in `res`.

```yaml
steps:
  - name: Report
    uses: hello
    echo: "Checked {{outputs.health.endpoint}}"
```

## Parameters

The action takes no parameters of its own. Whatever is given in `with` is echoed back on `res`, which makes it a convenient way to publish computed values.

```yaml
steps:
  - name: Build a summary
    id: summary
    uses: hello
    with:
      run_id: "{{vars.run_id}}"
      checked_at: "{{now().Format('2006-01-02T15:04:05Z07:00')}}"
    outputs:
      run_id: res.run_id
      checked_at: res.checked_at
```

## Response Object

The step always succeeds, and whatever was passed in comes back unchanged.

| Property | Type | Description |
|----------|------|-------------|
| `res.<key>` | Any | Every key passed in `with` |
| `res.status` | Integer | Always `0` |
| `status` | Integer | Always `0` |
