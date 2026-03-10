const { spawn } = require('child_process');
const path = require('path');
const { getFreePort } = require('./utils');

async function startDevServers() {
  try {
    const vitePort = await getFreePort();
    console.log(`[Dev Orchestrator] Allocated Vite Port: ${vitePort}`);

    // 1. Boot Vite explicitly on the free port
    const viteProcess = spawn('npx', ['vite', '--port', vitePort], {
      cwd: path.resolve(__dirname, '..'),
      env: { ...process.env },
      shell: true,
      stdio: 'pipe'
    });

    viteProcess.stdout.on('data', data => console.log(`[Vite]: ${data}`));
    viteProcess.stderr.on('data', data => console.error(`[Vite ERROR]: ${data}`));

    // 2. Wait a moment for Vite to bind
    await new Promise(resolve => setTimeout(resolve, 2000));

    // 3. Boot Electron, passing the VITE_PORT explicitly as an ENV var
    console.log(`[Dev Orchestrator] Spawning Electron Window...`);
    const electronProcess = spawn('npx', ['cross-env', 'NODE_ENV=development', `VITE_PORT=${vitePort}`, 'electron', '.'], {
      cwd: path.resolve(__dirname, '..'),
      env: { ...process.env, VITE_PORT: vitePort, NODE_ENV: 'development' },
      shell: true,
      stdio: 'inherit' // Inherit stdio to pipe Go Engine logs straight to the shell natively
    });

    electronProcess.on('close', code => {
      console.log(`[Dev Orchestrator] Electron exited with code ${code}, shutting down Vite...`);
      viteProcess.kill();
      process.exit(code);
    });

    // Catch interrupts
    process.on('SIGINT', () => {
      electronProcess.kill();
      viteProcess.kill();
      process.exit();
    });

  } catch (err) {
    console.error(`[Dev Orchestrator] Failed to start environments:`, err);
    process.exit(1);
  }
}

startDevServers();
