import iconPlay from '../assets/play.svg';

export function QueryTabs({
  tabs, activeTabId, setActiveTabId, editingTabId, setEditingTabId, setTabs,
  addNewTab, closeTab, activeId, loading,
  executeQuery, executeExplain,
  activeConnType,
}) {
  const isRedis = activeConnType === 'redis';
  return (
    <div className="tabs-bar">
      {tabs.map(t => (
        <div
          key={t.id}
          className={`tab ${activeTabId === t.id ? 'active' : ''}`}
          onClick={() => setActiveTabId(t.id)}
          onDoubleClick={() => setEditingTabId(t.id)}
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
      <div style={{ flexGrow: 1 }} />

      {activeId && (
        <div style={{ display: 'flex', gap: '8px', padding: '0 8px', alignItems: 'center' }}>
          {!isRedis && (
            <div
              className="tab-action-btn"
              onClick={() => executeExplain(null)}
              style={{ opacity: loading ? 0.6 : 1, pointerEvents: loading ? 'none' : 'auto', background: 'transparent', border: '1px solid var(--border)', marginRight: '5px' }}
            >
              Explain
            </div>
          )}
          <div
            className="tab-action-btn"
            onClick={() => executeQuery(null, null, isRedis ? 'redis' : undefined)}
            title="Cmd/Ctrl + Enter"
            style={{ opacity: loading ? 0.6 : 1, pointerEvents: loading ? 'none' : 'auto' }}
          >
            <img src={iconPlay} className="icon-sm icon-light" alt="Play" />
            {loading ? 'Running...' : 'Run Query'}
          </div>
        </div>
      )}
    </div>
  );
}
