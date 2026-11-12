# Runner terminal

Admins can open an interactive shell on a runner from the Runner page. It
opens in a floating window, so a runner can be inspected (`docker ps`, logs,
disk space) while a task on it looks stuck, without SSH access to the runner.

The terminal is off by default, on the controller and on every runner. A
shell on a runner is powerful (see [Security](#security)), so enable it only
where you need it.

## How it works

```text
                     1. POST /api/admin/runner/terminal/ticket  (admin JWT)
browser ───────────────────────────────────────────────►  controller
        ◄─────────────────────── ticket (one-time, 30 s)
                     2. WebSocket /api/terminal/ws  (ticket)
browser ═══════════════════════════════════════════════►  controller
                                                              ║ 3. "open"
runner  ═══ WebSocket /api/run/runner/terminal/control ═══════╝   (always open)
runner  ═══ WebSocket /api/run/runner/terminal/session?id= ══►  controller
        4. dials back for the session, attaches a new PTY
                     5. the controller copies bytes between the two sockets
```

- A runner with the terminal enabled keeps one outbound control WebSocket to
  the controller, next to its heartbeat. The runner never listens on a port,
  and the controller never connects to a runner or holds a runner credential.
- For each session the controller sends `open` on the control channel. The
  runner starts the shell on a new PTY and dials back a data WebSocket for
  that session. The controller then relays between the browser's socket and
  the runner's socket.
- WebSocket paths are under the same base URL as the runner's other API
  calls, so a runner with an `https://` `controller_url` uses `wss`.

### Protocol

- Subprotocol `it-terminal.v1`. The browser sends the ticket as a second
  subprotocol, `ticket.<id>`: browsers cannot set headers on a WebSocket, and
  this keeps the ticket out of URLs and access logs.
- Binary frames carry terminal bytes. Text frames carry JSON control
  messages: `resize` (browser → runner), `ready`, `exit` and `error`
  (→ browser); `hello`, `open` and `rejected` on the control channel.
- Close codes sent to the browser:

  | Code | Meaning |
  | --- | --- |
  | 4403 | not allowed: the admin role was removed, or the runner refused the session |
  | 4408 | idle timeout or session time limit |
  | 4409 | runner disconnected |
  | 4429 | session limit reached |
  | 4504 | runner did not answer the open request |

## Security

### Who can open a terminal

- Accounts with the admin role (the same rule as login), optionally narrowed
  with `terminal.allowedUsers`. The role is checked when the ticket is issued,
  when the socket opens, and again every minute; a session also ends when the
  admin's JWT expires.
- Tickets are 32 random bytes, valid once, for 30 seconds, for one runner,
  and only from the client address that asked for them. Unused tickets are
  capped per user and in total.
- The browser must send an `Origin` header that matches the controller's own
  host or `terminal.allowedOrigins`, so another site cannot open a terminal
  with an admin's browser.
- `terminal.allowedClientCIDRs` can limit where browsers connect from.

### HTTPS

With `terminal.allowInsecureHttp: false` (the default), a browser must reach
the controller over HTTPS through a reverse proxy listed in
`backend.trustedProxies` ([HTTPS](../deb/https.md)). The controller serves
plain HTTP itself, so it believes `X-Forwarded-Proto: https` only from those
proxies. Without any trusted proxy the controller does not start the
terminal.

`allowInsecureHttp: true` allows plain HTTP. Then the admin's JWT and
everything typed in the terminal cross the network unencrypted; use it only
over a VPN.

This setting covers browsers. The runner's own connection uses the runner's
configuration: `controller_url: https://...`, or plain HTTP over the network
the runner uses to reach `controller_ip` (keep that a VPN).

### Runner identity

The runner signs in with its runner JWT, like its heartbeat. Its token must
also carry the id of the runner's current registration (the `rid` claim);
deleting and registering a runner again makes its old tokens useless for the
terminal. Tokens issued before this feature have no `rid`: register such a
runner again before enabling its terminal (see [Runner](#runner)). When a
control channel replaces an older one, the controller logs both addresses.

### Startup checks

The controller refuses to start the terminal (and logs why) when:

- `backend.jwt.secret` or `backend.runnerJwt.secret` is the default from this
  repository, shorter than 32 characters, or both are the same;
- `backend.password` is the default;
- neither HTTPS through a trusted proxy nor `allowInsecureHttp` is set, or a
  trusted proxy entry covers every address (`0.0.0.0/0`);
- `terminal.ticketTTL` is over one minute, or an allowed origin is only
  wildcards.

### The shell

- It runs as `terminal.user` on the runner, which is required. Use a
  dedicated account. An account in the `docker` group controls Docker and is
  therefore root-equivalent on that runner; that is what makes it useful for
  inspecting test containers, so treat it as root.
- It runs with `nice` 10 so it yields to tests.
- When a session ends, the runner hangs up the shell, then kills everything
  left in the shell's session and any process that still holds the terminal.

### Limits and logs

- Sessions per runner (default 2) and in total (6) on the controller, and per
  runner on the runner (`max_sessions`, 2).
- A session closes after 15 minutes without input from the browser and after
  2 hours in any case.
- The controller and runner log who opened which session, from where, how
  long it lasted, the bytes in and out, and why it closed. Terminal content is
  not recorded.

### Known limitations

- The idle timer counts any input from the browser, including the replies
  xterm.js sends to terminal queries (such as a cursor position report). A
  program that keeps querying the terminal keeps the session open until the
  time limit.
- There are no per-session CPU or memory limits beyond `nice`.

## Setup

### Controller

1. Serve the controller over HTTPS and set `backend.trustedProxies`
   ([HTTPS](../deb/https.md)).
2. Replace the default secrets and password in
   `/etc/it-system/config-controller.yaml`:

   ```yaml
     password: "<a real password>"
     jwt:
       secret: "<openssl rand -hex 32>"
     runnerJwt:
       secret: "<another openssl rand -hex 32>"
   ```

   A new `runnerJwt.secret` invalidates every runner token, so every runner
   has to be registered again (step 3 under [Runner](#runner)). A new
   `jwt.secret` logs every user out.
3. Enable the terminal:

   ```yaml
     terminal:
       enabled: true
   ```

4. `sudo systemctl restart it-system-controller`. The log shows
   `Terminal is enabled for browsers that come over HTTPS ...`, or the reason
   it stayed off.

### Runner

1. Create the account the shell runs as, for example with Docker access:

   ```bash
   sudo useradd --create-home --shell /bin/bash --groups docker itshell
   ```

2. In `/etc/it-system/config-runner.yaml`, under `runner`:

   ```yaml
     terminal:
       enabled: true
       user: "itshell"
   ```

3. If the runner's token is from before this feature, or the controller's
   `runnerJwt.secret` changed: delete the runner on the Runner page, then
   reinstall the runner package so that it registers again:

   ```bash
   sudo dpkg -r it-system-runner
   sudo dpkg -i it-system-runner_<version>_amd64.deb
   ```

4. `sudo systemctl restart it-system-runner`. The runner log shows
   `Terminal is enabled: admins of the controller can open a shell on this
   runner as "itshell"`, and the controller log shows
   `terminal control channel connected: runner="<name>"`.

### Use

On the Runner page, admins see a terminal button on each runner card. Each
terminal opens in its own window, which can be moved and resized; several can
be open at once.

## Troubleshooting

| Message | Cause |
| --- | --- |
| `Terminal is disabled on this controller` | `terminal.enabled` is false, or a startup check failed; the controller log says which. |
| `Terminal requires HTTPS; ...` | The page was opened over `http://`; use the `https://` address. |
| `Runner <name> is not connected for terminal` | The runner is offline, its terminal is off, or its control channel cannot connect; see `Terminal control channel` in the runner log. |
| runner log: `controller answered 403 Forbidden` | The runner's token has no current registration id; register it again (Runner step 3). |
| `Terminal is not allowed from this address` | The browser is outside `terminal.allowedClientCIDRs`. |
| `Terminal session limit reached ...` | Close another terminal on that runner, or raise the limits. |
