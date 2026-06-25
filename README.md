# All-in-One SCP

Cross-platform Go app for:

- opening an SSH terminal
- downloading remote files or folders over SFTP
- skipping paths like `node_modules`, `.git`, or `*.log`
- using the same UI on Windows, macOS, and Linux

The app starts a local web UI in your browser, but the SSH/SFTP work stays in Go.

## Why this shape

A browser UI keeps the app portable:

- no C toolchain needed
- no OS-specific desktop widget toolkit required
- the same code works on Windows, Linux, and macOS

## Project tree

```text
all-in-one-scp/
├── .gitignore
├── go.mod
├── main.go
├── README.md
└── internal/
    ├── app/
    │   ├── browser_open_*.go
    │   ├── log_hub.go
    │   ├── server.go
    │   └── static/
    │       ├── app-icon.png
    │       ├── app.js
    │       ├── index.html
    │       └── styles.css
    └── transfer/
        ├── auth.go
        ├── config.go
        ├── pull.go
        ├── ssh_command.go
        └── terminal_*.go
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

- Remote path: `/root/meow`
- Local path on Windows: `C:\Users\mahdi\Downloads\backup`
- Local path on Linux/macOS: `~/Downloads/backup`
- Excludes: `node_modules,.git,*.log`

## Notes

- For SSH, the app launches a system terminal window and runs `ssh` there.
- For downloads and connection tests, the app uses Go SSH/SFTP libraries directly.
- If your OS does not have `ssh` installed in `PATH`, the terminal-launch feature will fail, but download/test can still work if the Go connection settings are valid.
