import iconPlay from '../assets/play.svg';
import { useEffect, useMemo, useState } from 'react';

export function QueryTabs({
  tabs, activeTabId, setActiveTabId, editingTabId, setEditingTabId, setTabs,
  addNewTab, closeTab, activeId, loading,
  executeQuery, executeExplain, cancelActiveRequest,
  activeConnType,
}) {
  const stopRequest = cancelActiveRequest || (() => {});
  const isRedis = activeConnType === 'redis';
  const isApi = activeConnType === 'http' || activeConnType === 'grpc';

  const [menu, setMenu] = useState(null); // { tabId, x, y }

  const tabIndexById = useMemo(() => {
    const m = new Map();
    tabs.forEach((t, i) => m.set(t.id, i));
    return m;
  }, [tabs]);

  useEffect(() => {
    if (!menu) return;
    const onMouseDown = () => setMenu(null);
    const onKeyDown = (e) => {
      if (e.key === 'Escape') setMenu(null);
    };
    window.addEventListener('mousedown', onMouseDown);
    window.addEventListener('keydown', onKeyDown);
    return () => {
      window.removeEventListener('mousedown', onMouseDown);
      window.removeEventListener('keydown', onKeyDown);
    };
  }, [menu]);

  const withNextActive = (nextTabs, preferredActiveId) => {
    setTabs(nextTabs);
    if (nextTabs.length === 0) return;
    const nextActive =
      nextTabs.find((t) => t.id === preferredActiveId)?.id || nextTabs[0].id;
    setActiveTabId(nextActive);
    if (editingTabId && !nextTabs.some((t) => t.id === editingTabId)) {
      setEditingTabId(null);
    }
  };

  const closeTabsToRight = (tabId) => {
    const idx = tabIndexById.get(tabId);
    if (idx === undefined) return;
    const nextTabs = tabs.slice(0, idx + 1);
    withNextActive(nextTabs, tabId);
  };

  const closeTabsToLeft = (tabId) => {
    const idx = tabIndexById.get(tabId);
    if (idx === undefined) return;
    const nextTabs = tabs.slice(idx);
    withNextActive(nextTabs, tabId);
  };

  const closeAllExcept = (tabId) => {
    const t = tabs.find((x) => x.id === tabId);
    if (!t) return;
    withNextActive([t], tabId);
  };

  const duplicateTab = (tabId) => {
    const idx = tabIndexById.get(tabId);
    const t = tabs.find((x) => x.id === tabId);
    if (idx === undefined || !t) return;
    const newId = `tab-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
    const copy = {
      ...t,
      id: newId,
      name: `${t.name} copy`,
      lastExecutedQuery: '',
      results: null,
      plan: null,
      activeView: 'results',
      currentPage: 1,
      selectedRows: [],
      lastSelectedIndex: null,
      history: [],
      historyIndex: -1,
      targetTable: null,
    };
    const nextTabs = [...tabs.slice(0, idx + 1), copy, ...tabs.slice(idx + 1)];
    withNextActive(nextTabs, newId);
  };

  return (
    <div className="tabs-bar">
      <div className="tabs-scroll">
        {tabs.map(t => (
          <div
            key={t.id}
            className={`tab ${activeTabId === t.id ? 'active' : ''}`}
            onClick={() => setActiveTabId(t.id)}
            onDoubleClick={() => setEditingTabId(t.id)}
            onContextMenu={(e) => {
              e.preventDefault();
              e.stopPropagation();
              const pad = 8;
              const mw = 240;
              const mh = 200;
              const x = Math.min(e.clientX, window.innerWidth - mw - pad);
              const y = Math.min(e.clientY, window.innerHeight - mh - pad);
              setMenu({ tabId: t.id, x: Math.max(pad, x), y: Math.max(pad, y) });
            }}
          >
            {editingTabId === t.id ? (
              <input
                autoFocus
                defaultValue={t.name}
                onClick={e => e.stopPropagation()}
                onBlur={(e) => {
                  const newName = e.target.value.trim() || t.name;
                  setTabs(prev => prev.map(tab => tab.id === t.id ? { ...tab, name: newName } : tab));
                  setEditingTabId(null);
                }}
                onKeyDown={e => {
                  if (e.key === 'Enter') e.target.blur();
                  if (e.key === 'Escape') setEditingTabId(null);
                }}
                style={{ background: 'transparent', border: 'none', color: 'inherit', outline: 'none', width: '80px', fontSize: 'inherit' }}
              />
            ) : t.name}
            {tabs.length > 1 && <span className="close-tab" onClick={(e) => closeTab(e, t.id)}>×</span>}
          </div>
        ))}

        <div className="tab add-tab" onClick={addNewTab}>+</div>
      </div>

      {activeId && !isApi && (
        <div className="tabs-actions">
          {!isRedis && (
            <div
              className="tab-action-btn secondary"
              onClick={() => executeExplain(null)}
              style={{ opacity: loading ? 0.6 : 1, pointerEvents: loading ? 'none' : 'auto' }}
            >
              Explain
            </div>
          )}
          {loading ? (
            <div
              className="tab-action-btn secondary"
              onClick={stopRequest}
            >
              Stop
            </div>
          ) : (
            <div
              className="tab-action-btn"
              onClick={() => executeQuery(null, null, isRedis ? 'redis' : undefined)}
              title="Cmd/Ctrl + Enter"
            >
              <img src={iconPlay} className="icon-sm icon-light" alt="Play" />
              Run Query
            </div>
          )}
        </div>
      )}

      {menu && (
        <div
          className="tab-context-menu"
          style={{ left: menu.x, top: menu.y }}
          onMouseDown={(e) => e.stopPropagation()}
        >
          <button type="button" className="tab-context-item" onClick={() => { closeTabsToRight(menu.tabId); setMenu(null); }}>
            Close Tab to the right
          </button>
          <button type="button" className="tab-context-item" onClick={() => { closeTabsToLeft(menu.tabId); setMenu(null); }}>
            Close Tab to the left
          </button>
          <button type="button" className="tab-context-item" onClick={() => { closeAllExcept(menu.tabId); setMenu(null); }}>
            Close all except this tab
          </button>
          <div className="tab-context-sep" />
          <button type="button" className="tab-context-item" onClick={() => { duplicateTab(menu.tabId); setMenu(null); }}>
            Duplicate Tab
          </button>
        </div>
      )}
    </div>
  );
}
