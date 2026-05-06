# HubFly Files API Reference

This document describes the current HTTP API exposed by HubFly Files as implemented in the repository on May 6, 2026.

It covers:

- runtime ports and server split
- authentication and session handling
- request and response formats
- endpoint-by-endpoint behavior
- known error responses
- permission model
- implementation notes that affect client behavior

## Overview

HubFly Files exposes two HTTP servers:

- UI/API server, default port `10015`
- management server, default port `10014`

By default:

- the UI and file API are served from `http://<host>:10015`
- session creation is served from `http://<host>:10014/sessions`

## Configuration

Runtime configuration is controlled by flags or environment variables.

| Setting | Flag | Environment variable | Default |
| --- | --- | --- | --- |
| UI/API port | `-api-port` | `HUBFLY_API_PORT` | `10015` |
| Management port | `-mgmt-port` | `HUBFLY_MGMT_PORT` | `10014` |
| Demo directory | `-demo-dir` | `HUBFLY_DEMO_DIR` | `./demo` |
| Built UI directory | `-ui-dir` | `HUBFLY_UI_DIR` | `./frontend/dist` |

## Base URLs

| Surface | Base URL |
| --- | --- |
| UI/API | `http://<host>:10015` |
| Management API | `http://<host>:10014` |

All UI/API endpoints are rooted at `/api/...`.

## Authentication Model

### Session token transport

The UI/API server authenticates requests using a session token.

Preferred form:

```http
Authorization: Bearer <session-token>
```

Compatibility fallback:

- query parameter `session=<token>`

Notes:

- query-parameter tokens are still accepted
- the server logs a warning when a token is supplied through the query string
- query tokens are less secure and should be avoided

### Demo mode

If no valid session is found:

- an empty token or the token `demo` activates demo mode
- demo mode is always read-only
- demo mode disables upload, edit, and delete operations
- the configured demo directory is created automatically if needed
- `GET /api/storage` returns a fixed fake volume in demo mode:
  `20 MiB` total, `5 MiB` used, `15 MiB` available

Demo mode permissions:

| Capability | Value |
| --- | --- |
| Read files | allowed |
| List directories | allowed |
| Upload | denied |
| Edit files | denied |
| Delete files | denied |
| Create directories | denied |
| Rename | denied |
| Zip | denied |
| Extract | denied |

### Session permissions

A created session carries these permission flags:

- `readonly`
- `allowUpload`
- `allowEdit`
- `allowDelete`

Behavior:

- if `readonly=true`, mutation endpoints are blocked before more specific permission checks
- `allowUpload`, `allowEdit`, and `allowDelete` apply to their related endpoints
- `mkdir`, `rename`, `zip`, and `extract` are blocked by `readonly`, but do not have separate fine-grained flags beyond that

## Content Types

### Common request content types

| Endpoint type | Content type |
| --- | --- |
| JSON request bodies | `application/json` |
| File upload | `multipart/form-data` |
| Raw file write | any raw body, typically `text/plain` or `application/octet-stream` |

### Common response content types

| Response type | Content type |
| --- | --- |
| JSON responses | `application/json` |
| File download/view | no forced content type; served as raw bytes |
| Errors from `http.Error` | `text/plain; charset=utf-8` |

## Request Size Limits

The server enforces these request size limits:

| Limit | Value | Applies to |
| --- | --- | --- |
| JSON body limit | `1 MiB` | `/api/mkdir`, `/api/rename`, `/api/delete`, `/api/zip`, `/api/extract` |
| Upload body limit | `10 MiB` | `/api/upload` |

Notes:

- `PUT /api/file` does not use the JSON-body middleware and is not explicitly capped by the same `1 MiB` limit
- `POST /sessions` on the management server does not use the JSON-body middleware either
- upload parsing rejects oversized multipart bodies with `413 Request Entity Too Large`

## Data Models

### FileInfo

Returned by `GET /api/list`.

```json
{
  "name": "report.pdf",
  "isDir": false,
  "size": 34892,
  "modTime": "2026-05-06 18:33:10"
}
```

Fields:

| Field | Type | Description |
| --- | --- | --- |
| `name` | string | Base name of the entry |
| `isDir` | boolean | Whether the entry is a directory |
| `size` | integer | File size in bytes |
| `modTime` | string | Modification time formatted as `YYYY-MM-DD HH:MM:SS` |

### StorageInfo

Returned by `GET /api/storage`.

```json
{
  "path": "/srv/data/projects",
  "totalBytes": 107374182400,
  "usedBytes": 32212254720,
  "availableBytes": 75161927680,
  "usedPercent": 30
}
```

Fields:

| Field | Type | Description |
| --- | --- | --- |
| `path` | string | Resolved filesystem path used for the stat call |
| `totalBytes` | integer | Total capacity of the mounted filesystem |
| `usedBytes` | integer | Used bytes on that filesystem |
| `availableBytes` | integer | Available bytes on that filesystem |
| `usedPercent` | number | Used capacity percentage |

### Session

Returned by `POST /sessions`.

```json
{
  "sessionCode": "f5d2c1...",
  "root": "/srv/data",
  "expiresAt": "2026-05-06T20:10:51.539355+02:00",
  "readonly": false,
  "allowUpload": true,
  "allowEdit": true,
  "allowDelete": false
}
```

Fields:

| Field | Type | Description |
| --- | --- | --- |
| `sessionCode` | string | Hex session token |
| `root` | string | Absolute root path for the session |
| `expiresAt` | string | Expiration timestamp |
| `readonly` | boolean | Read-only mode flag |
| `allowUpload` | boolean | Upload permission |
| `allowEdit` | boolean | Edit permission |
| `allowDelete` | boolean | Delete permission |

## Error Response Format

Errors are plain text, not structured JSON.

Typical response body examples:

- `Unauthorized`
- `Read-only session`
- `AllowEdit not allowed for this session`
- `Invalid request body`
- `Invalid source path`
- `Internal server error`

The server uses `http.Error`, so the body normally ends with a trailing newline.

## UI/API Endpoints

## `GET /api/list`

Lists the contents of a directory inside the session root.

### Authentication

Required, unless using demo mode.

### Query parameters

| Name | Required | Type | Description |
| --- | --- | --- | --- |
| `path` | no | string | Path relative to the session root. Empty means the root directory. |

### Successful response

Status:

- `200 OK`

Body:

- JSON array of `FileInfo`

Example:

```bash
curl -H "Authorization: Bearer <token>" \
  "http://localhost:10015/api/list?path=/docs"
```

Example response:

```json
[
  {
    "name": "manual.pdf",
    "isDir": false,
    "size": 89123,
    "modTime": "2026-05-06 12:00:01"
  },
  {
    "name": "archive",
    "isDir": true,
    "size": 4096,
    "modTime": "2026-05-06 11:51:43"
  }
]
```

### Error responses

| Status | Body | When |
| --- | --- | --- |
| `401 Unauthorized` | `Unauthorized` | Invalid non-demo token |
| `400 Bad Request` | `Invalid path` | Session-relative path failed validation |
| `403 Forbidden` | `Permission denied` | OS-level permission failure |
| `404 Not Found` | `Not found` | Directory does not exist |
| `500 Internal Server Error` | `Internal server error` | Unexpected server-side failure |

Notes:

- path validation failures are currently surfaced as `500`, not `400`
- unauthorized path traversal attempts become generic internal errors to the client

## `GET /api/storage`

Returns capacity information for the mounted filesystem behind the requested path.

### Authentication

Required, unless using demo mode.

### Query parameters

| Name | Required | Type | Description |
| --- | --- | --- | --- |
| `path` | no | string | Path relative to the session root. Empty means the root directory. |

### Successful response

Status:

- `200 OK`

Body:

- JSON `StorageInfo`

Example:

```bash
curl -H "Authorization: Bearer <token>" \
  "http://localhost:10015/api/storage?path=/"
```

### Error responses

| Status | Body | When |
| --- | --- | --- |
| `401 Unauthorized` | `Unauthorized` | Invalid non-demo token |
| `400 Bad Request` | `Invalid path` | Session-relative path failed validation |
| `403 Forbidden` | `Permission denied` | OS-level permission failure |
| `404 Not Found` | `Not found` | Path does not exist |
| `500 Internal Server Error` | `Internal server error` | Unexpected stat failure |

## `GET /api/file`

Reads a file and returns its raw content.

### Authentication

Required, unless using demo mode.

### Query parameters

| Name | Required | Type | Description |
| --- | --- | --- | --- |
| `path` | yes | string | Path to the file relative to the session root |

### Successful response

Status:

- `200 OK`

Body:

- raw file bytes

Notes:

- this endpoint does not force a `Content-Type`
- binary and text content are both returned as-is

Example:

```bash
curl -H "Authorization: Bearer <token>" \
  "http://localhost:10015/api/file?path=/notes.txt"
```

### Error responses

| Status | Body | When |
| --- | --- | --- |
| `401 Unauthorized` | `Unauthorized` | Invalid non-demo token |
| `400 Bad Request` | `Invalid path` | Session-relative path failed validation |
| `403 Forbidden` | `Permission denied` | OS-level permission failure |
| `404 Not Found` | `Not found` | File does not exist |
| `500 Internal Server Error` | `Internal server error` | Unexpected read failure |

## `PUT /api/file`

Creates or replaces a file using the raw request body.

### Authentication

Requires a non-read-only session with `allowEdit=true`.

### Query parameters

| Name | Required | Type | Description |
| --- | --- | --- | --- |
| `path` | yes | string | Destination path relative to the session root |

### Request body

- raw bytes

### Successful response

Status:

- `200 OK`

Body:

- empty

### Ownership behavior

If the service runs as root:

- an existing file keeps its inode and ownership
- a newly created file inherits owner/group from the destination file path or parent directory instead of defaulting to `root:root`

### Error responses

| Status | Body | When |
| --- | --- | --- |
| `401 Unauthorized` | `Unauthorized` | Invalid non-demo token |
| `403 Forbidden` | `Read-only session` | Session is read-only |
| `403 Forbidden` | `AllowEdit not allowed for this session` | `allowEdit=false` |
| `400 Bad Request` | `Invalid path` | Session-relative path failed validation |
| `403 Forbidden` | `Permission denied` | OS-level permission failure |
| `404 Not Found` | `Not found` | Parent path does not exist |
| `500 Internal Server Error` | `Internal server error` | Unexpected write failure |

## `POST /api/upload`

Uploads one file using multipart form data.

### Authentication

Requires a non-read-only session with `allowUpload=true`.

### Content type

`multipart/form-data`

### Form fields

| Field | Required | Type | Description |
| --- | --- | --- | --- |
| `file` | yes | file | Uploaded file |
| `path` | no | string | Destination directory relative to session root |

The final destination path is:

```text
<path>/<original filename>
```

### Successful response

Status:

- `200 OK`

Body:

- empty

### Size limits

- maximum upload request size: `10 MiB`

### Ownership behavior

If a new file is created while the service runs as root, it inherits the owner/group of the destination directory instead of defaulting to `root:root`.

### Error responses

| Status | Body | When |
| --- | --- | --- |
| `401 Unauthorized` | `Unauthorized` | Invalid non-demo token |
| `403 Forbidden` | `Read-only session` | Session is read-only |
| `403 Forbidden` | `AllowUpload not allowed for this session` | `allowUpload=false` |
| `400 Bad Request` | `Invalid file upload` | Multipart body missing `file` or malformed |
| `413 Request Entity Too Large` | `File too large (max 10MB)` | Upload exceeds limit |
| `400 Bad Request` | `Invalid path` | Destination path failed validation |
| `403 Forbidden` | `Permission denied` | OS-level permission failure |
| `404 Not Found` | `Not found` | Destination parent directory does not exist |
| `500 Internal Server Error` | `Internal server error` | Unexpected write failure |

## `POST /api/mkdir`

Creates a directory path recursively.

### Authentication

Requires a non-read-only session.

### Request body

```json
{
  "path": "/reports/2026/may"
}
```

### Successful response

Status:

- `200 OK`

Body:

- empty

### Ownership behavior

If the service runs as root, new directories inherit the owner/group of the nearest existing parent directory.

### Error responses

| Status | Body | When |
| --- | --- | --- |
| `401 Unauthorized` | `Unauthorized` | Invalid non-demo token |
| `403 Forbidden` | `Read-only session` | Session is read-only |
| `400 Bad Request` | `Invalid request body` | Invalid JSON |
| `400 Bad Request` | `Invalid path` | Session-relative path failed validation |
| `403 Forbidden` | `Permission denied` | OS-level permission failure |
| `404 Not Found` | `Not found` | Nearest required parent path does not exist |
| `500 Internal Server Error` | `Internal server error` | Unexpected creation failure |

## `POST /api/rename`

Renames or moves a file or directory.

### Authentication

Requires a non-read-only session.

### Request body

```json
{
  "oldPath": "/docs/old.txt",
  "newPath": "/docs/new.txt"
}
```

### Successful response

Status:

- `200 OK`

Body:

- empty

### Error responses

| Status | Body | When |
| --- | --- | --- |
| `401 Unauthorized` | `Unauthorized` | Invalid non-demo token |
| `403 Forbidden` | `Read-only session` | Session is read-only |
| `400 Bad Request` | `Invalid request body` | Invalid JSON |
| `400 Bad Request` | `Invalid path` | Source or target path failed validation |
| `403 Forbidden` | `Permission denied` | OS-level permission failure |
| `404 Not Found` | `Not found` | Source path does not exist |
| `500 Internal Server Error` | `Internal server error` | Unexpected rename failure |

## `DELETE /api/delete`

Deletes a file or directory tree.

### Authentication

Requires a non-read-only session with `allowDelete=true`.

### Query parameters

| Name | Required | Type | Description |
| --- | --- | --- | --- |
| `path` | yes | string | Path to delete relative to the session root |

### Successful response

Status:

- `200 OK`

Body:

- empty

### Behavior

- deletion uses recursive removal
- deleting a directory removes all of its contents

### Error responses

| Status | Body | When |
| --- | --- | --- |
| `401 Unauthorized` | `Unauthorized` | Invalid non-demo token |
| `403 Forbidden` | `Read-only session` | Session is read-only |
| `403 Forbidden` | `AllowDelete not allowed for this session` | `allowDelete=false` |
| `400 Bad Request` | `Invalid path` | Session-relative path failed validation |
| `403 Forbidden` | `Permission denied` | OS-level permission failure |
| `500 Internal Server Error` | `Internal server error` | Unexpected removal failure |

## `POST /api/zip`

Creates a zip archive from a file or directory.

### Authentication

Requires a non-read-only session.

### Request body

```json
{
  "source": "/docs",
  "target": "/docs.zip"
}
```

### Successful response

Status:

- `200 OK`

Body:

- empty

### Behavior

- if the source is a directory, the base directory name is included in the archive
- new zip files inherit destination ownership when the service runs as root

### Error responses

| Status | Body | When |
| --- | --- | --- |
| `401 Unauthorized` | `Unauthorized` | Invalid non-demo token |
| `403 Forbidden` | `Read-only session` | Session is read-only |
| `400 Bad Request` | `Invalid request body` | Invalid JSON |
| `400 Bad Request` | `Invalid source path` | Source path failed validation |
| `400 Bad Request` | `Invalid target path` | Target path failed validation |
| `403 Forbidden` | `Permission denied` | OS-level permission failure |
| `404 Not Found` | `Not found` | Source path or target parent path does not exist |
| `500 Internal Server Error` | `Internal server error` | Unexpected zip failure |

## `POST /api/extract`

Extracts a zip archive into a target directory.

### Authentication

Requires a non-read-only session.

### Request body

```json
{
  "source": "/backup.zip",
  "target": "/restore"
}
```

### Successful response

Status:

- `200 OK`

Body:

- empty

### Behavior

- extracted directories and files inherit target ownership when the service runs as root
- symlinks inside zip archives are rejected
- ZipSlip-style path traversal entries are rejected
- file permissions from the zip are not preserved; extracted files are created with safe defaults

### Error responses

| Status | Body | When |
| --- | --- | --- |
| `401 Unauthorized` | `Unauthorized` | Invalid non-demo token |
| `403 Forbidden` | `Read-only session` | Session is read-only |
| `400 Bad Request` | `Invalid request body` | Invalid JSON |
| `400 Bad Request` | `Invalid source path` | Source path failed validation |
| `400 Bad Request` | `Invalid target path` | Target path failed validation |
| `400 Bad Request` | `Invalid archive` | Zip contains blocked entries such as symlinks or traversal paths |
| `403 Forbidden` | `Permission denied` | OS-level permission failure |
| `404 Not Found` | `Not found` | Source archive or target parent path does not exist |
| `500 Internal Server Error` | `Internal server error` | Unexpected unzip failure |

## Management API

## `POST /sessions`

Creates a new session token for a root path.

Base URL:

- `http://<host>:10014/sessions`

### Authentication

None at the HTTP layer.

Important:

- this endpoint is currently unauthenticated by design in the current implementation
- it should only be exposed on trusted networks or behind an external access-control layer

### Request body

```json
{
  "root": "/srv/data",
  "ttlSeconds": 3600,
  "readonly": false,
  "allowUpload": true,
  "allowEdit": true,
  "allowDelete": false
}
```

### Request fields

| Field | Required | Type | Description |
| --- | --- | --- | --- |
| `root` | yes | string | Root directory for the session |
| `ttlSeconds` | yes | integer | Session lifetime in seconds |
| `readonly` | yes | boolean | Read-only mode |
| `allowUpload` | yes | boolean | Upload permission |
| `allowEdit` | yes | boolean | Edit permission |
| `allowDelete` | yes | boolean | Delete permission |

### Validation

Rules:

- `root` must be non-empty
- `ttlSeconds` must be between `1` and `2592000` inclusive
- `root` is cleaned and converted to an absolute path before the session is stored

### Successful response

Status:

- `200 OK`

Body:

- JSON `Session`

### Rate limit

Current in-memory rate limit:

- maximum `10` session creations per minute

### Capacity limit

Current in-memory session store limit:

- maximum `1000` active sessions

### Error responses

| Status | Body | When |
| --- | --- | --- |
| `400 Bad Request` | `Invalid request body` | Invalid JSON |
| `400 Bad Request` | `root is required` | Missing or empty `root` |
| `400 Bad Request` | `ttlSeconds must be between 1 and 2592000` | TTL out of range |
| `409 Conflict` | `Max sessions reached` | In-memory active session limit reached |
| `429 Too Many Requests` | `Rate limit exceeded` | More than 10 session creations in one minute |
| `500 Internal Server Error` | `Internal server error` | Token generation or other unexpected failure |

## Method Handling Notes

### `/sessions`

Only `POST` is supported.

Any other method returns:

- `405 Method Not Allowed`

### `/api/file`

The route is wired for `GET` and `PUT`.

Methods other than `GET` and `PUT` return:

- `405 Method Not Allowed`

### `/api/upload`

The route is registered without explicit method filtering in the handler.

Expected client usage:

- `POST`

### Other JSON mutation endpoints

These endpoints enforce method checks:

- `/api/mkdir`
- `/api/rename`
- `/api/delete`
- `/api/zip`
- `/api/extract`

Allowed methods:

- `POST` for `mkdir`, `rename`, `zip`, `extract`
- `DELETE` for `delete`

## CORS

The server currently sets:

```http
Access-Control-Allow-Origin: *
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
Access-Control-Allow-Headers: Content-Type, Authorization
```

For `OPTIONS` requests, the server returns:

- `200 OK`

## Path Rules and Security Notes

The server validates requested paths against the session root.

Current protections:

- absolute path normalization
- prefix check against the session root
- symlink resolution to prevent escaping through symlink traversal
- parent symlink validation for not-yet-created files

Behavior notes:

- many path validation failures are intentionally returned to clients as generic internal errors
- zip extraction additionally blocks ZipSlip-style entries and symlink entries

## UI Asset Routes

Any non-`/api/` route on the UI/API port is handled as a UI asset or SPA route.

Behavior:

- existing non-directory files under the configured UI directory are served directly
- unknown routes fall back to `index.html`
- only `GET` and `HEAD` are accepted

Possible responses:

| Status | When |
| --- | --- |
| `200 OK` | Asset or SPA shell served |
| `405 Method Not Allowed` | Method is not `GET` or `HEAD` |
| `503 Service Unavailable` | UI directory is not configured |

## Example Flows

## Create a writable session

```bash
curl -X POST http://localhost:10014/sessions \
  -H "Content-Type: application/json" \
  -d '{
    "root": "/srv/data",
    "ttlSeconds": 3600,
    "readonly": false,
    "allowUpload": true,
    "allowEdit": true,
    "allowDelete": true
  }'
```

## Use the session to list files

```bash
curl -H "Authorization: Bearer <sessionCode>" \
  "http://localhost:10015/api/list?path=/"
```

## Read a file

```bash
curl -H "Authorization: Bearer <sessionCode>" \
  "http://localhost:10015/api/file?path=/notes.txt"
```

## Replace a file

```bash
curl -X PUT \
  -H "Authorization: Bearer <sessionCode>" \
  --data-binary @./notes.txt \
  "http://localhost:10015/api/file?path=/notes.txt"
```

## Upload a file

```bash
curl -X POST \
  -H "Authorization: Bearer <sessionCode>" \
  -F "file=@./photo.jpg" \
  -F "path=/uploads" \
  "http://localhost:10015/api/upload"
```

## Create a directory

```bash
curl -X POST \
  -H "Authorization: Bearer <sessionCode>" \
  -H "Content-Type: application/json" \
  -d '{"path":"/new-folder"}' \
  "http://localhost:10015/api/mkdir"
```

## Delete a file

```bash
curl -X DELETE \
  -H "Authorization: Bearer <sessionCode>" \
  "http://localhost:10015/api/delete?path=/old.txt"
```

## Archive a directory

```bash
curl -X POST \
  -H "Authorization: Bearer <sessionCode>" \
  -H "Content-Type: application/json" \
  -d '{"source":"/docs","target":"/docs.zip"}' \
  "http://localhost:10015/api/zip"
```

## Extract an archive

```bash
curl -X POST \
  -H "Authorization: Bearer <sessionCode>" \
  -H "Content-Type: application/json" \
  -d '{"source":"/docs.zip","target":"/restored"}' \
  "http://localhost:10015/api/extract"
```

## Client Recommendations

- use `Authorization: Bearer <token>`, not query-string tokens
- treat error bodies as plain text
- do not rely on structured JSON errors
- do not rely on unsupported methods returning `405`
- enforce your own timeout and retry policy in clients
- keep the management port private or protected by a reverse proxy or firewall
- if exposing this beyond a trusted network, add authentication in front of `/sessions`
