# Changelog

Release notes for each version are generated from the commits by GoReleaser.
This file records what those notes cannot carry well: changes that break code
importing probe as a library.

## v1.19.0 (2026-10-06)

### Breaking Changes

This release breaks the Go API in two narrow ways, both in what was added
since v1.18.0. Code that only calls these functions is unaffected; code that
uses them as values or compares them has to be updated.

`(*Job).RunStandalone` takes options after its other parameters, as
`...StandaloneOption`, so that a job run by the embedded action can be told
the run of the step that embeds it with `WithRunID`. A call compiles as it
did, but the method as a value has another type, so assigning it to a
variable of the old function type, or an interface declaring the old
signature, no longer compiles.

| Before | After |
|---|---|
| `func(map[string]any, *Printer, string, string) JobRun` | `func(map[string]any, *Printer, string, string, ...StandaloneOption) JobRun` |

`expr.Expr` has a field of function type, `BeforeTemplate`, which `template()`
calls with the text it is about to expand, so `Expr` is no longer
comparable: comparing two with `==`, or using one as a map key, no longer
compiles.

### Behaviour Changes

Every request of the http action has a cookie jar, so a cookie a server
sets on a redirect is sent on to where the redirect leads, as a browser
sends it; it used to be dropped. A `cookie` header written in `headers` is
still sent. The cookies set are in `res.cookies`, and are kept across steps
only with the new `keep_cookies`.

A key of a map in `with`, `vars` or a step's `vars` that holds `{{ }}` is
now evaluated as a template, as a value is; it used to be sent as written.
Two keys that come to the same key are an error.

More is hidden in the output: the values of the http action's `cookies` and
`res.cookies`, and a secret or credential as a form body percent-encodes it.
With `-v`, the http action logs the method, URL, status and headers of each
request and response, with credentials hidden, where it used to log only a
warning that they could not be encoded.

## v1.18.0 (2026-10-05)

### Breaking Changes

This release breaks the Go API in a minor version, as v1.16.0 did: code
that imports probe as a library has to be updated. Workflow files are
affected too, by the first behaviour change below: a template that cannot
be evaluated now fails its step, where its value used to be sent on with
the error written into it.

The packages that implement the built-in actions moved under `actions/`,
each into the package of the action it serves, so that the repository root
keeps the engine and its helpers. No aliases are left behind, so code that
imports them has to be updated as follows. The command line, workflow files
and external actions, which only use `actionrpc`, are unaffected.

| Before | After |
|---|---|
| `github.com/linyows/probe/http` | `github.com/linyows/probe/actions/http` |
| `github.com/linyows/probe/grpc` | `github.com/linyows/probe/actions/grpc` |
| `github.com/linyows/probe/db` | `github.com/linyows/probe/actions/db` |
| `github.com/linyows/probe/browser` | `github.com/linyows/probe/actions/browser` |
| `github.com/linyows/probe/shell` | `github.com/linyows/probe/actions/shell` |
| `github.com/linyows/probe/ssh` | `github.com/linyows/probe/actions/ssh` |
| `github.com/linyows/probe/imap` | `github.com/linyows/probe/actions/imap` |
| `github.com/linyows/probe/embedded` | `github.com/linyows/probe/actions/embedded` |
| `github.com/linyows/probe/grpc/testserver/pb` | `github.com/linyows/probe/actions/grpc/testserver/pb` |

The `mail` package served two actions, so it was split between them:

| Before | After |
|---|---|
| `mail.Send`, `mail.Req`, `mail.Res`, `mail.Result`, `mail.NewReq`, `mail.Option`, `mail.Callback`, `mail.WithBefore`, `mail.WithAfter`, `mail.Mail`, `mail.Bulk`, `mail.NewBulk`, `mail.DeliveryResult`, `mail.Client`, `mail.Dial`, `mail.NewClient`, `mail.StartTLSOff`, `mail.StartTLSAuto`, `mail.StartTLSRequired`, `mail.TLS`, `mail.OptimisticUID`, `mail.MockServer`, `mail.MockServerSession` | the same names in `github.com/linyows/probe/actions/smtp` |
| `mail.GetLatencies`, `mail.Latency`, `mail.Latencies`, `mail.ReadFirstBytes`, `mail.IsMailText`, `mail.HasFlexedPrefix`, `mail.CreateHistogramByBucket`, `mail.CreateHistogramByDuration` | the same names in `github.com/linyows/probe/actions/mail-latency` (package `maillatency`) |

`expr.EvalTemplateMap` now returns an error as well as the map, and
`expr.EvalTemplate` returns an error for a template that cannot be
evaluated. Both used to write the error into the value instead, as
`[CompileError: ...]`, `[RuntimeError: ...]` or `[EvaluationError]`, and
return no error. The error is a `*expr.TemplateError` naming the template;
`EvalTemplateMap` joins one `*expr.FieldError` for each value that failed,
naming its path, such as `headers.authorization`, and returns the map with
nil in place of each such value.

| Before | After |
|---|---|
| `m := ev.EvalTemplateMap(in, env)` | `m, err := ev.EvalTemplateMap(in, env)` |

### Behaviour Changes

A template that cannot be evaluated now fails what needs it instead of
being sent on with the error written into it. In a step's `with`, `vars`
or `name`, the step fails with the new failure kind `template` and the
action does not run; in a workflow's `vars`, the run stops before the first
job with exit status 2. A name that is not defined still reads as nil and is
not an error. `echo` shows the error, indented as its other lines are,
instead of the Go representation of the error value.

The result of a step in a job with `repeat` now carries why its first
failing iteration failed, so a JSON report keeps the kind, such as `action`
or `template`, and JUnit reports an `<error>` for it instead of an assertion
`<failure>`.

`mapping.AssignStruct` now enforces `validate:"required"` on an `int` field:
a missing key is reported as `params '<name>' is required`, as it already was
for a string, and as `mapping.MapToStructByTags` does. It used to accept a
missing required int and leave the field at zero. A value of `0` that is
given still passes. Code that relied on omitting such a field has to give it.

## v1.16.0 (2026-10-04)

### Breaking Changes

This release breaks the Go API in a minor version: code that imports probe
as a library has to be updated, while the command line and workflow files
are unaffected.

Helpers that do not depend on the workflow engine moved out of the root
`probe` package into packages of their own. No aliases are left behind, so
code that imports them has to be updated as follows.

| Before | After |
|---|---|
| `probe.Actions` | `actionrpc.Action` |
| `probe.ActionsPlugin` | `actionrpc.Plugin` |
| `probe.ActionsClient` | `actionrpc.Client` |
| `probe.ActionsServer` | `actionrpc.Server` |
| `probe.Handshake` | `actionrpc.Handshake` |
| `probe.PluginMap` | `actionrpc.PluginMap` |
| `probe.BuiltinCmd` | `actionrpc.BuiltinCmd` |
| `probe.LogActionParams` | `actionrpc.LogParams` |
| `probe.LogActionOutcome` | `actionrpc.LogOutcome` |
| `Workflow.RenderDagAscii()` | `dag.ASCII{}.Render(w.Graph())` |
| `Workflow.RenderDagMermaid()` | `dag.Mermaid{}.Render(w.Graph())` |
| `probe.Expr` | `expr.Expr` |
| `probe.Report` | `report.Report` |
| `probe.ReportSummary` | `report.Summary` |
| `probe.ReportCount` | `report.Count` |
| `probe.JobReport` | `report.Job` |
| `probe.StepReport` | `report.Step` |
| `probe.RetryReport` | `report.Retry` |
| `probe.RepeatReport` | `report.Repeat` |
| `probe.FailureReport` | `report.Failure` |
| `probe.ReportPassed`, `ReportFailed`, `ReportSkipped`, `ReportUntested` | `report.Passed`, `Failed`, `Skipped`, `Untested` |
| `probe.ReportFormat` | `report.Format` |
| `probe.ReportJSON`, `ReportJUnit`, `ReportMarkdown`, `ReportGitHubSummary` | `report.JSON`, `JUnit`, `Markdown`, `GitHubSummary` |
| `probe.ReportTarget` | `report.Target` |
| `probe.ParseReportTargets` | `report.ParseTargets` |
| `probe.ErrNoStepSummary` | `report.ErrNoStepSummary` |
| `probe.ErrStepSummaryFull` | `report.ErrStepSummaryFull` |
| `probe.MatchJSON` | `jsonutil.Match` |
| `probe.DiffJSON` | `jsonutil.Diff` |
| `probe.ParseJSON` | `jsonutil.Parse` |
| `probe.Masker` | `mask.Masker` |
| `probe.NewMasker` | `mask.New` |
| `probe.TruncateString` | `truncate.String` |
| `probe.TruncateMapStringString` | `truncate.MapString` |
| `probe.TruncateMapStringAny` | `truncate.Map` |
| `probe.GetTruncationMessage` | `truncate.Message` |
| `probe.MaxLogStringLength` | `truncate.MaxLogLength` |
| `probe.MaxStringLength` | `truncate.MaxLength` |

`Printer.SetMasker`, `Printer.Masker`, `Report.Mask` and `RunOptions.Masker`
now take or return a `*mask.Masker`, and `Step.Expr` is a `*expr.Expr`.
`BuildReport` stays in the root package and returns a `*report.Report`, and
`Config.Reports` is a `[]report.Target`.

`probe.ActionsArgs` and `probe.ActionsParams` are removed; nothing used them.
An action is now served with `actionrpc.Serve`, which builds the logger and
the plugin that each action used to build for itself.

`probe.DagRendererBase`, `probe.DagAsciiRenderer`, `probe.DagAsciiJobNode`,
`probe.DagMermaidRenderer` and their constructors are removed. The renderers
are now `dag.ASCII` and `dag.Mermaid`, which draw the `dag.Graph` that
`Workflow.Graph` returns.

The new packages are imported from `github.com/linyows/probe/actionrpc`,
`github.com/linyows/probe/expr`, `github.com/linyows/probe/jsonutil`,
`github.com/linyows/probe/mask`, `github.com/linyows/probe/report` and
`github.com/linyows/probe/truncate`, and the renderers from the existing
`github.com/linyows/probe/dag`. Two more new packages, `safefile` and
`procgroup`, hold what used to be unexported in the root package.
