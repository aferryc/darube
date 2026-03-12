const net = require("net");

/**
 * Finds an open port by binding to port 0 (OS assigns a free port).
 * @returns {Promise<number>} The allocated port number
 */
function getFreePort() {
  return new Promise((resolve, reject) => {
    console.log("[Utils] Attempting to find a free port...");
    const srv = net.createServer();
    srv.unref();
    srv.on("error", (err) => {
      console.error("[Utils] Error in getFreePort:", err);
      reject(err);
    });
    // Bind to localhost explicitly to avoid environments that disallow 0.0.0.0 binds.
    srv.listen(0, "127.0.0.1", () => {
      const port = srv.address().port;
      console.log(`[Utils] Successfully allocated port: ${port}`);
      srv.close(() => resolve(port));
    });
  });
}

module.exports = { getFreePort };
