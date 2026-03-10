import { useState, useEffect } from 'react';
import { getTargetTable } from '../utils/queryUtils';

const EMPTY_TAB = (n, id, type = 'query') => ({
  id,
  type,
  name: type === 'query' ? `Query ${n}` : `Tab ${n}`,
  query: '',
  connectionId: null,
  lastExecutedQuery: '',
  results: null,
  plan: null,
  activeView: 'results',
  currentPage: 1,
  rowsPerPage: 50,
  selectedRows: [],
  lastSelectedIndex: null,
  history: [],
  historyIndex: -1,
  targetTable: null,
});

export function useTabs(apiUrl, activeId, setLoading) {
  const [tabs, setTabs] = useState([EMPTY_TAB(1, 'tab-1')]);
  const [activeTabId, setActiveTabId] = useState('tab-1');
  const [editingTabId, setEditingTabId] = useState(null);

  const activeTab = tabs.find(t => t.id === activeTabId) || tabs[0];

  // If the initial tab is unbound and we have an active connection, bind it.
  useEffect(() => {
    if (!activeId) return;
    if (!activeTab?.connectionId && activeTab?.id) {
      setTabs(prev => prev.map(t => t.id === activeTab.id ? { ...t, connectionId: activeId } : t));
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
          const hydrated = data.tabs.map(t => ({
            ...EMPTY_TAB(1, t.id, t.type || 'query'),
            id: t.id,
            name: t.name,
            query: t.query,
            connectionId: t.connection_id || null,
          }));
          setTabs(hydrated);
          setActiveTabId(hydrated[0].id);
        }
      } catch { /* ignore */ }
    })();
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Debounced auto-save (5 s idle)
  useEffect(() => {
    if (tabs.length === 1 && tabs[0].id === 'tab-1' && !tabs[0].query) return;
    const timer = setTimeout(() => {
      const stripped = tabs.map(t => ({ id: t.id, name: t.name, query: t.query, type: t.type, connection_id: t.connectionId || '' }));
      fetch(`${apiUrl}/api/workspace`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ tabs: stripped }),
      }).catch(e => console.error('Failed to autosave workspace', e));
    }, 5000);
    return () => clearTimeout(timer);
  }, [tabs, apiUrl]);

  const updateActiveTab = (updates) => {
    setTabs(prev => prev.map(t => t.id === activeTabId ? { ...t, ...updates } : t));
  };

  const addNewTab = () => {
    const newId = 'tab-' + Date.now();
    setTabs(prev => [...prev, { ...EMPTY_TAB(prev.length + 1, newId), connectionId: activeId || null }]);
    setActiveTabId(newId);
  };

  const addSpecialTab = (name, type, updates = {}, connectionId) => {
    const newId = 'tab-' + Date.now();
    const boundConnId = connectionId === undefined ? (activeId || null) : connectionId;
    const newTab = { ...EMPTY_TAB(tabs.length + 1, newId, type), name, connectionId: boundConnId, ...updates };
    setTabs(prev => [...prev, newTab]);
    setActiveTabId(newId);
  };

  const getTabsForConnection = (connectionId) => {
    if (!connectionId) return [];
    return tabs.filter(t => t.connectionId === connectionId);
  };

  const getFirstTabIdForConnection = (connectionId) => {
    const t = getTabsForConnection(connectionId)[0];
    return t ? t.id : null;
  };

  const addNewTabForConnection = (connectionId) => {
    const newId = 'tab-' + Date.now();
    setTabs(prev => [...prev, { ...EMPTY_TAB(prev.length + 1, newId), connectionId }]);
    setActiveTabId(newId);
    return newId;
  };

  const rebindActiveTabConnection = (connectionId) => {
    setTabs(prev => prev.map(t => {
      if (t.id !== activeTabId) return t;
      return {
        ...t,
        connectionId,
        // Prevent confusing carryover when the underlying connection changes.
        results: null,
        plan: null,
        lastExecutedQuery: '',
        activeView: 'results',
        currentPage: 1,
        selectedRows: [],
        lastSelectedIndex: null,
        history: [],
        historyIndex: -1,
        targetTable: null,
      };
    }));
  };

  const closeTab = (e, idToClose) => {
    e.stopPropagation();
    if (tabs.length === 1) return;
    const newTabs = tabs.filter(t => t.id !== idToClose);
    setTabs(newTabs);
    if (activeTabId === idToClose) setActiveTabId(newTabs[0].id);
  };

  const executeQuery = async (queryToRun, connectionId, connectionType) => {
    const finalQuery = queryToRun || activeTab.query;
    const targetCId  = connectionId || activeTab.connectionId || activeId;
    if (!targetCId || !finalQuery) return;
    setLoading(true);
    const t0 = performance.now();
    try {
      const isRedis = connectionType === 'redis';
      const url = isRedis 
        ? `${apiUrl}/api/redis/${targetCId}/query`
        : `${apiUrl}/api/connections/${targetCId}/query`;
      
      const bodyKey = isRedis ? 'command' : 'query';
      
      const res = await fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ [bodyKey]: finalQuery }),
      });
      const data = await res.json();
      
      if (!isRedis && data.rows) {
        data.rows = data.rows.map(r => {
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
        activeView: 'results',
        currentPage: 1,
        selectedRows: [],
        lastSelectedIndex: null,
        history: [],
        historyIndex: -1,
        targetTable,
      });
    } catch (err) {
      updateActiveTab({
        lastExecutedQuery: finalQuery,
        results: { success: false, error: err.message, durationMs: performance.now() - t0 },
        activeView: 'results',
        currentPage: 1,
        selectedRows: [],
        lastSelectedIndex: null,
      });
    } finally {
      setLoading(false);
    }
  };

  const executeExplain = async (queryToRun) => {
    const finalQuery = queryToRun || activeTab.query;
    const targetCId = activeTab.connectionId || activeId;
    if (!targetCId || !finalQuery) return;
    setLoading(true);
    const t0 = performance.now();
    try {
      const res = await fetch(`${apiUrl}/api/connections/${targetCId}/explain`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ query: finalQuery }),
      });
      const data = await res.json();
      const durationMs = performance.now() - t0;
      if (data.success) {
        updateActiveTab({ plan: data.plan, activeView: 'plan', results: activeTab.results ? { ...activeTab.results, durationMs } : { durationMs } });
      } else {
        updateActiveTab({ results: { success: false, error: data.error, durationMs }, activeView: 'results' });
      }
    } catch (err) {
      updateActiveTab({ results: { success: false, error: err.message, durationMs: performance.now() - t0 }, activeView: 'results' });
    } finally {
      setLoading(false);
    }
  };

  const handleKeyDown = (e, connectionId, connectionType) => {
    if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
      e.preventDefault();
      const ta = e.target;
      const { selectionStart: s, selectionEnd: en } = ta;
      executeQuery(s !== en ? activeTab.query.substring(s, en) : activeTab.query, connectionId, connectionType);
    }
  };

  return {
    tabs, setTabs, activeTabId, setActiveTabId, editingTabId, setEditingTabId,
    activeTab,
    updateActiveTab, addNewTab, addSpecialTab, closeTab,
    getTabsForConnection, getFirstTabIdForConnection, addNewTabForConnection, rebindActiveTabConnection,
    executeQuery, executeExplain, handleKeyDown,
  };
}
