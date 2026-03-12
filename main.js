const { app, BrowserWindow, ipcMain, dialog } = require("electron");
const path = require("path");
const { spawn } = require("child_process");
const fs = require("fs");
const { getFreePort } = require("./scripts/utils");

let mainWindow;
let engineProcess = null;
let savedEnginePort = null;

// --- Logging Setup ---
const logFile = path.join(app.getPath("userData"), "app.log");
function logMessage(msg) {
  const time = new Date().toISOString();
  const fullMsg = `[${time}] ${msg}\n`;
  console.log(msg); // Keep stdout for dev
  try {
    fs.appendFileSync(logFile, fullMsg);
  } catch (e) {
    // ignore logging errors
  }
}

logMessage("--- Darube Starting ---");
logMessage(`Log file: ${logFile}`);

function createWindow(enginePort) {
  logMessage(`Creating window with enginePort: ${enginePort}`);
  mainWindow = new BrowserWindow({
    width: 1200,
    height: 800,
    webPreferences: {
      preload: path.join(__dirname, "preload.js"),
      nodeIntegration: false,
      contextIsolation: true,
    },
  });

  // Load from Vite dev server if running locally, otherwise load built files
  if (process.env.NODE_ENV === "development") {
    const vitePort = process.env.VITE_PORT || 5173;
    logMessage(`Development mode: loading from http://localhost:${vitePort}`);
    mainWindow.loadURL(`http://localhost:${vitePort}?enginePort=${enginePort}`);
    mainWindow.webContents.openDevTools();
  } else {
    const indexPath = path.join(__dirname, "dist", "index.html");
    logMessage(`Production mode: loading ${indexPath}`);
    if (!fs.existsSync(indexPath)) {
      logMessage(`CRITICAL: index.html not found at ${indexPath}`);
      dialog.showErrorBox(
        "Resource Not Found",
        `Frontend assets missing at:\n${indexPath}`,
      );
    }
    mainWindow.loadFile(indexPath, {
      query: { enginePort: enginePort.toString() },
    });
  }

  mainWindow.on("closed", function () {
    mainWindow = null;
  });
}

async function startEngine() {
  try {
    logMessage("Locating free port for Darube Go Engine...");
    savedEnginePort = await getFreePort();
    logMessage(`Allocated engine port: ${savedEnginePort}`);

    if (!savedEnginePort) {
      throw new Error("Port allocation failed (returned null/undefined)");
    }

    // Decide where the binary is based on dev or prod
    let enginePath = path.join(__dirname, "engine", "bin", "engine");

    // In production the binary is bundled via extraResources
    if (app.isPackaged) {
      enginePath = path.join(process.resourcesPath, "engine", "bin", "engine");
    }

    logMessage(`Engine binary path: ${enginePath}`);

    if (!fs.existsSync(enginePath)) {
      const msg = `Darube engine binary not found at:\n${enginePath}\n\nPlease rebuild the app with "make build".`;
      logMessage(`ERROR: ${msg}`);
      dialog.showErrorBox("Engine Start Failed", msg);
      app.quit();
      return null;
    }

    // Use a writable data directory so the engine can persist connections.json,
    // workspace.json, etc. In a packaged .app the Resources dir is read-only.
    const dataDir = app.getPath("userData");
    if (!fs.existsSync(dataDir)) {
      fs.mkdirSync(dataDir, { recursive: true });
    }
    logMessage(`Engine working directory: ${dataDir}`);

    engineProcess = spawn(enginePath, [], {
      cwd: dataDir,
      env: { ...process.env, PORT: savedEnginePort.toString() },
    });

    engineProcess.on("error", (err) => {
      logMessage(`[Engine SPAWN ERROR]: ${err.message}`);
      dialog.showErrorBox(
        "Engine Spawn Error",
        `Failed to start engine: ${err.message}`,
      );
    });

    engineProcess.stdout.on("data", (data) => {
      logMessage(`[Engine Out]: ${data}`);
    });

    engineProcess.stderr.on("data", (data) => {
      logMessage(`[Engine Err]: ${data}`);
    });

    engineProcess.on("close", (code) => {
      logMessage(`[Engine] process exited with code ${code}`);
    });

    return savedEnginePort;
  } catch (err) {
    logMessage(`CRITICAL FAILURE in startEngine: ${err.stack || err}`);
    dialog.showErrorBox(
      "Initialization Error",
      `A critical error occurred while starting the engine:\n${err.message}`,
    );
    app.quit();
    return null;
  }
}

app.on("ready", async () => {
  logMessage("App Ready event triggered");
  const enginePort = await startEngine();
  if (!enginePort) {
    logMessage("Engine start returned null, skipping window creation.");
    return;
  }
  createWindow(enginePort);

  // Setup IPC handlers
  ipcMain.handle("dialog:openDirectory", async () => {
    const { canceled, filePaths } = await dialog.showOpenDialog(mainWindow, {
      properties: ["openDirectory", "createDirectory"],
    });
    if (canceled) {
      return null;
    } else {
      return filePaths[0];
    }
  });

  ipcMain.handle("dialog:openFile", async () => {
    const { canceled, filePaths } = await dialog.showOpenDialog(mainWindow, {
      properties: ["openFile"],
    });
    if (canceled) {
      return null;
    } else {
      return filePaths[0];
    }
  });
});

app.on("window-all-closed", function () {
  logMessage("All windows closed, quitting app...");
  app.quit();
});

app.on("activate", function () {
  if (mainWindow === null) {
    createWindow(savedEnginePort);
  }
});

// Kill the engine when Electron quits
app.on("will-quit", () => {
  if (engineProcess !== null) {
    logMessage("Killing engine process avant quit...");
    try {
      engineProcess.kill();
    } catch (e) {
      logMessage(`Error killing engine: ${e.message}`);
    }
  }
});
