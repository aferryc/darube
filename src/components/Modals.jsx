export function ContextMenu({ contextMenu, onAction }) {
  if (!contextMenu.visible) return null;

  return (
    <div className="context-menu" style={{ top: `${contextMenu.y}px`, left: `${contextMenu.x}px` }}>
      {contextMenu.type === 'connection' && (
        <>
          <div className="context-menu-item" onClick={() => onAction('duplicate')}>📝 Duplicate Connection</div>
          <div className="context-menu-item" onClick={() => onAction('rename')}>✏️ Rename Connection</div>
          <div className="context-menu-divider" />
          {contextMenu.data.status === 'connected'
            ? <div className="context-menu-item" onClick={() => onAction('disconnect')}>🛑 Disconnect</div>
            : <div className="context-menu-item" onClick={() => onAction('connect')}>🟢 Connect</div>
          }
          <div className="context-menu-item" onClick={() => onAction('refresh')}>🔄 Refresh</div>
          <div className="context-menu-divider" />
          <div className="context-menu-item text-danger" onClick={() => onAction('delete')}>🗑️ Delete Connection</div>
        </>
      )}
      {contextMenu.type === 'table' && (
        <>
          <div className="context-menu-item" onClick={() => onAction('view_data')}>👁️ View Data (TOP 100)</div>
          <div className="context-menu-divider" />
          <div className="context-menu-item" onClick={() => onAction('view_dml')}>📜 View DB DML</div>
          <div className="context-menu-item" onClick={() => onAction('view_indexes')}>🔑 View Indexes</div>
          <div className="context-menu-divider" />
          <div className="context-menu-item" onClick={() => onAction('export')}>📤 Export Data...</div>
        </>
      )}
    </div>
  );
}

export function ExportModal({ exportConfig, setExportConfig, loading, onSubmit, onSelectDirectory }) {
  if (!exportConfig) return null;

  const update = (field, value) => setExportConfig(prev => ({ ...prev, [field]: value }));
  const isRedis = exportConfig.targetType === 'redis';
  const title = isRedis
    ? 'Export Redis Result'
    : (exportConfig.targetType === 'table' ? `Export Table: ${exportConfig.targetName}` : 'Export Query Results');

  return (
    <div className="modal-overlay">
      <div className="modal-content w-450">
        <h3>{title}</h3>
        <form onSubmit={onSubmit} className="mt-20">
          <div className="form-group">
            <label>Export Format</label>
            <select value={exportConfig.format} onChange={e => update('format', e.target.value)}>
              <option value="csv">CSV (Comma-Separated Values)</option>
              <option value="json">JSON (Array of Objects)</option>
              {!isRedis && <option value="sql">SQL (INSERT INTO statements)</option>}
              <option value="excel">Excel (.xlsx)</option>
            </select>
          </div>
          {(exportConfig.format === 'csv' || exportConfig.format === 'excel') && (
            <div className="form-group mt-10">
              <label className="form-checkbox-label">
                <input type="checkbox" checked={exportConfig.headers} onChange={e => update('headers', e.target.checked)} />
                Include Headers (Column Names)
              </label>
            </div>
          )}
          <div className="form-group mt-15">
            <label>Destination Directory (Absolute Path)</label>
            <div className="form-file-input-row">
              <input required value={exportConfig.path} onChange={e => update('path', e.target.value)} placeholder="/tmp/ or C:\Users\" className="flex-grow" />
              <button type="button" onClick={onSelectDirectory} className="secondary" title="Requires Darube Desktop App">Browse</button>
            </div>
          </div>
          <div className="form-group">
            <label>Filename (without extension)</label>
            <input required value={exportConfig.filename} onChange={e => update('filename', e.target.value)} placeholder="export_data" />
          </div>
          <div className="modal-footer no-border">
            <button type="button" className="secondary" onClick={() => setExportConfig(null)} disabled={loading}>Cancel</button>
            <button type="submit" disabled={loading}>{loading ? 'Exporting...' : 'Export NOW'}</button>
          </div>
        </form>
      </div>
    </div>
  );
}

export function ConnectionSwitchModal({ show, connectionName, onCreateNewTab, onRebindCurrentTab, onCancel }) {
  if (!show) return null;
  return (
    <div className="modal-overlay">
      <div className="modal-content w-400 conn-switch-modal">
        <div className="conn-switch-header">
          <div className="conn-switch-title">Open Connection</div>
          <div className="conn-switch-subtitle">
            No tabs yet for <span className="conn-pill">{connectionName}</span>
          </div>
        </div>
        <div className="conn-switch-body">
          Create a new tab for this connection, or rebind the current tab.
        </div>
        <div className="conn-switch-actions">
          <button type="button" className="secondary conn-btn" onClick={onCancel}>Cancel</button>
          <button type="button" className="secondary conn-btn" onClick={onRebindCurrentTab}>Change Tab</button>
          <button type="button" className="conn-btn primary" onClick={onCreateNewTab}>New Tab</button>
        </div>
      </div>
    </div>
  );
}

export function HelpModal({ show, onClose }) {
  if (!show) return null;
  return (
    <div className="modal-overlay">
      <div className="modal-content w-500">
        <h3>Help & Information</h3>
        <div className="help-content">
          <p>Welcome to <strong>Darube</strong>!</p><br />
          <p>Using the Sidebar you can add new Database Connections, stop operations, edit, or remove them.</p><br />
          <p><strong>Running Queries</strong></p>
          <ul className="help-list">
            <li>Type your SQL queries in the Editor.</li>
            <li>Press <code>CMD/CTRL + ENTER</code> to run the query, or use the Run Query button.</li>
            <li>You can highlight a specific part of your code to selectively run only that query piece!</li>
          </ul>
        </div>
        <div className="modal-footer">
          <button type="button" onClick={onClose}>OK</button>
        </div>
      </div>
    </div>
  );
}
