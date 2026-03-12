const { contextBridge, ipcRenderer, clipboard } = require('electron');

contextBridge.exposeInMainWorld('darube', {
  openDirectory: () => ipcRenderer.invoke('dialog:openDirectory'),
  openFile: () => ipcRenderer.invoke('dialog:openFile'),
  clipboard: {
    writeText: (text) => clipboard.writeText(String(text ?? '')),
    readText: () => clipboard.readText(),
  },
});

