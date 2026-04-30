# HubFly Files - To-Do List

## 1. Project Setup
- [x] Initialize Go module.
- [x] Initialize React frontend project.
- [x] Set up basic backend directory structure (`cmd`, `internal`).
- [ ] Configure `.gitignore` for Go and Node.js.

## 2. Backend Implementation (Go)
- [x] Implement `internal/config` (Parse flags/env for ports 8080 and 9090).
- [x] Implement `internal/sessions` (Session creation, validation, TTL, memory store).
- [x] Implement `internal/filesystem` (File listing, reading, writing, deleting, path validation/security).
- [x] Implement `internal/archive` (Zip, Unzip functions).
- [x] Implement `internal/server` (HTTP handlers, Authentication middleware).
  - [x] `GET /api/list?path=`
  - [x] `GET /api/file?path=`
  - [x] `PUT /api/file?path=`
  - [x] `POST /api/upload`
  - [x] `POST /api/mkdir`
  - [x] `POST /api/rename`
  - [x] `DELETE /api/delete`
  - [x] `POST /api/zip`
  - [x] `POST /api/extract`
- [x] Implement Management Server (`POST /sessions` on port 9090).
- [x] Implement Demo mode handling when no session is provided.
- [x] Integrate servers in `main.go` and add CORS support.

## 3. Frontend Implementation (React)
- [x] Set up basic styling (Vanilla CSS, variables for Light/Dark mode).
- [x] Implement Main Container Layout (Centered, max-width, rounded corners, soft shadow).
- [x] Implement Breadcrumb & Top actions header (Upload, New, Refresh, View Toggle).
- [x] Implement File Explorer Views:
  - [x] List View (compact rows, hover actions).
  - [x] Grid View (medium cards, centered icons).
- [x] Implement API client (`src/api`).
- [x] Implement File Actions (Rename, Delete, Download).
- [x] Implement File Editor / Preview (Text editor, Image preview).
- [x] Implement Drag-and-drop upload.
- [x] Implement Zip / Extract UI.
- [x] Implement Search within directory.
- [x] Implement Empty folder & Session expired states (handled via error states).

## 4. Final Polish & Testing
- [x] Verify security rules (Path traversal blocking).
- [x] Verify UI pixel-perfect spacing and smooth transitions.
- [x] End-to-end testing of session flow.
