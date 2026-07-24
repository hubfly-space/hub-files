# Using HubFiles

This is the practical guide for running HubFiles and creating sessions.

## What HubFiles runs

HubFiles starts two HTTP servers:

- `10015`: web UI and file API
- `10014`: management API used to create sessions

The UI needs a session token to work with real files. Without a token, HubFiles falls back to demo mode.

## Run locally

Build the web UI first:

```bash
cd frontend
npm ci
npm run build
```

Start the backend:

```bash
cd ..
go run ./cmd/hubfly-files
```

Open:

```text
http://localhost:10015
```

## Create a local session

Create a session from the management API:

```bash
curl -s http://localhost:10014/sessions \
  -H 'Content-Type: application/json' \
  -d '{
    "root": "/srv/files",
    "ttlSeconds": 3600,
    "readonly": false,
    "allowUpload": true,
    "allowEdit": true,
    "allowDelete": true
  }'
```

The response includes `sessionCode`. Use it in the UI as query parameter {session:"your session"} or API as a bearer token:

```bash
curl http://localhost:10015/api/list?path=/ \
  -H "Authorization: Bearer <sessionCode>"
```

Permissions mean:

- `readonly`: blocks all write actions
- `allowUpload`: allows uploads
- `allowEdit`: allows editing file contents
- `allowDelete`: allows delete actions

## SMB session

Use an SMB URL as the root:

```bash
curl -s http://localhost:10014/sessions \
  -H 'Content-Type: application/json' \
  -d '{
    "root": "smb://fileserver/team/docs",
    "ttlSeconds": 3600,
    "readonly": false,
    "allowUpload": true,
    "allowEdit": true,
    "allowDelete": true,
    "smbUsername": "alice",
    "smbPassword": "secret",
    "smbDomain": "WORKGROUP"
  }'
```

SMB supports listing, reading, downloading, editing, uploading, creating folders/files, renaming, deleting, and storage info. Zip and extract are only for local filesystem sessions.

## FTP session

Use an FTP URL as the root:

```bash
curl -s http://localhost:10014/sessions \
  -H 'Content-Type: application/json' \
  -d '{
    "root": "ftp://fileserver/public/docs",
    "ttlSeconds": 3600,
    "readonly": false,
    "allowUpload": true,
    "allowEdit": true,
    "allowDelete": true,
    "ftpUsername": "alice",
    "ftpPassword": "secret"
  }'
```

If username is empty, HubFiles uses anonymous FTP. FTP storage size is reported as unknown. Zip and extract are only for local filesystem sessions.

## Host mounts

Native SMB and FTP sessions do not need host mounts. Enable host mounts only when you want the remote share mounted on the HubFiles host machine.

```bash
HUBFILES_ALLOW_HOST_MOUNTS=true
HUBFILES_MOUNT_ROOT=/mnt/hubfiles
HUBFILES_HOST_MOUNT_CONFIG_ROOT=/var/lib/hubfiles/hostmount
HUBFILES_HOST_MOUNT_UID=1000
HUBFILES_HOST_MOUNT_GID=1000
HUBFILES_HOST_MOUNT_UMASK=002
```

SMB host mounts need `cifs-utils`. FTP host mounts need `rclone` and FUSE permissions.

## Search

Local sessions are indexed in SQLite. Search uses the session root, so results stay scoped to the current session.

```bash
curl 'http://localhost:10015/api/search?q=report' \
  -H "Authorization: Bearer <sessionCode>"
```

Search is for local filesystem sessions. Remote SMB and FTP sessions can still browse normally.

## Common configuration

Set these in the environment or the systemd env file:

```bash
HUBFILES_API_PORT=10015
HUBFILES_MGMT_PORT=10014
HUBFILES_DEMO_DIR=./demo
HUBFILES_UI_DIR=./frontend/dist
HUBFILES_MAX_UPLOAD_BYTES=104857600
```

Set `HUBFILES_MAX_UPLOAD_BYTES=0` to remove the upload limit.

## Release build

A tag like `v1.0.0` triggers the release workflow. It builds the frontend, builds Linux `amd64` and `arm64` archives, and publishes checksums.
