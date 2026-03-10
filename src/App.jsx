import { useState, useEffect } from 'react'
import Split from 'react-split'

import { SqlAutocomplete } from './components/SqlAutocomplete'
import { RedisAutocomplete } from './components/RedisAutocomplete'
import { Sidebar }          from './components/Sidebar'
import { QueryTabs }        from './components/QueryTabs'
import { ResultsPane }      from './components/ResultsPane'
import { RedisPane }        from './components/RedisPane'
import { ScriptPane }       from './components/ScriptPane'
import { ConnectionModal }  from './components/ConnectionModal'
import { ContextMenu, ExportModal, HelpModal, ConnectionSwitchModal } from './components/Modals'

import { useConnections }  from './hooks/useConnections'
import { useTabs }         from './hooks/useTabs'
import { useEditableGrid } from './hooks/useEditableGrid'
import { useContextMenu }  from './hooks/useContextMenu'
import { useExport }       from './hooks/useExport'

const params    = new URLSearchParams(window.location.search);
const enginePort = params.get('enginePort') || '3000';
const apiUrl    = `http://localhost:${enginePort}`;

const EMPTY_FORM = { 
  connection_name: '', db_type: 'postgres', host: '', port: 5432, dbname: '',
  file_path: '',
  user: '', password: '', enable_ssl: false, 
  ca_cert_path: '', client_cert_path: '', client_key_path: '', 
  folder_id: '' 
};

function App() {
  // ── Shared UI state ──────────────────────────────────────────────────────
  const [activeId, setActiveId]               = useState(null);
  const [loading, setLoading]                 = useState(false);
  const [showModal, setShowModal]             = useState(false);
  const [showHelpModal, setShowHelpModal]     = useState(false);
  const [switchPrompt, setSwitchPrompt]       = useState(null); // { targetId, forceExpand }
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [layoutDirection, setLayoutDirection] = useState('vertical');
  const [formData, setFormData]               = useState(EMPTY_FORM);
  const [editingId, setEditingId]             = useState(null);

  // ── Hooks ─────────────────────────────────────────────────────────────────
  const connections = useConnections(apiUrl);
  const tabs        = useTabs(apiUrl, activeId, setLoading);
  const grid        = useEditableGrid(apiUrl, activeId, tabs.activeTab, tabs.updateActiveTab, tabs.executeQuery, setLoading);
  const ctxMenu     = useContextMenu();
  const exp         = useExport(apiUrl, activeId, tabs.activeTab, tabs.updateActiveTab, setLoading);

  // ── Initial polling ───────────────────────────────────────────────────────
  useEffect(() => {
    connections.fetchConnections();
    connections.fetchFolders();
    const interval = setInterval(connections.fetchConnections, 2000);
    return () => clearInterval(interval);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // ── Close context menu on global click ───────────────────────────────────
  useEffect(() => {
    const close = () => { if (ctxMenu.contextMenu.visible) ctxMenu.hideMenu(); };
    window.addEventListener('click', close);
    return () => window.removeEventListener('click', close);
  }, [ctxMenu]);

  // ── Connection form helpers ───────────────────────────────────────────────
  const openNewConnection = () => { setEditingId(null); setFormData(EMPTY_FORM); setShowModal(true); };

  const handleEditConnection = (c, e) => {
    e.stopPropagation();
    setEditingId(c.id);
    setFormData({ 
      connection_name: c.connection_name || '', 
      db_type: c.db_type || 'postgres', 
      host: c.host || '', 
      port: (typeof c.port === 'number' ? c.port : 5432),
      dbname: c.dbname || '', 
      file_path: c.file_path || '',
      user: c.user || '', 
      password: '', 
      enable_ssl: c.enable_ssl || false, 
      ca_cert_path: c.ca_cert_path || '', 
      client_cert_path: c.client_cert_path || '', 
      client_key_path: c.client_key_path || '', 
      folder_id: c.folder_id || '' 
    });
    setShowModal(true);
  };

  const handleConnectNew = async (e) => {
    e.preventDefault();
    try {
      const unsupportedNoSql = ['mongodb', 'cassandra', 'elasticsearch', 'opensearch'].includes(formData.db_type);
      if (unsupportedNoSql) {
        alert(`${formData.db_type} connections are not implemented in the engine yet.`);
        return;
      }

      const isFileDb = formData.db_type === 'sqlite';
      const portInt = isFileDb ? 0 : parseInt(formData.port);
      if (!isFileDb && isNaN(portInt)) {
        alert('Please enter a valid port number');
        return;
      }

      const isRedis = formData.db_type === 'redis';
      const base    = isRedis ? `${apiUrl}/api/redis` : `${apiUrl}/api/connections`;
      const url     = editingId ? `${base}/${editingId}` : base;
      const method  = editingId ? 'PUT' : 'POST';
      
      const res    = await fetch(url, { 
        method, 
        headers: { 'Content-Type': 'application/json' }, 
        body: JSON.stringify({ ...formData, port: portInt }) 
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
    } catch (err) { alert('Failed to save connection: ' + err.message); }
  };

  const handleTestConnection = async (e) => {
    e.preventDefault();
    try {
      const unsupportedNoSql = ['mongodb', 'cassandra', 'elasticsearch', 'opensearch'].includes(formData.db_type);
      if (unsupportedNoSql) {
        alert(`${formData.db_type} connections are not implemented in the engine yet.`);
        return;
      }

      const isFileDb = formData.db_type === 'sqlite';
      const portInt = isFileDb ? 0 : parseInt(formData.port);
      if (!isFileDb && isNaN(portInt)) {
        alert('Please enter a valid port number');
        return;
      }

      const isRedis = formData.db_type === 'redis';
      const url     = isRedis ? `${apiUrl}/api/redis/test` : `${apiUrl}/api/connections/test`;
      
      const res  = await fetch(url, { 
        method: 'POST', 
        headers: { 'Content-Type': 'application/json' }, 
        body: JSON.stringify({ ...formData, port: portInt }) 
      });
      
      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || res.statusText);
      }

      const data = await res.json();
      data.success ? alert('Success: ' + data.message) : alert('Connection Failed:\n\n' + data.error);
    } catch (err) { alert('Error reaching engine: ' + err.message); }
  };

  const handleDeleteConnection = (id, e) => {
    e.stopPropagation();
    connections.handleDeleteConnection(id, activeId, setActiveId);
  };

  const handleDisconnect = (id) => connections.handleDisconnect(id, activeId, setActiveId);
  const handleReconnect  = (id) => { connections.handleReconnect(id); setActiveId(id); };

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

  const activeTabConnId = tabs.activeTab?.connectionId || null;
  useEffect(() => {
    if (activeTabConnId && activeTabConnId !== activeId) {
      setActiveId(activeTabConnId);
    }
  }, [activeTabConnId, activeId]);

  // ── Context menu action dispatcher ───────────────────────────────────────
  const handleMenuAction = async (action) => {
    ctxMenu.handleMenuAction(action, {
      onConnAction: async (act, conn) => {
        switch (act) {
          case 'duplicate': setEditingId(null); setFormData({ ...conn, connection_name: conn.connection_name + ' (Copy)', password: '' }); setShowModal(true); break;
          case 'delete':    handleDeleteConnection(conn.id, { stopPropagation: () => {} }); break;
          case 'refresh':   connections.fetchConnections(); break;
          case 'connect':   handleReconnect(conn.id); break;
          case 'disconnect':handleDisconnect(conn.id); break;
          case 'rename':    handleEditConnection(conn, { stopPropagation: () => {} }); break;
        }
      },
      onTableAction: async (act, { tbl, schemaName, cId }) => {
        switch (act) {
          case 'view_data': {
            const q = `SELECT * FROM ${schemaName}.${tbl.name} LIMIT 100;`;
            setActiveId(cId);
            tabs.updateActiveTab({ query: q, connectionId: cId });
            await tabs.executeQuery(q, cId);
            break;
          }
          case 'view_dml': {
            const res = await fetch(`${apiUrl}/api/connections/${cId}/metadata/schemas/${schemaName}/tables/${tbl.name}/dml`);
            const d   = await res.json();
            if (d.success) {
              tabs.addSpecialTab(`DML: ${tbl.name}`, 'dml', { results: { success: true, dml: d.dml } }, cId);
            } else {
              alert('Error fetching DML: ' + d.error);
            }
            break;
          }
          case 'view_indexes': {
            const res = await fetch(`${apiUrl}/api/connections/${cId}/metadata/schemas/${schemaName}/tables/${tbl.name}/indexes`);
            const d   = await res.json();
            if (!d.success) { alert('Error: ' + d.error); break; }
            if (!d.indexes?.length) { alert('No indexes found.'); break; }
            tabs.addSpecialTab(`Indexes: ${tbl.name}`, 'indexes', { 
              results: { 
                success: true, 
                columns: ['Name', 'Columns', 'Unique', 'Primary'], 
                rows: d.indexes.map(i => [i.name, i.columns.join(', '), i.unique ? 'Yes' : 'No', i.primary ? 'Yes' : 'No']) 
              } 
            }, cId);
            break;
          }
          case 'export': exp.handleExportClick('table', tbl.name, tbl.name); break;
        }
      },
    });
  };

  // ── Row selection (lives here because it touches tab state) ──────────────
  const handleRowClick = (e, targetIndex) => {
    let sel = [...(tabs.activeTab.selectedRows || [])];
    if (e.shiftKey && tabs.activeTab.lastSelectedIndex !== null) {
      const [lo, hi] = [Math.min(tabs.activeTab.lastSelectedIndex, targetIndex), Math.max(tabs.activeTab.lastSelectedIndex, targetIndex)];
      if (!e.metaKey && !e.ctrlKey) sel = [];
      for (let i = lo; i <= hi; i++) { if (!sel.includes(i)) sel.push(i); }
    } else if (e.metaKey || e.ctrlKey) {
      sel = sel.includes(targetIndex) ? sel.filter(i => i !== targetIndex) : [...sel, targetIndex];
    } else {
      sel = [targetIndex];
    }
    tabs.updateActiveTab({ selectedRows: sel, lastSelectedIndex: targetIndex });
  };

  // ── Connection info for status bar ───────────────────────────────────────
  const activeConn = connections.connections.find(c => c.id === activeId);

  // ── Render ────────────────────────────────────────────────────────────────
  return (
    <div className="app-container">
      <Sidebar
        {...connections}
        activeId={activeId}
        sidebarCollapsed={sidebarCollapsed}
        setSidebarCollapsed={setSidebarCollapsed}
        handleConnectionClick={handleConnectionClick}
        handleEditConnection={handleEditConnection}
        handleDeleteConnection={handleDeleteConnection}
        handleDisconnect={handleDisconnect}
        handleReconnect={handleReconnect}
        onNewConnection={openNewConnection}
        onNewScript={() => {
          const n = tabs.tabs.filter(t => t.type === 'script').length + 1;
          tabs.addSpecialTab(`Script ${n}`, 'script', { query: '', results: null }, null);
        }}
        onShowHelp={() => setShowHelpModal(true)}
        handleConnectionContextMenu={ctxMenu.handleConnectionContextMenu}
        handleTableContextMenu={ctxMenu.handleTableContextMenu}
        toggleTree={connections.toggleTree}
      />

      <button className="swap-layout-btn" onClick={() => setLayoutDirection(d => d === 'horizontal' ? 'vertical' : 'horizontal')} title="Swap Layout">
        ⇋ {layoutDirection === 'horizontal' ? 'Horizontal' : 'Vertical'}
      </button>

      <div className="main-area">
        <QueryTabs
          {...tabs}
          activeId={activeId}
          loading={loading}
          activeConnType={activeConn?.db_type}
        />

        {tabs.activeTab.type === 'script' ? (
          <ScriptPane
            apiUrl={apiUrl}
            activeTab={tabs.activeTab}
            updateActiveTab={tabs.updateActiveTab}
            loading={loading}
            setLoading={setLoading}
            connections={connections.connections}
          />
        ) : tabs.activeTab.type === 'query' ? (
          <Split
            key={`${layoutDirection}-${activeConn?.db_type || 'none'}`}
            className={`split-container ${layoutDirection}`}
            direction={layoutDirection}
            sizes={[40, 60]}
            minSize={100}
            gutterSize={8}
          >
            {/* Query editor pane */}
            <div className="pane query-section editor-pane">
              {activeConn?.db_type === 'redis' ? (
                <RedisAutocomplete
                  value={tabs.activeTab.query}
                  onChange={code => tabs.updateActiveTab({ query: code })}
                  onKeyDown={(e) => tabs.handleKeyDown(e, tabs.activeTab.connectionId || activeId, 'redis')}
                  disabled={!activeId}
                />
              ) : (
                <SqlAutocomplete
                  value={tabs.activeTab.query}
                  onChange={code => tabs.updateActiveTab({ query: code })}
                  onKeyDown={(e) => tabs.handleKeyDown(e, tabs.activeTab.connectionId || activeId, activeConn?.db_type)}
                  disabled={!activeId}
                  placeholder={activeId ? 'Type SQL query here... (Cmd/Ctrl + Enter to run)' : 'Select or add a connection to start'}
                  apiUrl={apiUrl}
                  connectionId={tabs.activeTab.connectionId || activeId}
                />
              )}

              {/* Status information bar (placed above the gutter) */}
              <div className="editor-status-bar">
                <div>{activeConn ? `${activeConn.connection_name} (${activeConn.db_type})` : 'No connection selected'}</div>
                <div>{tabs.activeTab.results?.durationMs !== undefined ? `${tabs.activeTab.results.durationMs.toFixed(2)} ms` : ''}</div>
              </div>
            </div>

            {/* Results pane */}
            {activeConn?.db_type === 'redis' ? (
              <RedisPane
                activeTab={tabs.activeTab}
                loading={loading}
                connectionId={activeId}
                onQuery={tabs.executeQuery}
                onExport={(redisResult, command) => exp.handleExportRedisResult(redisResult, command)}
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
                handleRowContextMenu={exp.handleRowContextMenu}
                handleExportClick={exp.handleExportClick}
                computeWorkingData={grid.computeWorkingData}
              />
            )}
          </Split>
        ) : (
          activeConn?.db_type === 'redis' ? (
            <RedisPane
              activeTab={tabs.activeTab}
              loading={loading}
              connectionId={activeId}
              onQuery={tabs.executeQuery}
              onExport={(redisResult, command) => exp.handleExportRedisResult(redisResult, command)}
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
              handleRowContextMenu={exp.handleRowContextMenu}
              handleExportClick={exp.handleExportClick}
              computeWorkingData={grid.computeWorkingData}
            />
          )
        )}
      </div>

      {/* Modals */}
      <ConnectionModal
        show={showModal}
        editingId={editingId}
        formData={formData}
        setFormData={setFormData}
        folders={connections.folders}
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
        connectionName={connections.connections.find(c => c.id === switchPrompt?.targetId)?.connection_name || 'Selected connection'}
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
      <HelpModal show={showHelpModal} onClose={() => setShowHelpModal(false)} />
      <ContextMenu contextMenu={ctxMenu.contextMenu} onAction={handleMenuAction} />
    </div>
  );
}

export default App
