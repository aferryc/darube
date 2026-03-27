import { useState, useEffect } from "react";
import Split from "react-split";

import { SqlAutocomplete } from "./components/SqlAutocomplete";
import { RedisAutocomplete } from "./components/RedisAutocomplete";
import { Sidebar } from "./components/Sidebar";
import { QueryTabs } from "./components/QueryTabs";
import { ResultsPane } from "./components/ResultsPane";
import { RedisPane } from "./components/RedisPane";
import { ScriptPane } from "./components/ScriptPane";
import { HttpRequestPane } from "./components/HttpRequestPane";
import { GrpcRequestPane } from "./components/GrpcRequestPane";
import { ConnectionModal } from "./components/ConnectionModal";
import {
  ContextMenu,
  ExportModal,
  HelpModal,
  ConnectionSwitchModal,
} from "./components/Modals";
import { SettingsModal } from "./components/SettingsModal";

import { useConnections } from "./hooks/useConnections";
import { useTabs } from "./hooks/useTabs";
import { useEditableGrid } from "./hooks/useEditableGrid";
import { useContextMenu } from "./hooks/useContextMenu";
import { useExport } from "./hooks/useExport";
import { applyBoxCut, getBoxSelectionText } from "./utils/boxSelection";
import { formatBytes } from "./utils/formatBytes";
import logoApp from "./assets/darube.png";

const params = new URLSearchParams(window.location.search);
const enginePort = params.get("enginePort") || "3000";
const apiUrl = `http://localhost:${enginePort}`;

const EMPTY_FORM = {
  connection_name: "",
  db_type: "postgres",
  host: "",
  port: 5432,
  dbname: "",
  file_path: "",
  user: "",
  password: "",
  enable_ssl: false,
  ca_cert_path: "",
  client_cert_path: "",
  client_key_path: "",
  folder_id: "",

  // API connections (HTTP / gRPC)
  base_url: "",
  address: "",
  tls: false,
  insecure_tls: false,
  server_name: "",
  default_headers: [],
  auth_type: "none",
  bearer_token: "",
  auth_username: "",
  auth_password: "",

  // Redis extras
  is_cluster: false,

  // Teleport (tsh) options
  teleport_enabled: false,
  teleport_profile_id: "",
  teleport_db_service: "",
  // Legacy inline fields (backwards compat when no profile_id)
  teleport_cluster: "",
  teleport_user: "",
  teleport_profile: "",
};

function App() {
  // ── Shared UI state ──────────────────────────────────────────────────────
  const [activeId, setActiveId] = useState(null);
  const [loading, setLoading] = useState(false);
  const [showModal, setShowModal] = useState(false);
  const [showHelpModal, setShowHelpModal] = useState(false);
  const [showSettingsModal, setShowSettingsModal] = useState(false);
  const [switchPrompt, setSwitchPrompt] = useState(null); // { targetId, forceExpand }
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [sidebarWidth, setSidebarWidth] = useState(() => {
    const raw = localStorage.getItem("darube.sidebarWidth");
    const n = raw ? Number(raw) : NaN;
    if (!Number.isFinite(n)) return 260;
    return Math.max(200, Math.min(520, n));
  });
  const [layoutDirection, setLayoutDirection] = useState("vertical");
  const [settings, setSettings] = useState(null);
  const [tableSizeStatus, setTableSizeStatus] = useState(null);
  const [formData, setFormData] = useState(EMPTY_FORM);
  const [editingId, setEditingId] = useState(null);

  // ── Hooks ─────────────────────────────────────────────────────────────────
  const connections = useConnections(apiUrl);
  const tabs = useTabs(apiUrl, activeId, setLoading, settings);
  const grid = useEditableGrid(
    apiUrl,
    activeId,
    tabs.activeTab,
    tabs.updateActiveTab,
    tabs.executeQuery,
    setLoading,
  );
  const ctxMenu = useContextMenu();
  const exp = useExport(
    apiUrl,
    activeId,
    tabs.activeTab,
    tabs.updateActiveTab,
    setLoading,
  );
  const activeConn = connections.connections.find((c) => c.id === activeId);
  const activeTabConnId = tabs.activeTab?.connectionId || null;
  const formatStatusTime = (value) => {
    if (!value) return "";
    const dt = new Date(value);
    if (Number.isNaN(dt.getTime())) return "";
    return dt.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  };

  // ── Initial polling ───────────────────────────────────────────────────────
  useEffect(() => {
    connections.fetchConnections();
    connections.fetchFolders();
    const interval = setInterval(connections.fetchConnections, 2000);
    return () => clearInterval(interval);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    try {
      localStorage.setItem("darube.sidebarWidth", String(sidebarWidth));
    } catch {
      // ignore persistence failures (private mode, etc.)
    }
  }, [sidebarWidth]);

  // ── Load settings on mount ───────────────────────────────────────────────
  useEffect(() => {
    fetch(`${apiUrl}/api/settings`)
      .then((r) => r.json())
      .then((data) => {
        if (data && data.success !== false) {
          if (data.layout_direction && (data.layout_direction === "vertical" || data.layout_direction === "horizontal")) {
            setLayoutDirection(data.layout_direction);
          }
          setSettings({
            layout_direction: data.layout_direction || "vertical",
            teleport_profiles: data.teleport_profiles || [],
            global_script_timeout_ms: data.global_script_timeout_ms ?? 0,
            global_query_timeout_ms: data.global_query_timeout_ms ?? 0,
            global_api_timeout_ms: data.global_api_timeout_ms ?? 0,
            max_lines_query: data.max_lines_query ?? 0,
            max_lines_script: data.max_lines_script ?? 0,
            max_lines_body: data.max_lines_body ?? 0,
            large_result_warn_mb: data.large_result_warn_mb ?? 50,
            theme_variant: data.theme_variant || "",
            ui_theme_custom: data.ui_theme_custom || "",
            ui_font_family: data.ui_font_family || "",
            ui_font_size: data.ui_font_size || 0,
            ui_font_color: data.ui_font_color || "",
            ui_text_primary: data.ui_text_primary || "",
            ui_text_muted: data.ui_text_muted || "",
            ui_text_accent: data.ui_text_accent || "",
          });
        }
      })
      .catch(() => {});
  }, [apiUrl]);

  // ── Apply theme / typography to document ─────────────────────────────────
  useEffect(() => {
    if (!settings) return;
    const root = document.documentElement;
    if (settings.ui_font_family) {
      root.style.setProperty("--app-font-family", settings.ui_font_family);
    } else {
      root.style.removeProperty("--app-font-family");
    }
    if (settings.ui_font_size && Number(settings.ui_font_size) > 0) {
      root.style.setProperty("--app-font-size", `${settings.ui_font_size}px`);
    } else {
      root.style.removeProperty("--app-font-size");
    }
    if (settings.ui_font_color) {
      root.style.setProperty("--app-font-color", settings.ui_font_color);
    } else {
      root.style.removeProperty("--app-font-color");
    }
    const primary = settings.ui_text_primary || settings.ui_font_color || "";
    if (primary) {
      root.style.setProperty("--text-main", primary);
    } else {
      root.style.removeProperty("--text-main");
    }
    if (settings.ui_text_muted) {
      root.style.setProperty("--text-muted", settings.ui_text_muted);
    } else {
      root.style.removeProperty("--text-muted");
    }
    if (settings.ui_text_accent) {
      root.style.setProperty("--accent", settings.ui_text_accent);
    } else {
      root.style.removeProperty("--accent");
    }
    if (settings.theme_variant) {
      root.setAttribute("data-theme", settings.theme_variant);
    } else {
      root.removeAttribute("data-theme");
    }
  }, [settings]);

  // ── Close context menu on global click ───────────────────────────────────
  useEffect(() => {
    const close = () => {
      if (ctxMenu.contextMenu.visible) ctxMenu.hideMenu();
    };
    window.addEventListener("click", close);
    const onKeyDown = (e) => {
      if (e.key === "Escape" && ctxMenu.contextMenu.visible) ctxMenu.hideMenu();
    };
    window.addEventListener("keydown", onKeyDown);
    return () => {
      window.removeEventListener("click", close);
      window.removeEventListener("keydown", onKeyDown);
    };
  }, [ctxMenu]);

  // ── Connection form helpers ───────────────────────────────────────────────
  const openNewConnection = () => {
    setEditingId(null);
    setFormData(EMPTY_FORM);
    setShowModal(true);
  };

  const handleEditConnection = async (c, e) => {
    e.stopPropagation();
    setEditingId(c.id);

    if (c.db_type === "http") {
      try {
        const res = await fetch(`${apiUrl}/api/http/${c.id}`);
        const data = await res.json();
        if (data.success && data.config) {
          const cfg = data.config;
          setFormData({
            ...EMPTY_FORM,
            connection_name: cfg.connection_name || c.connection_name || "",
            db_type: "http",
            base_url: cfg.base_url || "",
            default_headers: cfg.default_headers || [],
            folder_id: cfg.folder_id || c.folder_id || "",
            auth_type: cfg.auth?.type || "none",
            bearer_token: cfg.auth?.bearer_token || "",
            auth_username: cfg.auth?.username || "",
            auth_password: cfg.auth?.password || "",
          });
        } else {
          throw new Error(data.error || "Failed to load HTTP config");
        }
      } catch (err) {
        alert("Failed to load HTTP connection: " + err.message);
        setFormData({ ...EMPTY_FORM, connection_name: c.connection_name || "", db_type: "http" });
      }
      setShowModal(true);
      return;
    }

    if (c.db_type === "grpc") {
      try {
        const res = await fetch(`${apiUrl}/api/grpc/${c.id}`);
        const data = await res.json();
        if (data.success && data.config) {
          const cfg = data.config;
          setFormData({
            ...EMPTY_FORM,
            connection_name: cfg.connection_name || c.connection_name || "",
            db_type: "grpc",
            address: cfg.address || "",
            tls: !!cfg.tls,
            insecure_tls: !!cfg.insecure_tls,
            server_name: cfg.server_name || "",
            folder_id: cfg.folder_id || c.folder_id || "",
            auth_type: cfg.auth?.type || "none",
            bearer_token: cfg.auth?.bearer_token || "",
            auth_username: cfg.auth?.username || "",
            auth_password: cfg.auth?.password || "",
          });
        } else {
          throw new Error(data.error || "Failed to load gRPC config");
        }
      } catch (err) {
        alert("Failed to load gRPC connection: " + err.message);
        setFormData({ ...EMPTY_FORM, connection_name: c.connection_name || "", db_type: "grpc" });
      }
      setShowModal(true);
      return;
    }

    setFormData({
      ...EMPTY_FORM,
      connection_name: c.connection_name || "",
      db_type: c.db_type || "postgres",
      host: c.host || "",
      port: typeof c.port === "number" ? c.port : 5432,
      dbname: c.dbname || "",
      file_path: c.file_path || "",
      user: c.user || "",
      password: "",
      enable_ssl: c.enable_ssl || false,
      ca_cert_path: c.ca_cert_path || "",
      client_cert_path: c.client_cert_path || "",
      client_key_path: c.client_key_path || "",
      folder_id: c.folder_id || "",
      teleport_enabled: !!c.teleport_enabled,
      teleport_profile_id: c.teleport_profile_id || "",
      teleport_db_service: c.teleport_db_service || "",
      teleport_cluster: c.teleport_cluster || "",
      teleport_user: c.teleport_user || "",
      teleport_profile: c.teleport_profile || "",
    });
    setShowModal(true);
  };

  const handleConnectNew = async (e) => {
    e.preventDefault();
    try {
      const isHttp = formData.db_type === "http";
      const isGrpc = formData.db_type === "grpc";

      if (isHttp || isGrpc) {
        const base = isHttp ? `${apiUrl}/api/http` : `${apiUrl}/api/grpc`;
        const url = editingId ? `${base}/${editingId}` : base;
        const method = editingId ? "PUT" : "POST";

        const auth = {
          type: formData.auth_type || "none",
          bearer_token: formData.bearer_token || "",
          username: formData.auth_username || "",
          password: formData.auth_password || "",
        };

        const payload = isHttp
          ? {
              connection_name: formData.connection_name,
              base_url: formData.base_url,
              default_headers: formData.default_headers || [],
              auth,
              folder_id: formData.folder_id || "",
            }
          : {
              connection_name: formData.connection_name,
              address: formData.address,
              tls: !!formData.tls,
              insecure_tls: !!formData.insecure_tls,
              server_name: formData.server_name || "",
              auth,
              folder_id: formData.folder_id || "",
            };

        const res = await fetch(url, {
          method,
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(payload),
        });

        if (!res.ok) {
          const text = await res.text();
          throw new Error(text || res.statusText);
        }

        const data = await res.json();
        if (data.success) {
          setShowModal(false);
          setEditingId(null);
          connections.fetchConnections();
          setActiveId(data.id || (editingId ? editingId : activeId));
        } else {
          alert(data.error);
        }
        return;
      }

      const unsupportedNoSql = [
        "mongodb",
        "cassandra",
        "elasticsearch",
        "opensearch",
      ].includes(formData.db_type);
      if (unsupportedNoSql) {
        alert(
          `${formData.db_type} connections are not implemented in the engine yet.`,
        );
        return;
      }

      const isFileDb = formData.db_type === "sqlite";
      const portInt = isFileDb ? 0 : parseInt(formData.port);
      if (!isFileDb && isNaN(portInt)) {
        alert("Please enter a valid port number");
        return;
      }

      const isRedis = formData.db_type === "redis";
      const base = isRedis
        ? `${apiUrl}/api/redis`
        : `${apiUrl}/api/connections`;
      const url = editingId ? `${base}/${editingId}` : base;
      const method = editingId ? "PUT" : "POST";

      const res = await fetch(url, {
        method,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          ...formData,
          port: portInt,
        }),
      });

      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || res.statusText);
      }

      const data = await res.json();
      if (data.success) {
        setShowModal(false);
        setEditingId(null);
        connections.fetchConnections();
        setActiveId(data.id || (editingId ? editingId : activeId));
      } else {
        alert(data.error);
      }
    } catch (err) {
      alert("Failed to save connection: " + err.message);
    }
  };

  const handleTestConnection = async (e) => {
    e.preventDefault();
    try {
      const isHttp = formData.db_type === "http";
      const isGrpc = formData.db_type === "grpc";

      if (isHttp || isGrpc) {
        const auth = {
          type: formData.auth_type || "none",
          bearer_token: formData.bearer_token || "",
          username: formData.auth_username || "",
          password: formData.auth_password || "",
        };
        const payload = isHttp
          ? {
              connection_name: formData.connection_name,
              base_url: formData.base_url,
              default_headers: formData.default_headers || [],
              auth,
            }
          : {
              connection_name: formData.connection_name,
              address: formData.address,
              tls: !!formData.tls,
              insecure_tls: !!formData.insecure_tls,
              server_name: formData.server_name || "",
              auth,
            };

        const url = isHttp
          ? `${apiUrl}/api/http/test`
          : `${apiUrl}/api/grpc/test`;

        const res = await fetch(url, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(payload),
        });

        if (!res.ok) {
          const text = await res.text();
          throw new Error(text || res.statusText);
        }

        const data = await res.json();
        data.success
          ? alert("Success: " + data.message)
          : alert("Connection Failed:\n\n" + data.error);
        return;
      }

      const unsupportedNoSql = [
        "mongodb",
        "cassandra",
        "elasticsearch",
        "opensearch",
      ].includes(formData.db_type);
      if (unsupportedNoSql) {
        alert(
          `${formData.db_type} connections are not implemented in the engine yet.`,
        );
        return;
      }

      const isFileDb = formData.db_type === "sqlite";
      const portInt = isFileDb ? 0 : parseInt(formData.port);
      if (!isFileDb && isNaN(portInt)) {
        alert("Please enter a valid port number");
        return;
      }

      const isRedis = formData.db_type === "redis";
      const url = isRedis
        ? `${apiUrl}/api/redis/test`
        : `${apiUrl}/api/connections/test`;

      const res = await fetch(url, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          ...formData,
          port: portInt,
        }),
      });

      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || res.statusText);
      }

      const data = await res.json();
      data.success
        ? alert("Success: " + data.message)
        : alert("Connection Failed:\n\n" + data.error);
    } catch (err) {
      alert("Error reaching engine: " + err.message);
    }
  };

  const handleDeleteConnection = (id, e) => {
    e.stopPropagation();
    connections.handleDeleteConnection(id, activeId, setActiveId);
  };

  const handleDisconnect = (id) =>
    connections.handleDisconnect(id, activeId, setActiveId);
  const handleReconnect = (id) => {
    connections.handleReconnect(id);
    setActiveId(id);
  };

  const handleConnectionClick = async (id, forceExpand) => {
    const firstTabId = tabs.getFirstTabIdForConnection(id);
    if (firstTabId) {
      setActiveId(id);
      tabs.setActiveTabId(firstTabId);
      await connections.handleConnectionClick(id, forceExpand);
      return;
    }
    setSwitchPrompt({ targetId: id, forceExpand: !!forceExpand });
  };

  const buildTableSizeRows = (sizes) =>
    (sizes || []).map((s) => [
      s.schema || "",
      s.table || "",
      formatBytes(Number(s.size_bytes) || 0),
    ]);

  const openTableSizeCache = async (connId, connName) => {
    const res = await fetch(`${apiUrl}/api/connections/${connId}/table-sizes`);
    const data = await res.json();
    if (!data.success) {
      alert("Error: " + data.error);
      return;
    }
    const rows = buildTableSizeRows(data.sizes || []);
    tabs.addSpecialTab(
      `Table Sizes${connName ? `: ${connName}` : ""}`,
      "table_sizes",
      {
        results: {
          success: true,
          columns: ["Schema", "Table", "Size"],
          rows,
          meta: { updated_at: data.updated_at || "" },
        },
      },
      connId,
    );
  };

  const refreshTableSizeCache = async (connId) => {
    const res = await fetch(`${apiUrl}/api/connections/${connId}/table-sizes/refresh`, {
      method: "POST",
    });
    const data = await res.json();
    if (!data.success) {
      alert("Error: " + data.error);
      return;
    }
    const rows = buildTableSizeRows(data.sizes || []);
    tabs.updateActiveTab({
      results: {
        success: true,
        columns: ["Schema", "Table", "Size"],
        rows,
        meta: { updated_at: data.updated_at || "" },
      },
    });
  };

  useEffect(() => {
    if (activeTabConnId && activeTabConnId !== activeId) {
      setActiveId(activeTabConnId);
    }
  }, [activeTabConnId, activeId]);

  useEffect(() => {
    let cancelled = false;
    let timer = null;

    const shouldTrack =
      activeId &&
      activeConn &&
      !["redis", "http", "grpc"].includes(activeConn.db_type);

    if (!shouldTrack) {
      setTableSizeStatus(null);
      return () => {};
    }

    setTableSizeStatus({ status: "checking" });

    const poll = async () => {
      if (cancelled || !activeId) return;
      try {
        const res = await fetch(`${apiUrl}/api/connections/${activeId}/table-sizes/status`);
        let data = {};
        try {
          data = await res.json();
        } catch {
          data = {};
        }

        if (cancelled) return;

        if (!res.ok || !data.success) {
          const message =
            data.error ||
            (!res.ok ? `Status check failed (HTTP ${res.status})` : "Status check failed");
          setTableSizeStatus({ status: "checking", error: message });
          timer = setTimeout(poll, 3000);
          return;
        }

        const inferred =
          data.status || (typeof data.count === "number" && data.count > 0 ? "ready" : "idle");
        setTableSizeStatus({ ...data, status: inferred });
        if (!["ready", "unsupported", "error"].includes(inferred)) {
          timer = setTimeout(poll, 3000);
        }
      } catch (err) {
        if (!cancelled) {
          setTableSizeStatus({ status: "checking", error: "Status check failed" });
          timer = setTimeout(poll, 3000);
        }
      }
    };

    poll();

    return () => {
      cancelled = true;
      if (timer) clearTimeout(timer);
    };
  }, [activeId, activeConn?.db_type, apiUrl]);

  // ── Context menu action dispatcher ───────────────────────────────────────
  const handleMenuAction = async (action) => {
    ctxMenu.handleMenuAction(action, {
      onConnAction: async (act, conn) => {
        switch (act) {
          case "duplicate":
            setEditingId(null);
            setFormData({
              ...conn,
              connection_name: conn.connection_name + " (Copy)",
              password: "",
            });
            setShowModal(true);
            break;
          case "delete":
            handleDeleteConnection(conn.id, { stopPropagation: () => {} });
            break;
          case "refresh":
            connections.fetchConnections();
            break;
          case "connect":
            handleReconnect(conn.id);
            break;
          case "disconnect":
            handleDisconnect(conn.id);
            break;
          case "rename":
            handleEditConnection(conn, { stopPropagation: () => {} });
            break;
          case "view_table_sizes":
            await openTableSizeCache(conn.id, conn.connection_name);
            break;
        }
      },
      onTableAction: async (act, { tbl, schemaName, cId }) => {
        switch (act) {
          case "view_data": {
            const q = `SELECT * FROM ${schemaName}.${tbl.name} LIMIT 100;`;
            setActiveId(cId);
            tabs.updateActiveTab({ query: q, connectionId: cId });
            await tabs.executeQuery(q, cId);
            break;
          }
          case "view_dml": {
            const res = await fetch(
              `${apiUrl}/api/connections/${cId}/metadata/schemas/${schemaName}/tables/${tbl.name}/dml`,
            );
            const d = await res.json();
            if (d.success) {
              tabs.addSpecialTab(
                `DML: ${tbl.name}`,
                "dml",
                { results: { success: true, dml: d.dml } },
                cId,
              );
            } else {
              alert("Error fetching DML: " + d.error);
            }
            break;
          }
          case "view_indexes": {
            const res = await fetch(
              `${apiUrl}/api/connections/${cId}/metadata/schemas/${schemaName}/tables/${tbl.name}/indexes`,
            );
            const d = await res.json();
            if (!d.success) {
              alert("Error: " + d.error);
              break;
            }
            if (!d.indexes?.length) {
              alert("No indexes found.");
              break;
            }
            tabs.addSpecialTab(
              `Indexes: ${tbl.name}`,
              "indexes",
              {
                results: {
                  success: true,
                  columns: ["Name", "Columns", "Unique", "Primary"],
                  rows: d.indexes.map((i) => [
                    i.name,
                    i.columns.join(", "),
                    i.unique ? "Yes" : "No",
                    i.primary ? "Yes" : "No",
                  ]),
                },
              },
              cId,
            );
            break;
          }
          case "export":
            exp.handleExportClick("table", tbl.name, tbl.name);
            break;
        }
      },
      onTextAction: async (act, data) => {
        const writeText = async (text) => {
          if (navigator.clipboard?.writeText)
            return navigator.clipboard.writeText(String(text ?? ""));
          if (window.darube?.clipboard?.writeText) {
            try {
              window.darube.clipboard.writeText(String(text ?? ""));
              return;
            } catch {
              /* ignore */
            }
          }
          const ta = document.createElement("textarea");
          ta.value = String(text ?? "");
          ta.style.position = "fixed";
          ta.style.left = "-9999px";
          document.body.appendChild(ta);
          ta.select();
          document.execCommand("copy");
          document.body.removeChild(ta);
        };
        const readText = async () => {
          if (navigator.clipboard?.readText)
            return navigator.clipboard.readText();
          if (window.darube?.clipboard?.readText) {
            try {
              return window.darube.clipboard.readText();
            } catch {
              /* ignore */
            }
          }
          return "";
        };

        // Monaco editor path
        if (data?.kind === "monaco" && data?.editor) {
          const editor = data.editor;
          const model = editor.getModel?.();
          if (!model) return;

          const editorRole = data?.editorRole || null;
          const isRedis = editorRole === "redis";
          const connectionType = isRedis ? "redis" : undefined;
          const targetCId = tabs.activeTab.connectionId || activeId;

          const selections =
            editor.getSelections?.() ||
            (editor.getSelection ? [editor.getSelection()] : []);
          const isEmptySel = (s) => {
            if (!s) return true;
            if (typeof s.isEmpty === "function") return s.isEmpty();
            const sln = s.startLineNumber ?? s.selectionStartLineNumber;
            const sc = s.startColumn ?? s.selectionStartColumn;
            const eln =
              s.endLineNumber ??
              s.positionLineNumber ??
              s.selectionEndLineNumber;
            const ec = s.endColumn ?? s.positionColumn ?? s.selectionEndColumn;
            if (sln == null || sc == null || eln == null || ec == null)
              return true;
            return sln === eln && sc === ec;
          };
          const nonEmpty = (selections || []).filter((s) => !isEmptySel(s));
          const hasSelectionNow = nonEmpty.length > 0;
          const getSelectionText = () => {
            if (!hasSelectionNow) return "";
            const parts = nonEmpty
              .map((s) => {
                try {
                  return model.getValueInRange(s);
                } catch {
                  return "";
                }
              })
              .filter(Boolean);
            return parts.join("\n");
          };

          if (act === "run_query") {
            if (!targetCId) return;
            const selected = hasSelectionNow ? getSelectionText() : "";
            const query = (
              hasSelectionNow ? selected : model.getValue()
            ).trim();
            if (!query) return;
            await tabs.executeQuery(query, targetCId, connectionType);
            return;
          }

          if (act === "select_all") {
            editor.focus?.();
            try {
              editor.setSelection?.(model.getFullModelRange());
            } catch {
              /* ignore */
            }
            return;
          }
          if (act === "copy") {
            if (!hasSelectionNow) return;
            await writeText(getSelectionText());
            return;
          }
          if (act === "cut") {
            if (data?.readOnly || !hasSelectionNow) return;
            await writeText(getSelectionText());
            const edits = nonEmpty.map((r) => ({ range: r, text: "" }));
            try {
              editor.executeEdits?.("darube", edits);
            } catch {
              /* ignore */
            }
            return;
          }
          if (act === "paste") {
            if (data?.readOnly) return;
            const clip = await readText();
            const baseSelections =
              selections && selections.length
                ? selections
                : editor.getSelection
                  ? [editor.getSelection()]
                  : [];
            const edits = (baseSelections || [])
              .filter(Boolean)
              .map((r) => ({ range: r, text: String(clip ?? "") }));
            try {
              editor.executeEdits?.("darube", edits);
            } catch {
              /* ignore */
            }
            return;
          }
          return;
        }

        // DOM textarea/input path (legacy + non-Monaco inputs)
        const target = data?.el;
        if (!target) return;
        const hasSelection = !!data?.hasSelection;
        const readOnly = !!data?.readOnly;

        if (act === "run_query") {
          const role = target.dataset?.darubeEditorRole || null;
          const isRedis = role === "redis";
          const connectionType = isRedis ? "redis" : undefined;
          const targetCId = tabs.activeTab.connectionId || activeId;
          if (!targetCId) return;
          const box = target.dataset?.boxSelection || null;
          const parts = box
            ? String(box)
                .split(",")
                .map((n) => parseInt(n, 10))
            : null;
          const boxSel =
            parts && parts.length === 4 && parts.every((n) => !Number.isNaN(n))
              ? {
                  start: { row: parts[0], col: parts[1] },
                  end: { row: parts[2], col: parts[3] },
                }
              : null;
          const selected = hasSelection
            ? boxSel
              ? getBoxSelectionText(
                  target.value || "",
                  boxSel.start,
                  boxSel.end,
                )
              : (target.value || "").slice(
                  target.selectionStart ?? 0,
                  target.selectionEnd ?? 0,
                )
            : "";
          const query = (hasSelection ? selected : target.value || "").trim();
          if (!query) return;
          await tabs.executeQuery(query, targetCId, connectionType);
          return;
        }

        const box = target.dataset?.boxSelection || null;
        const parseBox = () => {
          if (!box) return null;
          const parts = String(box)
            .split(",")
            .map((n) => parseInt(n, 10));
          if (parts.length !== 4 || parts.some((n) => Number.isNaN(n)))
            return null;
          return {
            start: { row: parts[0], col: parts[1] },
            end: { row: parts[2], col: parts[3] },
          };
        };
        const boxSel = parseBox();
        const getSelection = () => {
          if (boxSel)
            return getBoxSelectionText(
              target.value || "",
              boxSel.start,
              boxSel.end,
            );
          const start = target.selectionStart ?? 0;
          const end = target.selectionEnd ?? 0;
          if (end <= start) return "";
          return (target.value || "").slice(start, end);
        };
        const dispatchInput = () =>
          target.dispatchEvent(new Event("input", { bubbles: true }));

        if (act === "select_all") {
          target.focus();
          target.setSelectionRange?.(0, (target.value || "").length);
          return;
        }
        if (act === "copy") {
          if (!hasSelection) return;
          await writeText(getSelection());
          return;
        }
        if (act === "cut") {
          if (readOnly || !hasSelection) return;
          await writeText(getSelection());
          const v = target.value || "";
          if (boxSel) {
            target.value = applyBoxCut(v, boxSel.start, boxSel.end);
            delete target.dataset.boxSelection;
            dispatchInput();
            return;
          } else {
            const start = target.selectionStart ?? 0;
            const end = target.selectionEnd ?? 0;
            target.value = v.slice(0, start) + v.slice(end);
            target.setSelectionRange?.(start, start);
            dispatchInput();
            return;
          }
        }
        if (act === "paste") {
          if (readOnly) return;
          const clip = await readText();
          const start = target.selectionStart ?? 0;
          const end = target.selectionEnd ?? 0;
          const v = target.value || "";
          target.value = v.slice(0, start) + clip + v.slice(end);
          const newPos = start + String(clip).length;
          target.setSelectionRange?.(newPos, newPos);
          dispatchInput();
          return;
        }
      },
      onResultsAction: async (act, { rowIndex, colIndex, selectedRows }) => {
        const writeText = async (text) => {
          if (navigator.clipboard?.writeText)
            return navigator.clipboard.writeText(String(text ?? ""));
          if (window.darube?.clipboard?.writeText) {
            try {
              window.darube.clipboard.writeText(String(text ?? ""));
              return;
            } catch {
              /* ignore */
            }
          }
          const ta = document.createElement("textarea");
          ta.value = String(text ?? "");
          ta.style.position = "fixed";
          ta.style.left = "-9999px";
          document.body.appendChild(ta);
          ta.select();
          document.execCommand("copy");
          document.body.removeChild(ta);
        };

        const cols = tabs.activeTab.results?.columns || [];
        const working = grid.computeWorkingData?.() || [];
        const row = (rowIndex != null ? working[rowIndex] : null) || null;

        if (act === "copy_cell") {
          if (rowIndex == null || colIndex == null) return;
          await writeText(String(row?.[colIndex] ?? ""));
          return;
        }
        if (act === "copy_row_tsv") {
          if (!row || !cols.length) return;
          const vals = cols.map((_, i) => String(row?.[i] ?? ""));
          await writeText(vals.join("\t"));
          return;
        }
        if (act === "copy_row_json") {
          if (!row || !cols.length) return;
          const obj = {};
          cols.forEach((c, i) => {
            obj[c] = row?.[i];
          });
          await writeText(JSON.stringify(obj, null, 2));
          return;
        }
        if (act === "copy_selected_tsv") {
          const sel = selectedRows?.length
            ? selectedRows
            : tabs.activeTab.selectedRows || [];
          if (!sel.length || !cols.length) return;
          const lines = sel
            .map((i) => working[i])
            .filter(Boolean)
            .map((r) => cols.map((_, ci) => String(r?.[ci] ?? "")).join("\t"));
          await writeText(lines.join("\n"));
          return;
        }
        if (act === "export_selected") {
          const sel = selectedRows?.length
            ? selectedRows
            : tabs.activeTab.selectedRows || [];
          if (!sel.length) return;
          const selectedData = sel.map((i) => working[i]).filter(Boolean);
          exp.setExportConfig({
            targetType: "data",
            targetName: `Selected Rows (${sel.length})`,
            queryTarget: tabs.activeTab.query,
            format: "csv",
            headers: true,
            path: "",
            filename: "selection_export",
            dataPayload: selectedData,
            columnsPayload: cols,
          });
        }
      },
    });
  };

  const handleResultsContextMenu = (e, targetIndex, cellInfo) => {
    let selected = tabs.activeTab.selectedRows || [];
    if (!selected.includes(targetIndex)) {
      selected = [targetIndex];
      tabs.updateActiveTab({
        selectedRows: selected,
        lastSelectedIndex: targetIndex,
      });
    }
    ctxMenu.handleResultsContextMenu(e, {
      rowIndex: targetIndex,
      colIndex: cellInfo?.colIndex ?? null,
      colName: cellInfo?.colName ?? null,
      selectedRows: selected,
    });
  };

  // ── Row selection (lives here because it touches tab state) ──────────────
  const handleRowClick = (e, targetIndex) => {
    let sel = [...(tabs.activeTab.selectedRows || [])];
    if (e.shiftKey && tabs.activeTab.lastSelectedIndex !== null) {
      const [lo, hi] = [
        Math.min(tabs.activeTab.lastSelectedIndex, targetIndex),
        Math.max(tabs.activeTab.lastSelectedIndex, targetIndex),
      ];
      if (!e.metaKey && !e.ctrlKey) sel = [];
      for (let i = lo; i <= hi; i++) {
        if (!sel.includes(i)) sel.push(i);
      }
    } else if (e.metaKey || e.ctrlKey) {
      sel = sel.includes(targetIndex)
        ? sel.filter((i) => i !== targetIndex)
        : [...sel, targetIndex];
    } else {
      sel = [targetIndex];
    }
    tabs.updateActiveTab({ selectedRows: sel, lastSelectedIndex: targetIndex });
  };

  // ── Connection info for status bar ───────────────────────────────────────

  // ── Render ────────────────────────────────────────────────────────────────
  return (
    <div className="app-container">
      <div className="app-topbar">
        <img className="app-topbar-icon" src={logoApp} alt="Darube" />
        <div className="app-topbar-title">Darube</div>
      </div>

      <div className="app-body">
        <Sidebar
          {...connections}
          tableSizes={connections.tableSizes}
          activeId={activeId}
          sidebarWidth={sidebarWidth}
          setSidebarWidth={setSidebarWidth}
          sidebarCollapsed={sidebarCollapsed}
          setSidebarCollapsed={setSidebarCollapsed}
          handleConnectionClick={handleConnectionClick}
          handleEditConnection={handleEditConnection}
          handleDeleteConnection={handleDeleteConnection}
          handleDisconnect={handleDisconnect}
          handleReconnect={handleReconnect}
          onNewConnection={openNewConnection}
          onNewScript={() => {
            const n = tabs.tabs.filter((t) => t.type === "script").length + 1;
            tabs.addSpecialTab(
              `Script ${n}`,
              "script",
              { query: "", results: null },
              null,
            );
          }}
          onShowHelp={() => setShowHelpModal(true)}
          onShowSettings={() => setShowSettingsModal(true)}
          handleConnectionContextMenu={ctxMenu.handleConnectionContextMenu}
          handleTableContextMenu={ctxMenu.handleTableContextMenu}
          toggleTree={connections.toggleTree}
        />

        <div className="main-area">
          <QueryTabs
            {...tabs}
            activeId={activeId}
            loading={loading}
            activeConnType={activeConn?.db_type}
          />

          {tabs.activeTab.type === "script" ? (
            <ScriptPane
              apiUrl={apiUrl}
              activeTab={tabs.activeTab}
              updateActiveTab={tabs.updateActiveTab}
              loading={loading}
              setLoading={setLoading}
              connections={connections.connections}
              onEditorContextMenu={ctxMenu.handleTextContextMenu}
            />
          ) : tabs.activeTab.type === "query" ? (
            activeConn?.db_type === "http" ? (
              <HttpRequestPane
                apiUrl={apiUrl}
                connectionId={tabs.activeTab.connectionId || activeId}
                activeTab={tabs.activeTab}
                updateActiveTab={tabs.updateActiveTab}
                loading={loading}
                setLoading={setLoading}
              />
            ) : activeConn?.db_type === "grpc" ? (
              <GrpcRequestPane
                apiUrl={apiUrl}
                connectionId={tabs.activeTab.connectionId || activeId}
                activeTab={tabs.activeTab}
                updateActiveTab={tabs.updateActiveTab}
                loading={loading}
                setLoading={setLoading}
              />
            ) : (
              <Split
                key={`${layoutDirection}-${activeConn?.db_type || "none"}`}
                className={`split-container ${layoutDirection}`}
                direction={layoutDirection}
                sizes={[40, 60]}
                minSize={100}
                gutterSize={8}
              >
              {/* Query editor pane */}
              <div className="pane query-section editor-pane">
                {activeConn?.db_type === "redis" ? (
                  <RedisAutocomplete
                    value={tabs.activeTab.query}
                    onChange={(code) => tabs.updateActiveTab({ query: code })}
                    onKeyDown={(e) =>
                      tabs.handleKeyDown(
                        e,
                        tabs.activeTab.connectionId || activeId,
                        "redis",
                      )
                    }
                    onContextMenu={ctxMenu.handleTextContextMenu}
                    disabled={!activeId}
                  />
                ) : (
                  <SqlAutocomplete
                    value={tabs.activeTab.query}
                    onChange={(code) => tabs.updateActiveTab({ query: code })}
                    onKeyDown={(e) =>
                      tabs.handleKeyDown(
                        e,
                        tabs.activeTab.connectionId || activeId,
                        activeConn?.db_type,
                      )
                    }
                    onContextMenu={ctxMenu.handleTextContextMenu}
                    disabled={!activeId}
                    placeholder={
                      activeId
                        ? "Type SQL query here... (Cmd/Ctrl + Enter to run)"
                        : "Select or add a connection to start"
                    }
                    apiUrl={apiUrl}
                    connectionId={tabs.activeTab.connectionId || activeId}
                  />
                )}

                {/* Status information bar (placed above the gutter) */}
                <div className="editor-status-bar">
                  <div className="status-left">
                    <span>
                      {activeConn
                        ? `${activeConn.connection_name} (${activeConn.db_type})`
                        : "No connection selected"}
                    </span>
                    {(() => {
                      const meta = [];
                      if (typeof tableSizeStatus?.count === "number") {
                        meta.push(`${tableSizeStatus.count.toLocaleString()} tables`);
                      }
                      const updatedLabel = formatStatusTime(tableSizeStatus?.updated_at);
                      if (updatedLabel) {
                        meta.push(`updated ${updatedLabel}`);
                      }
                      if (tableSizeStatus?.error) {
                        meta.push(tableSizeStatus.error);
                      }
                      const metaText = meta.length ? ` • ${meta.join(" • ")}` : "";

                      if (tableSizeStatus?.status === "running") {
                        return (
                          <span className="status-pill warning">
                            Table size cache: estimating{metaText}
                            <span className="status-progress warning" />
                          </span>
                        );
                      }
                      if (tableSizeStatus?.status === "ready") {
                        return <span className="status-pill">Table size cache: ready{metaText}</span>;
                      }
                      if (tableSizeStatus?.status === "unsupported") {
                        return (
                          <span className="status-pill muted">Table size cache: unsupported</span>
                        );
                      }
                      if (tableSizeStatus?.status === "error") {
                        return <span className="status-pill danger">Table size cache: failed</span>;
                      }
                      if (tableSizeStatus?.status === "checking") {
                        return (
                          <span
                            className={`status-pill ${tableSizeStatus?.error ? "danger" : "muted"}`}
                          >
                            Table size cache: checking{metaText}
                            <span className={`status-progress ${tableSizeStatus?.error ? "danger" : "info"}`} />
                          </span>
                        );
                      }
                      if (tableSizeStatus?.status === "idle") {
                        return (
                          <span className="status-pill muted">Table size cache: idle</span>
                        );
                      }
                      return null;
                    })()}
                  </div>
                  <div>
                    {tabs.activeTab.results?.durationMs !== undefined
                      ? `${tabs.activeTab.results.durationMs.toFixed(2)} ms`
                      : ""}
                  </div>
                </div>
              </div>

              {/* Results pane */}
              {activeConn?.db_type === "redis" ? (
                <RedisPane
                  activeTab={tabs.activeTab}
                  loading={loading}
                  connectionId={activeId}
                  onQuery={tabs.executeQuery}
                  onExport={(redisResult, command) =>
                    exp.handleExportRedisResult(redisResult, command)
                  }
                />
              ) : (
                <ResultsPane
                  activeTab={tabs.activeTab}
                  layoutDirection={layoutDirection}
                  loading={loading}
                  editingCell={grid.editingCell}
                  setEditingCell={grid.setEditingCell}
                  updateActiveTab={tabs.updateActiveTab}
                  undoMutation={grid.undoMutation}
                  redoMutation={grid.redoMutation}
                  cancelMutations={grid.cancelMutations}
                  saveMutations={grid.saveMutations}
                  handleCellDoubleClick={grid.handleCellDoubleClick}
                  handleCellBlur={grid.handleCellBlur}
                  handleRowAction={grid.handleRowAction}
                  handleRowClick={handleRowClick}
                  handleRowContextMenu={handleResultsContextMenu}
                  handleExportClick={exp.handleExportClick}
                  computeWorkingData={grid.computeWorkingData}
                  onRefreshTableSizes={refreshTableSizeCache}
                />
              )}
              </Split>
            )
          ) : activeConn?.db_type === "redis" ? (
            <RedisPane
              activeTab={tabs.activeTab}
              loading={loading}
              connectionId={activeId}
              onQuery={tabs.executeQuery}
              onExport={(redisResult, command) =>
                exp.handleExportRedisResult(redisResult, command)
              }
            />
          ) : (
            <ResultsPane
              activeTab={tabs.activeTab}
              layoutDirection={layoutDirection}
              loading={loading}
              editingCell={grid.editingCell}
              setEditingCell={grid.setEditingCell}
              updateActiveTab={tabs.updateActiveTab}
              undoMutation={grid.undoMutation}
              redoMutation={grid.redoMutation}
              cancelMutations={grid.cancelMutations}
              saveMutations={grid.saveMutations}
              handleCellDoubleClick={grid.handleCellDoubleClick}
              handleCellBlur={grid.handleCellBlur}
              handleRowAction={grid.handleRowAction}
              handleRowClick={handleRowClick}
              handleRowContextMenu={handleResultsContextMenu}
              handleExportClick={exp.handleExportClick}
              computeWorkingData={grid.computeWorkingData}
              onRefreshTableSizes={refreshTableSizeCache}
            />
          )}
        </div>
      </div>

      {/* Modals */}
      <ConnectionModal
        show={showModal}
        editingId={editingId}
        formData={formData}
        setFormData={setFormData}
        folders={connections.folders}
        apiUrl={apiUrl}
        onSubmit={handleConnectNew}
        onTest={handleTestConnection}
        onClose={() => setShowModal(false)}
        onFileUpload={(e, field) => exp.handleFileUpload(e, field, setFormData)}
      />
      <ExportModal
        exportConfig={exp.exportConfig}
        setExportConfig={exp.setExportConfig}
        loading={loading}
        onSubmit={exp.handleExecuteExport}
        onSelectDirectory={exp.handleSelectDirectory}
      />
      <ConnectionSwitchModal
        show={!!switchPrompt}
        connectionName={
          connections.connections.find((c) => c.id === switchPrompt?.targetId)
            ?.connection_name || "Selected connection"
        }
        onCreateNewTab={async () => {
          const targetId = switchPrompt?.targetId;
          if (!targetId) return;
          setSwitchPrompt(null);
          setActiveId(targetId);
          tabs.addNewTabForConnection(targetId);
          await connections.handleConnectionClick(targetId, true);
        }}
        onRebindCurrentTab={async () => {
          const targetId = switchPrompt?.targetId;
          if (!targetId) return;
          setSwitchPrompt(null);
          setActiveId(targetId);
          tabs.rebindActiveTabConnection(targetId);
          await connections.handleConnectionClick(targetId, true);
        }}
        onCancel={() => setSwitchPrompt(null)}
      />
      <SettingsModal
        show={showSettingsModal}
        onClose={() => setShowSettingsModal(false)}
        apiUrl={apiUrl}
        layoutDirection={layoutDirection}
        onLayoutChange={(dir) => {
          setLayoutDirection(dir);
          setSettings((prev) => (prev ? { ...prev, layout_direction: dir } : prev));
        }}
        settings={settings}
        onSettingsChange={(next) => {
          setSettings((prev) => ({ ...(prev || {}), ...next }));
        }}
      />
      <HelpModal show={showHelpModal} onClose={() => setShowHelpModal(false)} />
      <ContextMenu
        contextMenu={ctxMenu.contextMenu}
        onAction={handleMenuAction}
      />
    </div>
  );
}

export default App;
