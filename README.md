# Darube

Darube is a desktop database manager (think DBeaver / DataGrip) built with Electron + React, backed by a local Go “engine”.

## Features

- Connections: create/save/edit/delete, connect/disconnect/reconnect
- Databases: PostgreSQL, MySQL/MariaDB, SQL Server, SQLite, Oracle
- Redis: connect, run commands, export results, edit values from the results pane
- Query tabs: multiple tabs, autosaved workspace, tabs are bound to connections
- Scripting: run JavaScript (goja) to query across multiple connections
- Schema explorer: databases/schemas/tables/columns, view DML, view indexes
- Export: CSV / JSON / Excel for SQL results; JSON/CSV/Excel for Redis results
- Folders: organize connections (SQL and Redis) via drag and drop

## Architecture

- Electron main process spawns the Go engine on a free local port and passes it to the UI as `?enginePort=...`.
- The Go engine is a local HTTP server that:
  - persists connection configs and folders as JSON files
  - keeps active connections in memory
  - runs SQL queries/mutations and Redis commands

## Development

### Prereqs

- Node.js + npm
- Go (engine)

### Common Commands

```sh
# Start dev (Vite + Electron)
make dev

# Build the Go engine binary (required by Electron)
make build-engine

# Package the full app (engine + Vite + electron-builder)
make build

# Run tests
make test
```

### Notes

- If you change Go engine code, you must rebuild the engine binary and restart the Electron app:
  - `make build-engine`
- Workspace persistence:
  - Query tabs are autosaved via the engine to `workspace.json`.
  - Tabs are bound to a specific connection. Clicking another connection switches to its existing tab(s); if none exist, Darube prompts to create a new tab or rebind the current tab.

## Scripting

Darube scripts run JavaScript using goja inside the local Go engine. Scripts are synchronous and interact only via a small wrapper API.

### API

```js
const pg = db.conn("prod-postgres") // id (or unique connection name)
const redis = db.conn("cache")      // id (or unique connection name)

const users = pg.query("SELECT id FROM users")
for (const u of users) {
  redis.set(`user:${u.id}`, "active")
}

sleep(250) // or utils.sleep(250)
console.log(utils.uuidv7(), utils.now(), utils.nowUnixMs())
```

Connection methods:

- SQL: `query(sql) -> []object`, `exec(sql) -> number`, `one(sql) -> object`, `scalar(sql) -> primitive`
- Redis: `set(key, value)`, `get(key)`, `del(key)`

Utilities:

- `sleep(ms)` / `utils.sleep(ms)`
- `utils.uuidv7() -> string`
- `utils.now() -> string` (RFC3339)
- `utils.nowUnixMs() -> number`
