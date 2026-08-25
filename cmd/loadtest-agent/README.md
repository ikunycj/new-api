# AllToken Load-Test Agent

The agent runs the repository's controlled k6 workload outside the browser. It
only accepts structured tasks from an AllToken account that paired the agent;
the server cannot execute arbitrary shell commands on the machine.

## Build

From the repository root:

```bash
go build -trimpath -o alltoken-loadtest-agent ./cmd/loadtest-agent
```

Cross-compile examples:

```bash
GOOS=darwin GOARCH=arm64 go build -trimpath -o alltoken-loadtest-agent-darwin-arm64 ./cmd/loadtest-agent
GOOS=windows GOARCH=amd64 go build -trimpath -o alltoken-loadtest-agent-windows-amd64.exe ./cmd/loadtest-agent
GOOS=linux GOARCH=amd64 go build -trimpath -o alltoken-loadtest-agent-linux-amd64 ./cmd/loadtest-agent
```

Install k6 separately and make sure it is available as `k6` in `PATH`.

On macOS with Homebrew:

```bash
brew install k6
```

On Debian/Ubuntu, install k6 from the official Grafana package repository. Do
not run an untrusted binary downloaded from a chat message; verify the release
checksum before placing it in `PATH`.

## Pair and run

1. In the Load Test Demo, create a pairing code.
2. Run the displayed command on the load-generator computer.
3. Keep the agent running with:

```bash
./alltoken-loadtest-agent run
```

The credential is stored in the operating system configuration directory with
mode `0600`. The API key is written only to a temporary `0600` file while k6
runs and is removed when the run exits.

## Test modes

The agent always sends requests through the selected AllToken API key and its
configured channel routing. To measure gateway capacity without consuming a
real provider account, create a dedicated test key whose billing-group route
points to the repository's mock channels, then select that key in the demo.
The bundled isolated mock stack is under `deploy/loadtest`; it is deliberately
separate from production and uses no real provider credentials. After the mock
capacity test passes, select a dedicated real-provider key for the real-account
test. The agent does not silently switch between mock and real channels.

## Task lifecycle

The browser creates a durable run record. The paired agent polls for one task,
reports heartbeats, and uploads the final k6 summary (request count, status and
AllToken error-code breakdown, token usage, and P50/P95/P99). Stopping a run is
cooperative; if the agent is restarted, a pending stop is acknowledged and the
run is finalized as cancelled instead of remaining stuck as running.

The task payload is structured and validated by both server and agent. It does
not contain arbitrary shell commands, and the server never receives the
agent's local filesystem contents.
