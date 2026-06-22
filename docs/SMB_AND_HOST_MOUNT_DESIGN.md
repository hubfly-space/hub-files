# SMB and Host Mount Design

This document explains the native SMB support and optional host-mount feature added to HubFiles.

## Goal

HubFiles should support two related workflows:

1. Browse and manage SMB shares directly inside HubFiles without requiring the share to be mounted by the operating system first.
2. Let a user click a button to mount the same SMB share onto the machine where HubFiles is installed, so other local apps can use the share outside the browser.

These are intentionally separate features. Native SMB browsing is normal app functionality. Host mounting is an operating-system action and is disabled by default because it requires mount privileges.

## Core Approach

The important design decision was to avoid putting SMB-specific logic directly into every HTTP handler.

Instead, the backend now has a file backend abstraction:

```go
type Backend interface {
    List(ctx context.Context, subPath string) ([]filesystem.FileInfo, error)
    Storage(ctx context.Context, subPath string) (*filesystem.StorageInfo, error)
    Read(ctx context.Context, subPath string) (io.ReadCloser, error)
    Write(ctx context.Context, subPath string, data io.Reader) error
    WriteAtomic(ctx context.Context, subPath string, data io.Reader) error
    Delete(ctx context.Context, subPath string) error
    Rename(ctx context.Context, oldSubPath, newSubPath string) error
    Mkdir(ctx context.Context, subPath string) error
    Touch(ctx context.Context, subPath string) error
}
```

The server handlers call this interface. The current session decides which implementation is used:

- local filesystem session uses `filebackend.Local`
- SMB session uses `filebackend.SMB`

That keeps the API stable for the React frontend. The browser still calls `/api/list`, `/api/file`, `/api/upload`, etc. It does not need to know whether the storage is local or SMB.

## Files Added

### `internal/filebackend/backend.go`

Defines the shared backend interface.

This is the contract the server depends on. It describes the essential file operations HubFiles needs for browsing and editing.

Why it exists:

- keeps HTTP handlers storage-agnostic
- allows local and SMB implementations to share the same API behavior
- makes future storage backends possible, such as SFTP, WebDAV, object storage, etc.

### `internal/filebackend/local.go`

Adapts the existing local filesystem code to the new backend interface.

It mostly forwards calls to the old `internal/filesystem` package:

- `List` calls `filesystem.ListDir`
- `Read` calls `filesystem.ReadFile`
- `WriteAtomic` calls `filesystem.WriteFileAtomic`
- and so on

Why it exists:

- preserves existing local behavior
- avoids rewriting the already-tested local filesystem code
- lets local and SMB sessions pass through the same server handlers

### `internal/filebackend/smb.go`

Implements native SMB access using `github.com/hirochachacha/go-smb2`.

Main responsibilities:

- connect to SMB over TCP port 445 by default
- authenticate with NTLM credentials
- mount the remote SMB share inside the Go SMB client
- list folders, read files, write files, upload files, rename, delete, mkdir, touch
- provide storage info from SMB `Statfs`
- protect paths from traversal like `../secret`
- cache SMB sessions so browsing does not reconnect on every click

Important concepts in this file:

`SMBPool`

Keeps active SMB connections keyed by HubFiles session code. Without this, every folder click could create a fresh TCP connection and login, which would feel slow.

`safeSMBRelativePath`

Normalizes user paths and rejects dangerous path parts. SMB uses backslashes internally, but HubFiles UI uses browser-style `/` paths. This function converts and validates paths before they touch the SMB share.

`WriteAtomic`

Uploads to a temporary hidden file first, then renames it to the final target. This protects an existing file from being replaced by a partial upload if the browser disconnects mid-upload.

Retry behavior

Read/list style operations retry once after dropping the cached SMB connection. This helps when a TCP session goes stale. Mutating streaming writes do not retry automatically, because the request body may already be partially consumed.

### `internal/hostmount/hostmount.go`

Implements optional OS-level mounting of an SMB session onto the HubFiles host machine.

This is different from native SMB browsing. It shells out to:

```bash
mount -t cifs //host/share /mnt/hubfiles/... -o credentials=...,iocharset=utf8,vers=3.0,noserverino
```

Main responsibilities:

- build a stable mount path under `HUBFILES_MOUNT_ROOT`
- write a `0600` CIFS credentials file
- check `/proc/self/mountinfo` to avoid mounting the same target twice
- reject unsafe values that could corrupt mount options
- return the local mount path to the UI

Why it is disabled by default:

- it performs a privileged OS action
- it needs `cifs-utils` installed
- it usually requires root or equivalent mount capability
- it writes credentials to disk, although with restrictive permissions

### `internal/hostmount/hostmount_test.go`

Tests the safe parts of host mounting without actually mounting anything.

It verifies:

- mount paths are stable and scoped under the configured root
- unsafe config values are rejected

Actual OS mounts are intentionally not run in unit tests because they require machine-level privileges.

### `internal/server/smb_session_test.go`

Tests SMB session parsing and session capability reporting.

It verifies:

- `smb://` roots are parsed correctly
- credentials embedded in the URL are sanitized out of the returned root
- traversal in SMB base paths is rejected
- `/api/session` reports SMB sessions and whether host mounting is enabled

## Files Modified

### `internal/sessions/sessions.go`

Adds `SMBConfig` and allows a session to carry SMB connection details.

The password is stored in memory only and marked `json:"-"`, so it is not returned in API responses.

Conceptually, a session now has:

- public session data, such as root path and permissions
- optional private SMB connection data

### `internal/server/server.go`

This is where the main backend wiring happens.

Important additions:

`parseSMBSessionRoot`

Parses roots like:

```text
smb://fileserver/share/base/path
smb://DOMAIN;alice:secret@fileserver:445/share/base/path
```

It extracts:

- host
- port
- share
- base path
- username
- password
- domain

It returns a sanitized root like:

```text
smb://fileserver/share/base/path
```

No password is returned to the client.

`backendForRequest`

Chooses the correct backend for every request:

- SMB session: `filebackend.SMB`
- local session: `filebackend.Local`

`handleSessionInfo`

Adds `GET /api/session` so the frontend can know:

- whether this is a local or SMB session
- whether host mounting is allowed
- current permission flags

`handleHostMount`

Adds `POST /api/host-mount`.

This only works when:

- the current session is SMB
- `HUBFILES_ALLOW_HOST_MOUNTS=true`

Otherwise it returns an error.

Zip/extract gating

ZIP and extract still depend on local OS paths. For SMB sessions they now return `501 Not Implemented`. This is intentional for the first version because archive support over SMB needs a separate streaming design.

### `internal/config/config.go`

Adds host mount configuration:

```bash
HUBFILES_ALLOW_HOST_MOUNTS=true
HUBFILES_MOUNT_ROOT=/mnt/hubfiles
```

The code still accepts old `HUBFLY_*` environment variables as fallbacks to avoid breaking existing installs.

### `frontend/src/api/index.ts`

Adds frontend API calls for:

- `api.session()` -> `GET /api/session`
- `api.hostMount()` -> `POST /api/host-mount`

Also adds TypeScript types:

- `SessionInfo`
- `HostMountResult`

### `frontend/src/hooks/useFileSystem.ts`

Loads session info in parallel with files and storage:

```ts
const [sessionData, data, storageData] = await Promise.all([
  api.session(),
  api.list(path),
  api.storage(path),
]);
```

Why this matters:

- avoids an extra sequential wait
- toolbar can know whether to show the host-mount button
- keeps session capability state close to file browsing state

### `frontend/src/App.tsx`

Adds the click handler for mounting:

- calls `api.hostMount()`
- shows success toast with the returned mount path
- shows destructive toast on failure
- tracks `hostMounting` state so the button can show progress/disable itself

### `frontend/src/components/Toolbar.tsx`

Adds the host mount button.

It only renders when:

```ts
canHostMount === true
```

The icon used is `HardDriveDownload`.

The tooltip says:

```text
Mount SMB on this machine
```

### `frontend/src/components/app/AppHeader.tsx`

Passes host mount props from `App.tsx` into `Toolbar`.

### `README.md`

Documents:

- native SMB sessions
- SMB session request body
- host mount configuration
- runtime requirements for host mounting

### `packaging/hubfly-files.env.example`

Adds example config:

```bash
HUBFILES_ALLOW_HOST_MOUNTS=false
HUBFILES_MOUNT_ROOT=/mnt/hubfiles
```

## Data Flow: Native SMB Browsing

1. Management API receives `POST /sessions` with an SMB root.
2. Server detects `smb://` root.
3. Server parses the SMB root and credentials into `sessions.SMBConfig`.
4. Session store saves public root and private SMB config.
5. Browser opens HubFiles with `?session=<token>`.
6. UI calls normal file API endpoints.
7. Auth middleware loads the session and places it on request context.
8. `backendForRequest` sees `session.SMB != nil` and returns SMB backend.
9. SMB backend connects through `SMBPool`, performs the file operation, and returns normal API responses.

## Data Flow: Host Mount

1. User is in an SMB session.
2. UI calls `GET /api/session`.
3. Server responds with `type: "smb"` and `canHostMount: true` if enabled.
4. Toolbar shows the mount button.
5. User clicks mount.
6. UI calls `POST /api/host-mount`.
7. Server verifies host mounts are enabled and session is SMB.
8. `hostmount.MountSMB` creates the mount folder and credentials file.
9. It checks if already mounted.
10. If not mounted, it runs `mount -t cifs`.
11. API returns the local mount path.
12. UI displays that path in a toast.

## Security Notes

SMB credentials are sensitive. The implementation handles them with these rules:

- passwords are not serialized in session JSON
- returned SMB roots are sanitized
- host mounts are disabled by default
- host mounting requires an explicit environment flag
- CIFS credentials files are written with `0600` permissions
- path traversal like `..` is rejected for SMB paths
- unsafe mount option characters are rejected before shelling out to `mount`

One important limitation: host mounting still writes credentials to disk because `mount.cifs` expects credentials from options or a credentials file. The chosen file approach avoids putting passwords directly in process arguments, which are easier for other users/processes to inspect.

## Why Not Mount Automatically?

Automatic mounting would surprise administrators and users because it changes host OS state. It also requires privileges that normal browsing does not require.

The explicit button is safer:

- native SMB browsing works without OS mounts
- mounting happens only when the user asks
- admins can disable the feature globally
- failures are isolated to the host-mount flow

## Current Limitations

- ZIP and extract are local-only for now.
- Host mounting is Linux/CIFS-specific.
- There is no unmount button yet.
- There is no mount cleanup scheduler yet.
- There is no per-session choice of SMB protocol version yet; CIFS mount currently uses `vers=3.0`.
- There is no UI for creating SMB sessions yet; sessions are created through the management API.

## Future Improvements

Good next steps:

- add an unmount endpoint and button
- add mount status display in `/api/session`
- add a UI dialog for creating SMB sessions
- add configurable CIFS options, especially protocol version
- add archive support over SMB using streaming rather than local paths
- add idle cleanup for SMB client sessions and host mounts
- add integration tests with a Samba container

## Testing Commands

Targeted backend verification:

```bash
go test ./internal/hostmount ./internal/server ./internal/config ./internal/filebackend ./internal/sessions ./internal/filesystem ./cmd/hubfly-files
```

Frontend verification:

```bash
cd frontend
npm run build
```

Full Go test suite currently still depends on unrelated `internal/search` issues being fixed.
