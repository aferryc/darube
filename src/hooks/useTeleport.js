import { useRef, useState } from "react";

// useTeleport centralizes the in-app Teleport (tsh) login flow: checking session
// status, opening the login modal, and letting other hooks recover from an
// expired session. It mirrors the app's other hooks (useConnections, useTabs).
//
// Returns:
//   loginPrompt          state driving the TeleportLoginModal (null when hidden)
//   resolveLogin(ok)     resolve the pending login prompt (modal success/cancel)
//   ensureLogin(opts)    ensure a live session, opening the modal if needed
//   recoverRef           ref whose .current(connId) re-logs in a connection
//   getProxyForProfileId resolve a saved profile id to its proxy/host string
export function useTeleport(apiUrl, settings, connections) {
  const [loginPrompt, setLoginPrompt] = useState(null);
  const loginReqRef = useRef(null);
  const recoverRef = useRef(null);

  const getProxyForProfileId = (profileId) => {
    const list = settings?.teleport_profiles || [];
    const p = list.find((x) => String(x?.id || "") === String(profileId || ""));
    return String(p?.profile || "").trim();
  };

  const fetchStatus = async (profileName) => {
    const p = String(profileName || "").trim();
    const qs = p ? `?profile=${encodeURIComponent(p)}` : "";
    const res = await fetch(`${apiUrl}/api/teleport/status${qs}`);
    const data = await res.json().catch(() => ({}));
    return data || {};
  };

  // Opens the modal and returns a promise that resolves true/false once the user
  // completes or cancels. Concurrent callers share the one pending prompt.
  const requestLogin = ({ defaultProfileId, reason }) => {
    if (loginReqRef.current?.promise) return loginReqRef.current.promise;
    let resolve = null;
    const promise = new Promise((r) => {
      resolve = r;
    });
    loginReqRef.current = { promise, resolve };
    setLoginPrompt({
      defaultProfileId: String(defaultProfileId || ""),
      reason: String(reason || ""),
    });
    return promise;
  };

  const resolveLogin = (ok) => {
    const req = loginReqRef.current;
    loginReqRef.current = null;
    setLoginPrompt(null);
    req?.resolve?.(!!ok);
  };

  // Ensures a live Teleport session. Returns true when already logged in or the
  // user completes login; false when tsh is unavailable or login is cancelled.
  const ensureLogin = async ({ profileName, profileId, reason } = {}) => {
    const wantedProfile = String(profileName || "").trim();

    const st = await fetchStatus(wantedProfile);
    if (st?.tsh_available === false) {
      alert(st?.error || "Teleport (tsh) is not available on this machine.");
      return false;
    }
    if (st?.logged_in) return true;

    const ok = await requestLogin({
      defaultProfileId: profileId,
      reason: String(reason || st?.error || "Teleport session is missing or expired."),
    });
    if (!ok) return false;

    const st2 = await fetchStatus(wantedProfile);
    if (st2?.logged_in) return true;

    alert(st2?.error || "Teleport login did not complete. Please try again.");
    return false;
  };

  // Ensures a live session for a specific connection, resolving its proxy/profile
  // automatically. This is the form used everywhere a connection is known.
  const ensureLoginForConnection = (conn, reason) => {
    const proxy =
      getProxyForProfileId(conn?.teleport_profile_id) || conn?.teleport_profile;
    return ensureLogin({
      profileName: proxy,
      profileId: conn?.teleport_profile_id,
      reason: reason || "Your Teleport session has expired. Log in to continue.",
    });
  };

  // Reassigned each render so the closure stays current; other hooks call it via
  // the ref to recover from an expired session mid-operation.
  recoverRef.current = (connectionId) =>
    ensureLoginForConnection(
      connections?.connections?.find((c) => c.id === connectionId),
    );

  return { loginPrompt, resolveLogin, ensureLogin, ensureLoginForConnection, recoverRef };
}
