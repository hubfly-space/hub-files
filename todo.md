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
- [x] Set up basic styling (Tailwind CSS).
- [x] Integrate Shadcn UI components.
- [x] Implement Main Container Layout (Centered, glassmorphism, floating card).
- [x] Implement Breadcrumb (Lucide icons, polished).
- [x] Implement Toolbar (Search integration, tooltips).
- [x] Implement File Explorer Views:
  - [x] List View (Framer Motion animations, polished).
  - [x] Grid View (Compact cards, hover states).
- [x] Implement Custom Modals (Rename, Delete, New Folder) using Shadcn Dialog.
- [x] Implement Feedback System (Shadcn Toast).
- [x] Implement File Editor / Preview (Advanced layout, save feedback).
- [x] Implement Drag-and-drop upload (Visual feedback).
- [x] Implement Search (Focused, immediate).

## 4. Final Polish & Testing
- [x] Verify security rules (Path traversal blocking).
- [x] Verify UI pixel-perfect spacing and smooth transitions.
- [x] End-to-end testing of session flow.
- [x] Advanced UI remake with Shadcn and Tailwind.
