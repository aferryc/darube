import React from 'react';

export function ConnectionModal({
  show, editingId, formData, setFormData, folders,
  onSubmit, onTest, onClose,
}) {
  const [activeTab, setActiveTab] = React.useState('general');
  const [connectionCategory, setConnectionCategory] = React.useState('sql');

  // Sync category with form data if editing
  React.useEffect(() => {
    if (show && editingId) {
      if (formData.db_type === 'redis') {
        setConnectionCategory('nosql');
      } else {
        setConnectionCategory('sql');
      }
    }
  }, [show, editingId, formData.db_type]);

  if (!show) return null;

  const update = (field, value) => setFormData(prev => ({ ...prev, [field]: value }));

  const handleCategoryChange = (cat) => {
    setConnectionCategory(cat);
    if (cat === 'nosql') {
      update('db_type', 'redis');
      if (!formData.port) update('port', 6379);
    } else {
      update('db_type', 'postgres');
    }
  };

  return (
    <div className="modal-overlay">
      <div className="modal-content w-450">
        <div className="modal-header">
          <div className="modal-header-left">
            <h3>{editingId ? 'Edit Connection' : 'New Connection'}</h3>
            <div className="category-tabs">
              <span 
                className={`category-tab ${connectionCategory === 'sql' ? 'active' : ''}`}
                onClick={() => handleCategoryChange('sql')}
              >
                SQL
              </span>
              <span 
                className={`category-tab ${connectionCategory === 'nosql' ? 'active' : ''}`}
                onClick={() => handleCategoryChange('nosql')}
              >
                NoSQL
              </span>
            </div>
          </div>
          <div className="modal-tabs">
            {connectionCategory === 'sql' && (
              <>
                <button 
                  type="button" 
                  className={`modal-tab ${activeTab === 'general' ? 'active' : ''}`}
                  onClick={() => setActiveTab('general')}
                >
                  General
                </button>
                <button 
                  type="button" 
                  className={`modal-tab ${activeTab === 'ssl' ? 'active' : ''}`}
                  onClick={() => setActiveTab('ssl')}
                >
                  SSL
                </button>
              </>
            )}
          </div>
        </div>

        <form onSubmit={onSubmit} className="compact-form">
          {connectionCategory === 'sql' ? (
            activeTab === 'general' ? (
              <>
                <div className="form-group">
                  <label>Name</label>
                  <input autoFocus required value={formData.connection_name} onChange={e => update('connection_name', e.target.value)} placeholder="Production DB" />
                </div>

                <div className="form-row">
                  <div className="form-group">
                    <label>Database Type</label>
                    <select value={formData.db_type} onChange={e => update('db_type', e.target.value)}>
                      <option value="postgres">PostgreSQL</option>
                      <option value="mysql">MySQL / MariaDB</option>
                      <option value="sqlserver">SQL Server</option>
                    </select>
                  </div>
                  {folders.length > 0 && (
                    <div className="form-group">
                      <label>Folder (Optional)</label>
                      <select value={formData.folder_id || ''} onChange={e => update('folder_id', e.target.value)}>
                        <option value="">Uncategorized</option>
                        {folders.map(f => <option key={f.id} value={f.id}>{f.name}</option>)}
                      </select>
                    </div>
                  )}
                </div>

                <div className="form-row">
                  <div className="form-group flex-2">
                    <label>Host</label>
                    <input required value={formData.host} onChange={e => update('host', e.target.value)} placeholder="localhost" />
                  </div>
                  <div className="form-group flex-1">
                    <label>Port</label>
                    <input type="number" required value={formData.port} onChange={e => update('port', e.target.value)} />
                  </div>
                </div>

                <div className="form-group">
                  <label>Database (Optional)</label>
                  <input value={formData.dbname} onChange={e => update('dbname', e.target.value)} placeholder="postgres" />
                </div>

                <div className="form-row">
                  <div className="form-group">
                    <label>Username</label>
                    <input required value={formData.user} onChange={e => update('user', e.target.value)} />
                  </div>
                  <div className="form-group">
                    <label>Password</label>
                    <input type="password" value={formData.password} onChange={e => update('password', e.target.value)} />
                  </div>
                </div>
              </>
            ) : (
              <>
                <div className="form-group mb-20">
                  <label className="form-checkbox-label">
                    <input type="checkbox" checked={formData.enable_ssl} onChange={e => update('enable_ssl', e.target.checked)} />
                    Enable SSL Configuration
                  </label>
                  <div className="form-help-text">
                    Required for encrypted database connections (Postgres/MySQL/SQL Server).
                  </div>
                </div>

                {formData.enable_ssl && (
                  <div className="flex-column gap-12">
                    <div className="form-group">
                      <label>CA Certificate (PEM/CRT)</label>
                      <div className="form-file-input-row">
                        <input
                          value={formData.ca_cert_path || ''}
                          onChange={e => update('ca_cert_path', e.target.value)}
                          placeholder="/absolute/path/to/ca.pem"
                        />
                        <button
                          type="button"
                          className="secondary browse-btn"
                          onClick={async () => {
                            if (window.require) {
                              const { ipcRenderer } = window.require('electron');
                              const path = await ipcRenderer.invoke('dialog:openFile');
                              if (path) update('ca_cert_path', path);
                            }
                          }}
                        >
                          Browse...
                        </button>
                      </div>
                    </div>

                    <div className="form-group">
                      <label>Client Certificate</label>
                      <div className="form-file-input-row">
                        <input
                          value={formData.client_cert_path || ''}
                          onChange={e => update('client_cert_path', e.target.value)}
                          placeholder="/absolute/path/to/client.crt"
                        />
                        <button
                          type="button"
                          className="secondary browse-btn"
                          onClick={async () => {
                            if (window.require) {
                              const { ipcRenderer } = window.require('electron');
                              const path = await ipcRenderer.invoke('dialog:openFile');
                              if (path) update('client_cert_path', path);
                            }
                          }}
                        >
                          Browse...
                        </button>
                      </div>
                    </div>

                    <div className="form-group">
                      <label>Client Key</label>
                      <div className="form-file-input-row">
                        <input
                          value={formData.client_key_path || ''}
                          onChange={e => update('client_key_path', e.target.value)}
                          placeholder="/absolute/path/to/client.key"
                        />
                        <button
                          type="button"
                          className="secondary browse-btn"
                          onClick={async () => {
                            if (window.require) {
                              const { ipcRenderer } = window.require('electron');
                              const path = await ipcRenderer.invoke('dialog:openFile');
                              if (path) update('client_key_path', path);
                            }
                          }}
                        >
                          Browse...
                        </button>
                      </div>
                    </div>
                  </div>
                )}
              </>
            )
          ) : (
            <>
              {/* NoSQL / Redis Form */}
              <div className="form-group">
                <label>Connection Name</label>
                <input autoFocus required value={formData.connection_name} onChange={e => update('connection_name', e.target.value)} placeholder="Redis Production" />
              </div>

              <div className="form-row">
                <div className="form-group flex-2">
                  <label>Host</label>
                  <input required value={formData.host} onChange={e => update('host', e.target.value)} placeholder="localhost" />
                </div>
                <div className="form-group flex-1">
                  <label>Port</label>
                  <input type="number" value={formData.port || 6379} onChange={e => update('port', e.target.value)} />
                </div>
              </div>

              <div className="form-row">
                <div className="form-group">
                  <label>Username (Optional)</label>
                  <input value={formData.user} onChange={e => update('user', e.target.value)} placeholder="default" />
                </div>
                <div className="form-group">
                  <label>Password (Optional)</label>
                  <input type="password" value={formData.password} onChange={e => update('password', e.target.value)} />
                </div>
              </div>

              <div className="form-group mt-10">
                <label className="form-checkbox-label">
                  <input type="checkbox" checked={formData.is_cluster} onChange={e => update('is_cluster', e.target.checked)} />
                  Enable Cluster Mode
                </label>
                <div className="form-help-text">
                  Enable if connecting to a Redis Cluster (uses UniversalClient).
                </div>
              </div>
            </>
          )}

          <div className="modal-footer">
            <button type="button" className="secondary" onClick={onClose}>Cancel</button>
            <button type="button" className="secondary test-link-btn" onClick={onTest}>Test Link</button>
            <button type="submit">Save</button>
          </div>
        </form>
      </div>
    </div>
  );
}
