# Darube

Darube is a one-stop backend tool for developers. It brings SQL databases, NoSQL stores, API testing, and scripting into a single desktop app so you can connect, query, automate, and inspect data without juggling multiple tools.

## Highlights

<img width="1197" height="794" alt="image" src="https://github.com/user-attachments/assets/11c65531-117e-4c6b-b64e-79c491c4bb17" />


- SQL connections: PostgreSQL, MySQL/MariaDB, SQL Server, SQLite, Oracle
- NoSQL connections: Redis, MongoDB, Cassandra, Elasticsearch, OpenSearch
- API testing: HTTP and gRPC
- Scripting: run JavaScript (goja) across multiple connections
- Query tabs: multiple tabs per connection with autosaved workspace
- Schema explorer: databases, schemas, tables, columns, indexes
- Export: CSV / JSON / Excel for SQL and Redis results
- Folders: organize connections with drag-and-drop

## How It Works

Darube is an Electron + React app backed by a local Go engine.

- The Electron main process starts the Go engine on a free local port.
- The UI talks to the engine over HTTP using that port.
- The engine persists connection configs and workspace state locally.

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

# Package for Windows
make build-win

# Package for Linux
make build-linux

# Run tests
make test
```

### Notes

- If you change Go engine code, rebuild the engine binary and restart the Electron app:
  - `make build-engine`
- Workspace persistence:
  - Query tabs are autosaved to `workspace.json`.
  - Tabs are bound to a connection. Clicking another connection switches to its existing tabs; if none exist, Darube prompts to create a new tab or rebind the current tab.

## Scripting

Darube scripts run JavaScript using goja inside the local Go engine. Scripts are synchronous and interact only via a small wrapper API.

### API

```js
const pg = db.conn("prod-postgres"); // id (or unique connection name)
const redis = db.conn("cache"); // id (or unique connection name)

const users = pg.query("SELECT id FROM users");
for (const u of users) {
  redis.set(`user:${u.id}`, "active");
}

sleep(250); // or utils.sleep(250)
console.log(utils.uuidv7(), utils.now(), utils.nowUnixMs());
```

Connection methods:

- SQL: `query(sql) -> []object`, `exec(sql) -> number`, `one(sql) -> object`, `scalar(sql) -> primitive`
- Redis: `set(key, value)`, `get(key)`, `del(key)`
- HTTP: `get(url) -> string`, `post(url, body) -> string`, `put(url, body) -> string`, `delete(url) -> string`
- gRPC: `call(service, method, body) -> interface`

Utilities:

- `sleep(ms)` / `utils.sleep(ms)`
- `utils.uuidv7() -> string`
- `utils.now() -> string` (RFC3339)
- `utils.nowUnixMs() -> number`
