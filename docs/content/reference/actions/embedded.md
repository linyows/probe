# Embedded Action

The `embedded` action runs a **job file** as a single step, which lets shared setup or checks live in their own file and be reused from several workflows.

The file is a job, not a workflow: it holds `name`, `steps` and optionally `defaults` - there is no `jobs` key in it.

## Basic Syntax

**auth.yml:**
```yaml
name: Authentication
steps:
  - name: Get token
    id: get_token
    uses: shell
    with:
      cmd: echo "0123456789"
    test: res.code == 0
    outputs:
      mytoken: replace(res.stdout, '\n', '')
```

**workflow.yml:**
```yaml
jobs:
- name: Main
  steps:
    - name: Authenticate
      id: auth
      uses: embedded
      with:
        path: "./auth.yml"
        vars:
          environment: "{{vars.environment}}"
      test: res.code == 0
      outputs:
        token: res.outputs.mytoken

    - name: Call the API
      uses: http
      with:
        method: GET
        url: "{{vars.api_url}}/me"
        headers:
          authorization: "Bearer {{outputs.auth.token}}"
      test: res.code == 200
```

## Parameters

An embedded step names the job file to run and the variables to hand it.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `path` | String | Yes | - | Path to the job file. Resolved against the current working directory, not the workflow file |
| `vars` | Object | No | `{}` | Variables passed to the embedded job, read there as `vars.<name>` |

## Response Object

After the embedded job finishes, `res` carries its result and its outputs.

| Field | Type | Description |
|-------|------|-------------|
| `res.code` | Integer | `0` when every step of the embedded job passed |
| `res.outputs` | Object | The outputs published by the embedded job's steps, keyed by output name |
| `res.report` | String | The embedded job's report, which is also nested into the parent report |
| `res.error` | String | Error message, when the embedded job failed |
| `rt` | Object | Time spent running the embedded job |

`res.outputs` is keyed by output name, so a value published as `mytoken` is read as `res.outputs.mytoken`.
