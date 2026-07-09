// Detects engine errors that mean the Teleport (tsh) session is missing or
// expired and the user needs to log in again. These are the messages emitted by
// engine/teleport/*.go (e.g. "teleport: not logged in (run `tsh login ...`)").
export function isTeleportAuthError(msg) {
  const s = String(msg || "").toLowerCase();
  if (!s) return false;
  return (
    s.includes("tsh login") ||
    (s.includes("teleport") && s.includes("not logged in")) ||
    s.includes("credentials are missing or expired")
  );
}

// Pulls the proxy/profile out of an engine error like
// "teleport: not logged in (run `tsh login --proxy=example.teleport.sh`)".
// Returns "" if none is present.
export function teleportProxyFromError(msg) {
  const m = String(msg || "").match(/--proxy=([^\s`'")]+)/);
  return m ? m[1] : "";
}
