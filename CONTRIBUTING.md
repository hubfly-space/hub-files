# Contributing

Keep contributions focused and easy to review.

## Before a pull request

Run the checks that match your change:

```bash
go test ./...
```

```bash
cd frontend
npm ci
npm run build
```

Update [docs/USAGE.md](docs/USAGE.md) when you change setup, sessions, config, API behavior, or file operations.

## Commits

Use short messages that say what changed:

```text
server: validate FTP session roots
frontend: add upload progress state
docs: document host mounts
```

Sign off commits if you are contributing from a fork:

```bash
git commit -s
```

## Do not commit

- local `.env` files
- logs
- built frontend output
- `node_modules`
- runtime sqlite databases
- credentials, tokens, SMB passwords, or FTP passwords

## Security

Do not open a public issue for a vulnerability. Report it privately. contact a maintainer first.
