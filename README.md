# HubFiles

HubFiles is a web file browser for Linux servers. It has a Go backend, a React UI, session-based access, and support for local files, SMB shares, FTP servers, and local indexed search.

## Features

- Browse, preview, download, upload, edit, rename, and delete files.
- Create folders and empty files from the UI.
- Zip and extract local filesystem files.
- Create time-limited sessions with read-only, upload, edit, and delete permissions.
- Browse local folders, SMB shares, and FTP servers.
- Optional host mounts for SMB and FTP sessions.
- Search indexed local session files with SQLite.
- Release archives for Linux `amd64` and `arm64`.

## Install

Latest release:

```bash
curl -fsSL https://raw.githubusercontent.com/hubfly-space/hub-files/main/scripts/install.sh | bash
```

Specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/hubfly-space/hub-files/main/scripts/install.sh | bash -s -- --version v1.0.0
```

The installer puts the binary and web UI on the host, installs the systemd service, and keeps `/etc/hubfly-files/hubfly-files.env` when updating.

## Run from source

Requirements: Go, Node.js 22+, npm.

```bash
cd frontend
npm ci
npm run build

cd ..
go run ./cmd/hubfly-files
```

Default ports:

- `10015` for the web UI and file API
- `10014` for the management API

## Test

```bash
go test ./...

cd frontend
npm ci
npm run build
```

## Docs

Read [docs/USAGE.md](docs/USAGE.md) for sessions, SMB, FTP, host mounts, search, and common configuration.

## Contributing

Read [CONTRIBUTING.md](CONTRIBUTING.md) before opening issues or pull requests.

## License

Apache-2.0. See [LICENSE](LICENSE).
