# HubFly Files - To-Do List

## 1. Project Setup
- [ ] Initialize Go module.
- [ ] Initialize React frontend project (Vite + React + TS recommended for speed).
- [ ] Set up basic backend directory structure (`cmd`, `internal`).
- [ ] Configure `.gitignore` for Go and Node.js.

## 2. Backend Implementation (Go)
- [ ] Implement `internal/config` (Parse flags/env for ports 8080 and 9090).
- [ ] Implement `internal/sessions` (Session creation, validation, TTL, memory store).
- [ ] Implement `internal/filesystem` (File listing, reading, writing, deleting, path validation/security).
- [ ] Implement `internal/archive` (Zip, Unzip functions).
- [ ] Implement `internal/server` (HTTP handlers, Authentication middleware).
  - [ ] `GET /api/list?path=`
  - [ ] `GET /api/file?path=`
  - [ ] `PUT /api/file?path=`
  - [ ] `POST /api/upload`
  - [ ] `POST /api/mkdir`
  - [ ] `POST /api/rename`
  - [ ] `DELETE /api/delete`
  - [ ] `POST /api/zip`
  - [ ] `POST /api/extract`
- [ ] Implement Management Server (`POST /sessions` on port 9090).
- [ ] Implement Demo mode handling when no session is provided.

## 3. Frontend Implementation (React)
- [ ] Set up basic styling (Vanilla CSS, variables for Light/Dark mode).
- [ ] Implement Main Container Layout (Centered, max-width, rounded corners, soft shadow).
- [ ] Implement Breadcrumb & Top actions header (Upload, New, Refresh, View Toggle).
- [ ] Implement File Explorer Views:
  - [ ] List View (compact rows, hover actions).
  - [ ] Grid View (medium cards, centered icons).
- [ ] Implement API client (`src/api`).
- [ ] Implement File Actions (Rename, Delete, Download).
- [ ] Implement File Editor / Preview (Text editor, Image preview).
- [ ] Implement Drag-and-drop upload.
- [ ] Implement Zip / Extract UI.
- [ ] Implement Empty folder & Session expired states.

## 4. Final Polish & Testing
- [ ] Verify security rules (Path traversal blocking).
- [ ] Verify UI pixel-perfect spacing and smooth transitions.
- [ ] End-to-end testing of session flow.
