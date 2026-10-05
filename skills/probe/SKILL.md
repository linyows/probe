---
name: probe
description: Write, run and debug Probe workflows, the YAML files the `probe` CLI runs to test and monitor HTTP APIs, gRPC services, databases, mail, SSH hosts, shell commands and browsers. Use when creating or changing a Probe workflow, when a `probe` run fails, or when asked to add an API test, a health check or a monitor with Probe.
---

# Probe

Probe runs a YAML workflow: jobs that run in parallel unless they declare `needs`, each a list of steps that call an action and check the result with a `test` expression. The same file serves as a test in CI and, with `repeat`, as a monitor.

## Read the docs from the binary first

The installed `probe` prints its own documentation, matching its version. Read the pages you need before writing YAML instead of relying on memory:

```bash
probe guide                       # list the topics
probe guide yaml                  # every workflow, job and step field
probe guide http                  # one action's parameters and response (also: shell, db, grpc, ssh, ...)
probe guide concepts/expressions  # expressions, templates and the functions available
probe guide cli                   # flags, reports, exit codes
```

Before writing a new workflow, look for existing ones in the repository and follow their layout and naming.

## A workflow

```yaml
name: API check
secrets:
  - API_TOKEN                    # its value never appears in output or reports
vars:
  api_url: "{{API_URL ?? 'http://localhost:8080'}}"
  token: "{{API_TOKEN}}"
jobs:
- name: Users API
  id: users
  defaults:
    http:
      url: "{{vars.api_url}}"
      headers:
        authorization: "Bearer {{vars.token}}"
  steps:
  - name: Create a user
    id: create
    uses: http
    with:
      post: /users
      headers:
        content-type: application/json
      body:
        name: probe
    test: res.code == 201 && res.body.name == "probe"
    outputs:
      user_id: res.body.id
  - name: Read it back
    uses: http
    with:
      get: "/users/{{outputs.create.user_id}}"
    test: res.code == 200
    echo: "fetched {{res.body.name}}"
- name: After the users job
  needs: [users]
  steps:
  - name: Hello
    uses: hello
```

## Rules that are easy to get wrong

- `jobs` and `steps` are lists. `needs` lists job `id`s, so a job others depend on needs an `id`.
- Environment variables are read only in the top-level `vars`, by bare name: `"{{API_URL}}"`, with a fallback as `"{{API_URL ?? 'default'}}"`. Expressions in steps see `vars`, `res`, `req`, `rt`, `status`, `outputs` and `repeat_index`, and nothing else; there is no `env`.
- `test` and `outputs` values are expressions, written without braces. `with` values, `echo`, step and job names, and `vars` are templates with `{{ }}`.
- `test` must evaluate to a boolean. A number or string fails the step with `test_type`.
- `outputs` need the step's `id`. Read them as `outputs.<step-id>.<name>`; an id with a hyphen needs brackets: `outputs['create-user'].user_id`.
- Templates are evaluated in values, never in keys: `"{{vars.name}}": x` sends the literal key.
- A template ends at the `}}` that closes it: braces of a map literal nest, and `}}` inside a quoted string does not end it. Start a template that begins with a map literal with a space, `{{ {'a': 1} }}`, since `{{{` is read as a literal `{` followed by a template. A `{{` that is never closed is left as text.
- Defaults shared by the steps of a job go in that job's `defaults`, keyed by action name. There is no top-level `defaults` or `env`.
- List secrets under `secrets` by environment variable name. Never write a credential into the YAML.
- Do not guess parameters. If a field is not on the action's `probe guide <action>` page, it does not exist.

## Run

```bash
probe workflow.yml                    # run; the report goes to stdout
probe base.yml,staging.yml            # merge files, later ones override
probe dag workflow.yml                # check the job graph without running
probe -v workflow.yml                 # also print every request and response
probe --report json=probe.json,markdown=probe.md workflow.yml
```

The exit code says what to look at:

| Code | Meaning | Where to look |
|---|---|---|
| `0` | Every job succeeded | |
| `1` | A `test` was false, could not be evaluated, or was not a boolean, or a step's template could not be evaluated | The response, or the test expression or template |
| `2` | The workflow or the command line is wrong | The `[ERROR]` line on stderr: YAML, an unknown `needs`, a step id, a flag |
| `3` | An action returned an error, such as a refused connection or a timeout | Whether the target is reachable, and the action's parameters |

## Debug a failure

1. Rerun with `--report markdown=probe-report.md` and read that file. Each failed step has its test, the failure kind, and the request and response it saw.
2. Act on the kind:
   - `assertion`: the test was false. Compare the response with what the test expects; fix whichever is wrong.
   - `test_error` or `test_type`: the expression itself is wrong. Check field names against `probe guide <action>` and the syntax against `probe guide concepts/expressions`.
   - `template`: a template in the step's `with`, `vars` or `name` could not be evaluated, so the action did not run. The message names the value, as `with.headers.authorization`. Usually an earlier step did not publish the output it reads; check that step, or fall back with `?.` and `??`.
   - `action`: the action could not run. Check the URL or host, credentials, and `timeout`.
3. For more detail, run with `-v`, which prints every request and response. Declared secrets and credential headers are masked there too.
4. After a fix, run the workflow again and confirm the exit code is `0` before calling it done.
