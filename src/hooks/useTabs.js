import { useState, useEffect, useRef } from "react";
import { getTargetTable } from "../utils/queryUtils";
import { formatBytes } from "../utils/formatBytes";

const EMPTY_TAB = (n, id, type = "query") => ({
  id,
  type,
  name: type === "query" ? `Query ${n}` : `Tab ${n}`,
  query: "",
  connectionId: null,
  lastExecutedQuery: "",
  results: null,
  plan: null,
  activeView: "results",
  currentPage: 1,
  rowsPerPage: 50,
  selectedRows: [],
  lastSelectedIndex: null,
  history: [],
  historyIndex: -1,
  targetTable: null,
});

export function useTabs(apiUrl, activeId, setLoading, settings) {
  const [tabs, setTabs] = useState([EMPTY_TAB(1, "tab-1")]);
  const [activeTabId, setActiveTabId] = useState("tab-1");
  const [editingTabId, setEditingTabId] = useState(null);
  const activeRequestRef = useRef(null);
  const selectionByTabRef = useRef({});

  const activeTab = tabs.find((t) => t.id === activeTabId) || tabs[0];

  // If the initial tab is unbound and we have an active connection, bind it.
  useEffect(() => {
    if (!activeId) return;
    if (!activeTab?.connectionId && activeTab?.id) {
      setTabs((prev) =>
        prev.map((t) =>
          t.id === activeTab.id ? { ...t, connectionId: activeId } : t,
        ),
      );
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeId]);

  // Load persisted workspace on mount
  useEffect(() => {
    (async () => {
      try {
        const res = await fetch(`${apiUrl}/api/workspace`);
        if (!res.ok) return;
        const data = await res.json();
        if (data.tabs?.length > 0) {
          const hydrated = data.tabs.map((t) => ({
            ...EMPTY_TAB(1, t.id, t.type || "query"),
            id: t.id,
            name: t.name,
            query: t.query,
            connectionId: t.connection_id || null,
          }));
          setTabs(hydrated);
          setActiveTabId(hydrated[0].id);
        }
      } catch {
        /* ignore */
      }
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Debounced auto-save (5 s idle)
  useEffect(() => {
    if (tabs.length === 1 && tabs[0].id === "tab-1" && !tabs[0].query) return;
    const timer = setTimeout(() => {
      const stripped = tabs.map((t) => ({
        id: t.id,
        name: t.name,
        query: t.query,
        type: t.type,
        connection_id: t.connectionId || "",
      }));
      fetch(`${apiUrl}/api/workspace`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ tabs: stripped }),
      }).catch((e) => console.error("Failed to autosave workspace", e));
    }, 5000);
    return () => clearTimeout(timer);
  }, [tabs, apiUrl]);

  const updateActiveTab = (updates) => {
    setTabs((prev) =>
      prev.map((t) => (t.id === activeTabId ? { ...t, ...updates } : t)),
    );
  };

  const addNewTab = () => {
    const newId = "tab-" + Date.now();
    setTabs((prev) => [
      ...prev,
      { ...EMPTY_TAB(prev.length + 1, newId), connectionId: activeId || null },
    ]);
    setActiveTabId(newId);
  };

  const addSpecialTab = (name, type, updates = {}, connectionId) => {
    const newId = "tab-" + Date.now();
    const boundConnId =
      connectionId === undefined ? activeId || null : connectionId;
    const newTab = {
      ...EMPTY_TAB(tabs.length + 1, newId, type),
      name,
      connectionId: boundConnId,
      ...updates,
    };
    setTabs((prev) => [...prev, newTab]);
    setActiveTabId(newId);
  };

  const getLargeResultWarnBytes = () => {
    const raw = settings?.large_result_warn_mb;
    if (raw === undefined || raw === null || raw === "") {
      return 50 * 1024 * 1024;
    }
    const val = Number(raw);
    if (!Number.isFinite(val) || val <= 0) return 0;
    return val * 1024 * 1024;
  };

  const shouldEstimateResult = (query, connectionType) => {
    if (!query) return false;
    if (connectionType === "redis") return false;
    const trimmed = query.trim();
    if (!trimmed) return false;
    const upper = trimmed.toUpperCase();
    if (upper.startsWith("EXPLAIN")) return false;
    return upper.startsWith("SELECT") || upper.startsWith("WITH");
  };

  const estimateQueryResult = async (query, connectionId, connectionType, signal) => {
    if (!shouldEstimateResult(query, connectionType)) return null;
    try {
      const res = await fetch(`${apiUrl}/api/connections/${connectionId}/estimate`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ query }),
        signal,
      });
      if (!res.ok) return null;
      const data = await res.json();
      if (!data?.success || !data?.available || !data?.estimated_bytes) return null;
      return data;
    } catch (err) {
      if (err?.name === "AbortError") throw err;
      return null;
    }
  };

  const cancelActiveRequest = () => {
    if (activeRequestRef.current) {
      activeRequestRef.current.abort();
    }
  };

  const getTabsForConnection = (connectionId) => {
    if (!connectionId) return [];
    return tabs.filter((t) => t.connectionId === connectionId);
  };

  const getFirstTabIdForConnection = (connectionId) => {
    const t = getTabsForConnection(connectionId)[0];
    return t ? t.id : null;
  };

  const addNewTabForConnection = (connectionId) => {
    const newId = "tab-" + Date.now();
    setTabs((prev) => [
      ...prev,
      { ...EMPTY_TAB(prev.length + 1, newId), connectionId },
    ]);
    setActiveTabId(newId);
    return newId;
  };

  const rebindActiveTabConnection = (connectionId) => {
    setTabs((prev) =>
      prev.map((t) => {
        if (t.id !== activeTabId) return t;
        return {
          ...t,
          connectionId,
          // Prevent confusing carryover when the underlying connection changes.
          results: null,
          plan: null,
          lastExecutedQuery: "",
          activeView: "results",
          currentPage: 1,
          selectedRows: [],
          lastSelectedIndex: null,
          history: [],
          historyIndex: -1,
          targetTable: null,
        };
      }),
    );
  };

  const closeTab = (e, idToClose) => {
    e.stopPropagation();
    if (tabs.length === 1) return;
    const newTabs = tabs.filter((t) => t.id !== idToClose);
    setTabs(newTabs);
    if (activeTabId === idToClose) setActiveTabId(newTabs[0].id);
    delete selectionByTabRef.current[idToClose];
  };

  const updateActiveTabSelection = (selectionStart, selectionEnd, tabId = activeTabId) => {
    if (!tabId) return;
    selectionByTabRef.current[tabId] = {
      selectionStart: Number(selectionStart) || 0,
      selectionEnd: Number(selectionEnd) || 0,
    };
  };

  const shouldRequireSelectionForSql = (query) => {
    // When users write multiple independent queries on separate lines (without semicolons),
    // running the whole buffer is confusing. Require an explicit selection in that case.
    const raw = String(query || "");
    const lines = raw
      .split(/\r?\n/)
      .map((l) => l.trim())
      .filter(Boolean);
    if (lines.length <= 1) return false;
    // If the user already uses semicolons, allow multi-statement runs.
    if (raw.includes(";")) return false;
    // Heuristic: require selection only when we likely have multiple independent statements.
    // Most common "oops" case is multiple `SELECT` blocks stacked without semicolons.
    const selectStarts = lines.filter((l) => /^SELECT\b/i.test(l)).length;
    if (selectStarts >= 2) return true;
    const dmlStarts = lines.filter((l) => /^(INSERT|UPDATE|DELETE|EXPLAIN)\b/i.test(l)).length;
    return dmlStarts >= 2;
  };

  const executeQuery = async (queryToRun, connectionId, connectionType) => {
    const tabText = activeTab.query || "";
    const sel = selectionByTabRef.current[activeTabId] || null;
    const hasSelection =
      sel && Number.isFinite(sel.selectionStart) && Number.isFinite(sel.selectionEnd) && sel.selectionStart !== sel.selectionEnd;

    let source = queryToRun;
    if (!source && hasSelection) {
      const s = Math.max(0, Math.min(tabText.length, sel.selectionStart));
      const e = Math.max(0, Math.min(tabText.length, sel.selectionEnd));
      source = tabText.substring(Math.min(s, e), Math.max(s, e));
    }
    if (!source && connectionType !== "redis" && shouldRequireSelectionForSql(tabText)) {
      alert('Highlight a line (or selection) to run, or add semicolons to run multiple SQL statements.');
      return;
    }

    const finalQuery = (source || tabText).trim();
    const targetCId = connectionId || activeTab.connectionId || activeId;
    if (!targetCId || !finalQuery) return;

    // Global maximum lines for query/command (SQL + Redis).
    const maxLines = settings?.max_lines_query || 0;
    if (maxLines > 0) {
      const lineCount = finalQuery.split(/\r?\n/).length;
      if (lineCount > maxLines) {
        alert(
          `This tab is limited to ${maxLines} lines. Current query has ${lineCount} lines.`,
        );
        return;
      }
    }

    setLoading(true);
    const t0 = performance.now();
    const controller = new AbortController();
    activeRequestRef.current = controller;

    try {
      const warnBytes = getLargeResultWarnBytes();
      if (warnBytes > 0) {
        const estimate = await estimateQueryResult(finalQuery, targetCId, connectionType, controller.signal);
        if (estimate?.estimated_bytes >= warnBytes) {
          const confirmFn = typeof window !== "undefined" && window.confirm ? window.confirm : () => true;
          const ok = confirmFn(
            `Large result expected (~${formatBytes(estimate.estimated_bytes)}). Run anyway?`,
          );
          if (!ok) return;
        }
      }
      const isRedis = connectionType === "redis";
      const url = isRedis
        ? `${apiUrl}/api/redis/${targetCId}/query`
        : `${apiUrl}/api/connections/${targetCId}/query`;

      const bodyKey = isRedis ? "command" : "query";

      if (isRedis && finalQuery.includes("\n")) {
        const lines = finalQuery
          .split("\n")
          .map((l) => l.trim())
          .filter(Boolean);
        const results = [];
        for (const line of lines) {
          try {
            const res = await fetch(url, {
              method: "POST",
              headers: { "Content-Type": "application/json" },
              body: JSON.stringify({ [bodyKey]: line }),
              signal: controller.signal,
            });
            const data = await res.json();
            results.push({ command: line, ...data });
          } catch (err) {
            if (err?.name === "AbortError") throw err;
            results.push({ command: line, success: false, error: err.message });
          }
        }

        updateActiveTab({
          lastExecutedQuery: finalQuery,
          results: {
            success: true,
            multi: true,
            data: results,
            durationMs: performance.now() - t0,
          },
          activeView: "results",
          currentPage: 1,
          selectedRows: [],
          lastSelectedIndex: null,
          history: [],
          historyIndex: -1,
          targetTable: null,
        });
      } else {
        const res = await fetch(url, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ [bodyKey]: finalQuery }),
          signal: controller.signal,
        });
        const data = await res.json();

        if (!isRedis && data.rows) {
          data.rows = data.rows.map((r) => {
            const copy = [...r];
            copy._ui_id = Math.random().toString(36).substr(2, 9);
            return copy;
          });
        }

        const durationMs = performance.now() - t0;
        const targetTable = isRedis ? null : getTargetTable(finalQuery);

        updateActiveTab({
          lastExecutedQuery: finalQuery,
          results: { ...data, durationMs },
          activeView: "results",
          currentPage: 1,
          selectedRows: [],
          lastSelectedIndex: null,
          history: [],
          historyIndex: -1,
          targetTable,
        });
      }
    } catch (err) {
      if (err?.name === "AbortError") {
        updateActiveTab({
          lastExecutedQuery: finalQuery,
          results: {
            success: false,
            error: "Query cancelled",
            durationMs: performance.now() - t0,
          },
          activeView: "results",
          currentPage: 1,
          selectedRows: [],
          lastSelectedIndex: null,
        });
        return;
      }
      updateActiveTab({
        lastExecutedQuery: finalQuery,
        results: {
          success: false,
          error: err.message,
          durationMs: performance.now() - t0,
        },
        activeView: "results",
        currentPage: 1,
        selectedRows: [],
        lastSelectedIndex: null,
      });
    } finally {
      setLoading(false);
      if (activeRequestRef.current === controller) activeRequestRef.current = null;
    }
  };

  const executeExplain = async (queryToRun) => {
    const finalQuery = queryToRun || activeTab.query;
    const targetCId = activeTab.connectionId || activeId;
    if (!targetCId || !finalQuery) return;
    setLoading(true);
    const t0 = performance.now();
    const controller = new AbortController();
    activeRequestRef.current = controller;
    try {
      const res = await fetch(
        `${apiUrl}/api/connections/${targetCId}/explain`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ query: finalQuery }),
          signal: controller.signal,
        },
      );
      const data = await res.json();
      const durationMs = performance.now() - t0;
      if (data.success) {
        updateActiveTab({
          plan: data.plan,
          activeView: "plan",
          results: activeTab.results
            ? { ...activeTab.results, durationMs }
            : { durationMs },
        });
      } else {
        updateActiveTab({
          results: { success: false, error: data.error, durationMs },
          activeView: "results",
        });
      }
    } catch (err) {
      if (err?.name === "AbortError") {
        updateActiveTab({
          results: {
            success: false,
            error: "Query cancelled",
            durationMs: performance.now() - t0,
          },
          activeView: "results",
        });
        return;
      }
      updateActiveTab({
        results: {
          success: false,
          error: err.message,
          durationMs: performance.now() - t0,
        },
        activeView: "results",
      });
    } finally {
      setLoading(false);
      if (activeRequestRef.current === controller) activeRequestRef.current = null;
    }
  };

  const handleKeyDown = (e, connectionId, connectionType) => {
    if ((e.metaKey || e.ctrlKey) && e.key === "Enter") {
      e.preventDefault();
      const ta = e.target;
      const { selectionStart: s, selectionEnd: en } = ta;
      updateActiveTabSelection(s, en);
      if (connectionType !== "redis" && shouldRequireSelectionForSql(activeTab.query || "") && s === en) {
        alert('Highlight a line (or selection) to run, or add semicolons to run multiple SQL statements.');
        return;
      }
      executeQuery(s !== en ? activeTab.query.substring(s, en) : null, connectionId, connectionType);
    }
  };

  return {
    tabs,
    setTabs,
    activeTabId,
    setActiveTabId,
    editingTabId,
    setEditingTabId,
    activeTab,
    updateActiveTab,
    addNewTab,
    addSpecialTab,
    closeTab,
    getTabsForConnection,
    getFirstTabIdForConnection,
    addNewTabForConnection,
    rebindActiveTabConnection,
    executeQuery,
    executeExplain,
    cancelActiveRequest,
    handleKeyDown,
    updateActiveTabSelection,
  };
}
