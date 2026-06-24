# HubFiles

HubFiles is a Go-based file browser with a React UI, a management API for session creation, and a Linux release flow that ships the backend plus the built web UI together.

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

If the service runs as `root`, HubFiles now avoids creating unnecessary root-owned content during normal file operations.

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

The server reads built UI assets from `./frontend/dist` by default. Override that path with `HUBFILES_UI_DIR` or `-ui-dir`. Uploads are capped at `100 MiB` by default; adjust with `HUBFILES_MAX_UPLOAD_BYTES` or `-max-upload-bytes`.

## Native SMB sessions

HubFiles can browse SMB shares directly without mounting them first. Create a session with an `smb://host/share/optional/base/path` root and pass credentials in the session request body:

```json
{
  "root": "smb://fileserver/team/docs",
  "ttlSeconds": 3600,
  "readonly": false,
  "allowUpload": true,
  "allowEdit": true,
  "allowDelete": true,
  "smbUsername": "alice",
  "smbPassword": "secret",
  "smbDomain": "WORKGROUP"
}
```

The returned session JSON keeps the root sanitized and does not include the SMB password. Current native SMB support covers browsing, reading/downloading, editing, uploading, creating folders/files, renaming, deleting, and storage info. ZIP and extract are still local-filesystem-only.

If you also want users to mount the connected SMB share onto the HubFiles host machine, enable host mounts explicitly:

```bash
HUBFILES_ALLOW_HOST_MOUNTS=true
HUBFILES_MOUNT_ROOT=/mnt/hubfiles
HUBFILES_HOST_MOUNT_CONFIG_ROOT=/var/lib/hubfiles/hostmount
HUBFILES_HOST_MOUNT_UID=1000
HUBFILES_HOST_MOUNT_GID=1000
HUBFILES_HOST_MOUNT_UMASK=002
```

The service host must have CIFS support and `mount -t cifs` privileges. On most Linux installs that means installing `cifs-utils` and running the service with enough privileges to mount and unmount filesystems. HubFiles writes a `0600` credentials file under `${HUBFILES_HOST_MOUNT_CONFIG_ROOT}/credentials`, mounts the share under `${HUBFILES_MOUNT_ROOT}`, and can unmount it again from the toolbar.


## Native FTP sessions

HubFiles can also browse FTP servers directly. Create a session with an `ftp://host/optional/base/path` root and pass credentials in the session request body:

```json
{
  "root": "ftp://fileserver/public/docs",
  "ttlSeconds": 3600,
  "readonly": false,
  "allowUpload": true,
  "allowEdit": true,
  "allowDelete": true,
  "ftpUsername": "alice",
  "ftpPassword": "secret"
}
```

If `ftpUsername` is blank, HubFiles uses anonymous FTP credentials: username `anonymous` and password `anonymous@`. Current FTP support covers browsing, reading/downloading, editing, uploading, creating folders/files, renaming, and deleting. FTP storage capacity is reported as unknown, and ZIP/extract remain local-filesystem-only.

FTP host mounting is available through `rclone mount` and uses the same host-mount flag:

```bash
HUBFILES_ALLOW_HOST_MOUNTS=true
HUBFILES_MOUNT_ROOT=/mnt/hubfiles
```

The service host must have `rclone` installed with FUSE mount permissions. HubFiles writes a `0600` rclone config under `${HUBFILES_HOST_MOUNT_CONFIG_ROOT}/rclone`, mounts FTP sessions under `${HUBFILES_MOUNT_ROOT}`, and can unmount them again from the toolbar. Keep `${HUBFILES_MOUNT_ROOT}` as the user-facing folder and `${HUBFILES_HOST_MOUNT_CONFIG_ROOT}` as private app state. If the service runs as root, set `${HUBFILES_HOST_MOUNT_UID}` and `${HUBFILES_HOST_MOUNT_GID}` to the desktop user so file managers and editors can read the FUSE mount.

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
