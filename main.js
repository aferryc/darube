const { app, BrowserWindow, ipcMain, dialog } = require('electron');
const path = require('path');
const { spawn } = require('child_process');
const fs = require('fs');
const { getFreePort } = require('./scripts/utils');

let mainWindow;
let engineProcess = null;
let savedEnginePort = null;

function createWindow(enginePort) {
  mainWindow = new BrowserWindow({
    width: 1200,
    height: 800,
    webPreferences: {
      nodeIntegration: true,
      contextIsolation: false
    }
  });

  // Load from Vite dev server if running locally, otherwise load built files
  if (process.env.NODE_ENV === 'development') {
    const vitePort = process.env.VITE_PORT || 5173;
    mainWindow.loadURL(`http://localhost:${vitePort}?enginePort=${enginePort}`);
    mainWindow.webContents.openDevTools();
  } else {
    mainWindow.loadFile(path.join(__dirname, 'dist', 'index.html'), { query: { enginePort: enginePort.toString() } });
  }

  mainWindow.on('closed', function () {
    mainWindow = null;
  });
}

async function startEngine() {
  console.log("Locating free port for Darube Go Engine...");
  savedEnginePort = await getFreePort();
  
  // Decide where the binary is based on dev or prod
  let enginePath = path.join(__dirname, 'engine', 'engine');
  
  // In production the binary is bundled via extraResources
  if (app.isPackaged) {
    enginePath = path.join(process.resourcesPath, 'engine', 'engine');
  }
  
  if (!fs.existsSync(enginePath)) {
    const msg = `Darube engine binary not found at:\n${enginePath}\n\nPlease rebuild the app with "make build".`;
    console.error(`ERROR: ${msg}`);
    dialog.showErrorBox('Engine Start Failed', msg);
    app.quit();
    return null;
  }

  // Use a writable data directory so the engine can persist connections.json,
  // workspace.json, etc. In a packaged .app the Resources dir is read-only.
  const dataDir = app.getPath('userData');
  if (!fs.existsSync(dataDir)) {
    fs.mkdirSync(dataDir, { recursive: true });
  }

  engineProcess = spawn(enginePath, [], {
    cwd: dataDir,
    env: { ...process.env, PORT: savedEnginePort }
  });

  engineProcess.stdout.on('data', (data) => {
    console.log(`[Engine:${savedEnginePort}]: ${data}`);
  });

  engineProcess.stderr.on('error', (error) => {
    console.error(`[Engine ERROR]: ${error}`);
  });

  engineProcess.on('close', (code) => {
    console.log(`[Engine] process exited with code ${code}`);
  });
  
  return savedEnginePort;
}

app.on('ready', async () => {
  const enginePort = await startEngine();
  if (!enginePort) return; // startEngine already showed error dialog and quit
  createWindow(enginePort);

  // Setup IPC handlers
  ipcMain.handle('dialog:openDirectory', async () => {
    const { canceled, filePaths } = await dialog.showOpenDialog(mainWindow, {
      properties: ['openDirectory', 'createDirectory']
    });
    if (canceled) {
      return null;
    } else {
      return filePaths[0];
    }
  });

  ipcMain.handle('dialog:openFile', async () => {
    const { canceled, filePaths } = await dialog.showOpenDialog(mainWindow, {
      properties: ['openFile']
    });
    if (canceled) {
      return null;
    } else {
      return filePaths[0];
    }
  });
});

app.on('window-all-closed', function () {
  // Quit the app entirely on all platforms when closed, to ensure Go Engine teardown
  app.quit();
});

app.on('activate', function () {
  if (mainWindow === null) {
    createWindow(savedEnginePort);
  }
});

// Kill the engine when Electron quits
app.on('will-quit', () => {
    if (engineProcess !== null) {
        console.log("Killing engine process avant quit...");
        engineProcess.kill();
    }
});
