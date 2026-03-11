const { spawnSync } = require('child_process');
const fs = require('fs');
const os = require('os');
const path = require('path');

const engineDir = path.resolve(__dirname, '..', 'engine');
const defaultCacheDir = path.join(os.tmpdir(), 'darube-go-build-cache');

const gocache = process.env.GOCACHE || defaultCacheDir;
fs.mkdirSync(gocache, { recursive: true });

const res = spawnSync('go', ['test', './...'], {
  cwd: engineDir,
  env: {
    ...process.env,
    // Some sandboxed environments forbid using the default per-user cache dir.
    // Point Go's build cache at a writable temp folder to keep tests reliable.
    GOCACHE: gocache
  },
  stdio: 'inherit',
  shell: false
});

process.exit(typeof res.status === 'number' ? res.status : 1);

