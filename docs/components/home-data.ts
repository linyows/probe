/**
 * Every terminal block below is the literal output of the command shown with
 * it, captured from a real `probe` run of the workflow in `heroWorkflow`:
 * a stub API that sends the confirmation mail, SQLite, and a GreenMail IMAP
 * server. Only the connection details in the workflow were swapped for
 * presentable ones — the API host, the database path, the IMAP host and the
 * password, which is an env var here as it would be anywhere. Keep it that way.
 */

export type ReportLine = {
  text: string
  tone?: 'title' | 'job' | 'pass' | 'idle' | 'echo' | 'total'
}

export const heroWorkflow = `name: Sign-up Flow

vars:
  api: https://api.example.com
  mail_user: ada@example.com
  mail_pass: "{{MAIL_PASS}}"

jobs:
- name: Create account
  id: signup
  defaults:
    http:
      url: "{{vars.api}}"
  steps:
  - name: Post the form
    id: create
    uses: http
    with:
      post: /signup
    test: res.code == 201 && res.body.status == "pending"
    outputs:
      user_id: res.body.id

- name: Database
  id: db
  needs: [signup]
  defaults:
    db:
      dsn: file:./app.db
  steps:
  - name: Row is pending
    uses: db
    with:
      query: SELECT status FROM users WHERE id = ?
      params: ["{{outputs.create.user_id}}"]
    test: res.code == 0 && res.rows[0].status == "pending"

- name: Mailbox
  id: mailbox
  needs: [signup]
  defaults:
    imap:
      host: imap.example.com
      username: "{{vars.mail_user}}"
      password: "{{vars.mail_pass}}"
  steps:
  - name: Confirmation arrived
    uses: imap
    with:
      commands:
      - name: select
        mailbox: INBOX
      - name: fetch
        sequence: "1"
        dataitem: ALL
    test: |
      res.code == 0 &&
      res.data.fetch.messages[0].subject == "Confirm your address" &&
      res.data.fetch.messages[0].to == vars.mail_user

- name: Activate
  needs: [db, mailbox]
  defaults:
    http:
      url: "{{vars.api}}"
  steps:
  - name: Follow the link
    uses: http
    with:
      get: "/users/{{outputs.create.user_id}}"
    test: res.code == 200 && res.body.status == "active"
`

export const heroReport: ReportLine[] = [
  { text: 'Sign-up Flow', tone: 'title' },
  { text: '' },
  { text: '⏺ Create account (Completed in 0.11s)', tone: 'job' },
  { text: '  ⎿ 0. ✓  Post the form', tone: 'pass' },
  { text: '' },
  { text: '⏺ Database (Completed in 0.02s)', tone: 'job' },
  { text: '  ⎿ 0. ✓  Row is pending', tone: 'pass' },
  { text: '' },
  { text: '⏺ Mailbox (Completed in 0.03s)', tone: 'job' },
  { text: '  ⎿ 0. ✓  Confirmation arrived', tone: 'pass' },
  { text: '' },
  { text: '⏺ Activate (Completed in 0.01s)', tone: 'job' },
  { text: '  ⎿ 0. ✓  Follow the link', tone: 'pass' },
  { text: '' },
  { text: 'Total workflow time: 0.15s ✓ All jobs succeeded', tone: 'total' },
]

/*
 * The pair below was run the same way: `probe orders.yml` against a stub that
 * answers POST /login with a token and refuses GET /orders without it. Only
 * the API host was swapped for a presentable one.
 */

export const sharedWorkflow = `name: Orders

vars:
  api: https://api.example.com

jobs:
- name: Order history
  steps:
  - name: Log in
    id: auth
    uses: embedded
    with:
      path: ./login-job.yml
      vars:
        api: "{{vars.api}}"
        email: ada@example.com
        password: "{{PASSWORD}}"
    test: res.code == 0
    outputs:
      token: res.outputs.token

  - name: List orders
    uses: http
    with:
      url: "{{vars.api}}"
      get: /orders
      headers:
        authorization: "Bearer {{outputs.auth.token}}"
    test: res.code == 200`

export const sharedJob = `name: Log in

steps:
- name: Post credentials
  id: login
  uses: http
  with:
    url: "{{vars.api}}"
    post: /login
    body:
      email: "{{vars.email}}"
      password: "{{vars.password}}"
  test: res.code == 200
  outputs:
    token: res.body.access_token`

export const installCommand = 'go install github.com/linyows/probe/cmd/probe@latest'

export const sourceCommands = `git clone https://github.com/linyows/probe.git
cd probe
go build -o probe ./cmd/probe`
