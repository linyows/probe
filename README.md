<p align="right">English | <a href="https://github.com/linyows/probe/blob/main/README.ja.md">日本語</a></p>

<br><br><br><br><br><br>

<p align="center">
  <img alt="PROBE" src="https://github.com/linyows/probe/blob/main/misc/probe.svg" width="200">
</p>

<br><br><br><br><br><br>

<p align="center">
  <a href="https://github.com/linyows/probe/actions/workflows/build.yml">
    <img alt="GitHub Workflow Status" src="https://img.shields.io/github/actions/workflow/status/linyows/probe/build.yml?branch=main&style=for-the-badge&labelColor=666666">
  </a>
  <a href="https://github.com/linyows/probe/releases">
    <img src="http://img.shields.io/github/release/linyows/probe.svg?style=for-the-badge&labelColor=666666&color=DDDDDD" alt="GitHub Release">
  </a>
  <a href="http://godoc.org/github.com/linyows/probe">
    <img src="http://img.shields.io/badge/go-docs-blue.svg?style=for-the-badge&labelColor=666666&color=DDDDDD" alt="Go Documentation">
  </a>
  <a href="https://deepwiki.com/linyows/probe">
    <img src="http://img.shields.io/badge/deepwiki-docs-purple.svg?style=for-the-badge&labelColor=666666&color=DDDDDD" alt="Deepwiki Documentation">
  </a>
</p>

Probe runs a YAML workflow against your HTTP APIs, databases, mail servers, browsers and shells, checks every response, and prints a report you can read.

It is a single Go binary with no runtime to install, so the same file runs on your machine, in CI, and from a cron entry. The exit status reflects the result, and a file written as a test becomes a monitor by adding `repeat`.

**Documentation: [probe.linyo.ws](https://probe.linyo.ws/)**

![Architecture](/misc/probe-architecture.svg)

Quick Start
-----------

Install the binary:

```bash
go install github.com/linyows/probe/cmd/probe@latest
```

Write a workflow:

```yaml
# health-check.yml
name: API Health Check
jobs:
- name: Check API Status
  steps:
  - name: Ping API
    uses: http
    with:
      url: https://api.example.com
      get: /health
    test: res.code == 200
```

Run it:

```bash
probe health-check.yml
```

```
API Health Check

⏺ Check API Status (Completed in 0.02s)
  ⎿ 0. ✓  Ping API

Total workflow time: 0.02s ✓ All jobs succeeded
```

[Quickstart](https://probe.linyo.ws/guide/introduction/quickstart) goes through this step by step, and [Your First Workflow](https://probe.linyo.ws/guide/introduction/your-first-workflow) grows it into something you can leave running.

What Probe Does
---------------

- **One file, several protocols.** A workflow can call an HTTP endpoint, query the database behind it and check the mail it sent, in the same run.
- **Jobs run in parallel.** `needs` declares the order where it matters, and the rest runs at once. `probe dag` prints the resulting graph.
- **Steps share data.** A step publishes `outputs` that later steps and jobs read.
- **Every step asserts.** `test` is an expression over the response, so a workflow is a test rather than a script that happens to succeed.
- **Tests become monitors.** `repeat` runs a job on an interval and reports how many passes succeeded.
- **The awkward cases are covered.** `retry`, `skipif`, `iteration`, `wait` and `timeout` handle what does not pass first time, does not apply everywhere, or must not run forever.
- **Extensible.** Actions are plugins served over gRPC, so an action you need but Probe does not have is one you can add.

See [Understanding Probe](https://probe.linyo.ws/guide/introduction/understanding-probe) for how these fit together, and [Comparison](https://probe.linyo.ws/guide/introduction/comparison) for where another tool is the better answer.

Built-in Actions
----------------

| Action | Use it for |
|---|---|
| [`http`](https://probe.linyo.ws/reference/actions/http) | HTTP requests and their responses |
| [`db`](https://probe.linyo.ws/reference/actions/db) | Queries against MySQL, PostgreSQL and SQLite |
| [`smtp`](https://probe.linyo.ws/reference/actions/smtp) | Delivering mail |
| [`imap`](https://probe.linyo.ws/reference/actions/imap) | Reading a mailbox |
| [`mail-latency`](https://probe.linyo.ws/reference/actions/mail-latency) | Measuring delivery latency from received messages |
| [`ssh`](https://probe.linyo.ws/reference/actions/ssh) | Commands on a remote host |
| [`shell`](https://probe.linyo.ws/reference/actions/shell) | Commands on the machine running Probe |
| [`browser`](https://probe.linyo.ws/reference/actions/browser) | Driving a real browser |
| [`grpc`](https://probe.linyo.ws/reference/actions/grpc) | gRPC calls |
| [`embedded`](https://probe.linyo.ws/reference/actions/embedded) | Running another job file from a step |
| [`hello`](https://probe.linyo.ws/reference/actions/hello) | Printing a report line |

Documentation
-------------

- [Guide](https://probe.linyo.ws/guide) — installation, the CLI, and the concepts behind workflows, jobs, steps and data flow
- [How-tos](https://probe.linyo.ws/guide/how-tos/api-testing) — API testing, monitoring, performance, environments and error handling
- [Tutorials](https://probe.linyo.ws/guide/tutorials/first-monitoring-system) — building a monitoring system, an API test pipeline, and multi-environment tests
- [Reference](https://probe.linyo.ws/reference) — YAML configuration, CLI options, built-in functions and environment variables

Runnable workflows live in [`examples/`](./examples/).

Contributing
------------

Issues, feature requests and pull requests are all welcome.

License
-------

MIT. See [LICENSE](./LICENSE).

Author
------

[linyows](https://github.com/linyows)
