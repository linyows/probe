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

export const installCommand = 'go install github.com/linyows/probe/cmd/probe@latest'

export const sourceCommands = `git clone https://github.com/linyows/probe.git
cd probe
go build -o probe ./cmd/probe`
