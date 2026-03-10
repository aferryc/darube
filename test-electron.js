const electron = require('electron');
console.log("App Object exists?", !!electron.app);
console.log("BrowserWindow exists?", !!electron.BrowserWindow);
console.log("Keys:", Object.keys(electron).join(', '));
if (electron.app) electron.app.quit();
