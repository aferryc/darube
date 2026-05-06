import { useEffect, useState } from "react";

export function ContextMenu({ contextMenu, onAction }) {
  if (!contextMenu.visible) return null;

  const item = (label, action, { disabled = false } = {}) => (
    <div
      className={`context-menu-item${disabled ? ' disabled' : ''}`}
      onClick={disabled ? undefined : () => onAction(action)}
      role="menuitem"
      aria-disabled={disabled ? 'true' : 'false'}
    >
      {label}
    </div>
  );

  return (
    <div className="context-menu" style={{ top: `${contextMenu.y}px`, left: `${contextMenu.x}px` }}>
      {contextMenu.type === 'connection' && (
        <>
          {item('📝 Duplicate Connection', 'duplicate')}
          {item('✏️ Rename Connection', 'rename')}
          <div className="context-menu-divider" />
          {contextMenu.data.status === 'connected'
            ? item('🛑 Disconnect', 'disconnect')
            : item('🟢 Connect', 'connect')
          }
          {item('🔄 Refresh', 'refresh')}
          {item('📏 Table Size Cache', 'view_table_sizes', { disabled: contextMenu.data.status !== 'connected' })}
          <div className="context-menu-divider" />
          <div className="context-menu-item text-danger" onClick={() => onAction('delete')} role="menuitem">🗑️ Delete Connection</div>
        </>
      )}
      {contextMenu.type === 'table' && (
        <>
          {item('👁️ View Data (TOP 100)', 'view_data')}
          <div className="context-menu-divider" />
          {item('📜 View DB DML', 'view_dml')}
          {item('🔑 View Indexes', 'view_indexes')}
          <div className="context-menu-divider" />
          {item('📤 Export Data...', 'export')}
        </>
      )}
      {contextMenu.type === 'text' && (
        <>
          {(contextMenu.data?.editorRole === 'sql' || contextMenu.data?.editorRole === 'redis') && (
            <>
              {item(
                contextMenu.data?.hasSelection
                  ? 'Run Selected'
                  : (contextMenu.data?.editorRole === 'redis' ? 'Run Command' : 'Run Query'),
                'run_query',
                { disabled: !(contextMenu.data?.hasSelection || contextMenu.data?.hasText) },
              )}
              <div className="context-menu-divider" />
            </>
          )}
          {item('Cut', 'cut', { disabled: contextMenu.data?.readOnly || !contextMenu.data?.hasSelection })}
          {item('Copy', 'copy', { disabled: !contextMenu.data?.hasSelection })}
          {item('Paste', 'paste', { disabled: contextMenu.data?.readOnly })}
          <div className="context-menu-divider" />
          {item('Select All', 'select_all')}
        </>
      )}
      {contextMenu.type === 'results' && (
        <>
          {item('Copy Cell', 'copy_cell', { disabled: contextMenu.data?.colIndex == null })}
          {item('Copy Row (TSV)', 'copy_row_tsv', { disabled: contextMenu.data?.rowIndex == null })}
          {item('Copy Row (JSON)', 'copy_row_json', { disabled: contextMenu.data?.rowIndex == null })}
          <div className="context-menu-divider" />
          {item('Copy Selected Rows (TSV)', 'copy_selected_tsv', { disabled: !(contextMenu.data?.selectedRows?.length > 0) })}
          {item('Export Selected Rows...', 'export_selected', { disabled: !(contextMenu.data?.selectedRows?.length > 0) })}
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
      <div className="modal-content w-500 help-modal">
        <div className="help-header">
          <div>
            <h3>Help & Information</h3>
            <div className="help-subtitle">Quick guide to get started and move faster.</div>
          </div>
          <button type="button" className="btn-icon close-btn" onClick={onClose} aria-label="Close help">✕</button>
        </div>

        <div className="help-sections">
          <div className="help-card">
            <div className="help-card-title">Getting Started</div>
            <ul className="help-list">
              <li>Use the Sidebar to add new connections and switch between them.</li>
              <li>Right-click a connection to edit, refresh, or remove it.</li>
              <li>Create folders to keep related connections grouped.</li>
            </ul>
          </div>

          <div className="help-card">
            <div className="help-card-title">Running Queries</div>
            <ul className="help-list">
              <li>Type your SQL in the Editor, then run the query.</li>
              <li>Select a portion of SQL to run only that selection.</li>
              <li>Results and execution plans appear on the right panel.</li>
            </ul>
          </div>

            <div className="help-card">
              <div className="help-card-title">Shortcuts</div>
              <div className="help-shortcuts">
                <div className="help-shortcut">
                  <span>Run Query</span>
                  <span className="help-kbd">CMD/CTRL</span>
                  <span className="help-kbd">ENTER</span>
                </div>
              </div>
            </div>
        </div>

        <div className="modal-footer">
          <button type="button" className="secondary" onClick={onClose}>Close</button>
        </div>
      </div>
    </div>
  );
}

export function TeleportLoginModal({
  show,
  apiUrl,
  defaultProxy,
  reason,
  onCancel,
  onSuccess,
}) {
  const [proxy, setProxy] = useState(defaultProxy || "");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [output, setOutput] = useState("");

  useEffect(() => {
    if (!show) return;
    setProxy(defaultProxy || "");
    setLoading(false);
    setError("");
    setOutput("");
  }, [show, defaultProxy]);

  if (!show) return null;

  const doLogin = async (e) => {
    e?.preventDefault?.();
    const proxyHost = String(proxy || "").trim();
    if (!proxyHost) {
      setError("Proxy is required (e.g. mytenant.teleport.sh)");
      return;
    }

    setLoading(true);
    setError("");
    setOutput("");
    try {
      const res = await fetch(`${apiUrl}/api/teleport/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ proxy: proxyHost }),
      });
      const data = await res.json().catch(() => ({}));
      if (data?.success) {
        setOutput(String(data.output || ""));
        onSuccess?.({ output: String(data.output || ""), proxy: proxyHost });
        return;
      }
      setError(String(data?.error || "Teleport login failed"));
      if (data?.output) setOutput(String(data.output));
    } catch (err) {
      setError(err?.message ? String(err.message) : "Teleport login failed");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="modal-overlay">
      <div className="modal-content w-450">
        <div className="modal-header">
          <div className="modal-header-left">
            <h3>Teleport Login</h3>
            <span className="settings-subtitle">
              Darube uses <code>tsh</code> for Teleport database access.
            </span>
          </div>
          <button
            type="button"
            className="btn-icon close-btn"
            onClick={() => onCancel?.()}
            aria-label="Close"
            disabled={loading}
          >
            ✕
          </button>
        </div>

        {reason && (
          <div className="form-help-text" style={{ marginBottom: 10 }}>
            {reason}
          </div>
        )}

        <form onSubmit={doLogin}>
          <div className="form-group">
            <label>Proxy Host</label>
            <input
              value={proxy}
              onChange={(e) => setProxy(e.target.value)}
              placeholder="mytenant.teleport.sh"
              autoFocus
            />
            <div className="form-help-text">
              This may open your browser for SSO/MFA. If your cluster uses a custom HTTPS port, include it (e.g. <code>proxy.example.com:3080</code>).
            </div>
          </div>

          {error && (
            <div className="form-help-text" style={{ color: "var(--danger)", marginTop: 10 }}>
              {error}
            </div>
          )}
          {output && (
            <pre
              style={{
                marginTop: 10,
                maxHeight: 140,
                overflow: "auto",
                padding: 10,
                borderRadius: 10,
                background: "rgba(255,255,255,0.04)",
                border: "1px solid var(--border)",
                whiteSpace: "pre-wrap",
                color: "var(--text-muted)",
                fontSize: 12,
              }}
            >
              {output}
            </pre>
          )}

          <div className="modal-footer no-border">
            <button type="button" className="secondary" onClick={() => onCancel?.()} disabled={loading}>
              Cancel
            </button>
            <button type="submit" disabled={loading}>
              {loading ? "Logging in..." : "Login"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
