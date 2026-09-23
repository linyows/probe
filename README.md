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

Features
--------

Similar software is usually one of two things: a test runner, or a prober that monitors. Most of them speak one protocol. Probe is both, across the protocols a web system is actually made of.

- **One file reaches the whole system.** HTTP, gRPC, MySQL, PostgreSQL, SQLite, SMTP, IMAP, SSH, shell and a real browser are built in. A single run can call an endpoint, query the row it wrote, and read the mail it sent, with no glue between three tools.
- **The same file is a test and a monitor.** Add `repeat` to a job and it runs on an interval, reporting `3/3 success (100.0%)`. There is no second, rewritten copy of the workflow for monitoring.
- **Jobs are a graph, not a list.** `needs` declares only the order that matters, and everything else runs in parallel. `probe dag` prints that graph as ASCII or Mermaid. Scenario runners walk a file from top to bottom.
- **It behaves the same everywhere.** One Go binary, nothing to install alongside it and no service to keep running, so your terminal, a CI step and a cron entry all do the same thing.
- **Actions are plugins.** Each one is a separate process behind [go-plugin](https://github.com/hashicorp/go-plugin), so a protocol Probe does not cover yet can be added without forking the binary.

Beyond that, a workflow has `outputs` to pass data between steps and jobs, `test` to assert on any response, `retry`, `skipif`, `iteration`, `wait` and `timeout`, `defaults` for settings shared across steps, and merging of several YAML files on one command line.

[Understanding Probe](https://probe.linyo.ws/guide/introduction/understanding-probe) covers how these fit together, and [Comparison](https://probe.linyo.ws/guide/introduction/comparison) says where k6, Hurl, Venom, runn, Blackbox exporter or GitHub Actions is the better answer.

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
