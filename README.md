# All-in-One SCP

Cross-platform Go app for:

- opening an in-browser SSH console
- downloading remote files or folders over SFTP
- copying files or folders from server 1 to server 2
- saving reusable server profiles
- re-running recent transfers from history
- skipping paths like `node_modules`, `.git`, or `*.log`
- using the same UI on Windows, macOS, and Linux

The app starts a local web UI in your browser, but the SSH and SFTP work stays in Go.

## Why this shape

A browser UI keeps the app portable:

- no C toolchain needed
- no OS-specific desktop UI toolkit required
- the same app works on Windows, Linux, and macOS

## Project tree

```text
all-in-one-scp/
|-- .gitignore
|-- go.mod
|-- go.sum
|-- main.go
|-- README.md
`-- internal/
    |-- app/
    |   |-- browser_open_darwin.go
    |   |-- browser_open_linux.go
    |   |-- browser_open_windows.go
    |   |-- log_hub.go
    |   |-- server.go
    |   `-- static/
    |       |-- app-icon.png
    |       |-- app.js
    |       |-- index.html
    |       `-- styles.css
    `-- transfer/
        |-- auth.go
        |-- browser_shell.go
        |-- config.go
        |-- pull.go
        `-- remote_copy.go
```

## Run

```powershell
cd .\all-in-one-scp
go run .
```

The app opens your default browser automatically.

If you only want the URL and do not want auto-open:

```powershell
$env:ALL_IN_ONE_SCP_NO_BROWSER="1"
go run .
```

## Build

### Windows

```powershell
go build -ldflags="-H windowsgui" -o .\bin\all-in-one-scp.exe .
```

### Linux

```bash
go build -o ./bin/all-in-one-scp .
```

### macOS

```bash
go build -o ./bin/all-in-one-scp .
```

## UI examples

### Download from a server

- Source server host: `203.0.113.10`
- Source path: `/root/meow`
- Local path on Windows: `C:\Users\mahdi\Downloads\backup`
- Local path on Linux or macOS: `~/Downloads/backup`
- Excludes: `node_modules,.git,*.log`

### Copy from server 1 to server 2

- Mode: `Server to Server`
- Source server host: `203.0.113.10`
- Source path: `/root/meow`
- Destination server host: `203.0.113.20`
- Destination folder: `/root/backup`
- Excludes: `node_modules,.git,*.log`

If the source path is `/root/meow` and the destination folder is `/root/backup`, the app copies to:

```text
/root/backup/meow
```

## Built-in UX features

- Save `Server 1` and `Server 2` setups as quick profiles.
- Load a saved profile into either server card with one click.
- Keep a recent transfer list in the UI.
- Load or re-run a recent transfer without rebuilding the form.

## Notes

- The SSH terminal runs inside the app page.
- The app does not close on operation errors. It shows the error and stays open.
- In `Server to Server` mode, `Test Connection` checks both servers.
- For remote copy, the app streams data through your app process:

```text
server 1 -> your app -> server 2
```
