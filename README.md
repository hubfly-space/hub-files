# HubFly Files

HubFly Files is a Go-based file browser with a React UI, a management API for session creation, and a Linux release flow that ships the backend plus the built web UI together.

## Default ports

- `10014`: management API
- `10015`: UI and file API

## Linux install and update

Install the latest release:

```bash
curl -fsSL https://raw.githubusercontent.com/hubfly-space/hub-files/main/scripts/install.sh | bash
```

Install a specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/hubfly-space/hub-files/main/scripts/install.sh | bash -s -- --version v1.0.0
```

Re-running the installer updates an existing installation in place:

- replaces the `hubfly-files` binary
- refreshes the shipped web assets under `/opt/hubfly-files/web`
- preserves `/etc/hubfly-files/hubfly-files.env`
- restarts the `hubfly-files` systemd service if it already exists

## Ownership behavior

If the service runs as `root`, HubFly Files now avoids creating unnecessary root-owned content during normal file operations.

- new files inherit the owner and group of the destination file or parent directory
- new folders inherit the owner and group of the nearest existing parent directory
- zip outputs and extracted content inherit the target directory ownership

This keeps edits aligned with the mounted path you are managing instead of defaulting every new inode to `root:root`.

## Local development

Build the frontend first:

```bash
cd frontend
npm ci
npm run build
```

Then run the Go server from the repository root:

```bash
go run ./cmd/hubfly-files
```

The server reads built UI assets from `./frontend/dist` by default. Override that path with `HUBFLY_UI_DIR` or `-ui-dir`. Uploads are capped at `100 MiB` by default; adjust with `HUBFLY_MAX_UPLOAD_BYTES` or `-max-upload-bytes`.

## API documentation

Detailed HTTP API documentation is available in [docs/API.md](/home/bonheur/Desktop/Projects/hubfly/tools/hubfly-files/docs/API.md:1).

## Release process

Push a tag such as `v1.0.0` and GitHub Actions will:

- build the frontend
- package Linux `amd64` and `arm64` release archives
- publish the archives and a checksums file to the GitHub release

Each archive contains:

- `bin/hubfly-files`
- `web/` built UI assets
- `systemd/hubfly-files.service`
- `config/hubfly-files.env.example`
- `install.sh`
