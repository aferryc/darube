import React from 'react';

const NOSQL_TYPES = new Set(['redis', 'mongodb', 'cassandra', 'elasticsearch', 'opensearch']);

export function ConnectionModal({
  show, editingId, formData, setFormData, folders,
  onSubmit, onTest, onClose,
}) {
  const [activeTab, setActiveTab] = React.useState('general');
  const [connectionCategory, setConnectionCategory] = React.useState('sql');

  // Sync category with form data if editing
  React.useEffect(() => {
    if (show && editingId) {
      if (NOSQL_TYPES.has(formData.db_type)) {
        setConnectionCategory('nosql');
      } else {
        setConnectionCategory('sql');
      }
    }
  }, [show, editingId, formData.db_type]);

  React.useEffect(() => {
    if (formData.db_type === 'sqlite') setActiveTab('general');
  }, [formData.db_type]);

  if (!show) return null;

  const update = (field, value) => setFormData(prev => ({ ...prev, [field]: value }));
  const isSQLite = formData.db_type === 'sqlite';
  const isOracle = formData.db_type === 'oracle';

  const handleCategoryChange = (cat) => {
    setConnectionCategory(cat);
    if (cat === 'nosql') {
      if (!NOSQL_TYPES.has(formData.db_type)) {
        update('db_type', 'redis');
        update('port', 6379);
      }
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
            {connectionCategory === 'sql' && !isSQLite && (
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
                    <select
                      value={formData.db_type}
                      onChange={e => {
                        const next = e.target.value;
                        update('db_type', next);
                        if (next === 'sqlite') {
                          update('host', '');
                          update('port', 0);
                          update('user', '');
                          update('password', '');
                          update('enable_ssl', false);
                        } else if (!formData.port || formData.port === 0) {
                          // Restore a reasonable default if user came from sqlite.
                          update('port', next === 'mysql' ? 3306 : (next === 'sqlserver' ? 1433 : (next === 'oracle' ? 1521 : 5432)));
                        }
                      }}
                    >
                      <option value="postgres">PostgreSQL</option>
                      <option value="mysql">MySQL / MariaDB</option>
                      <option value="sqlserver">SQL Server</option>
                      <option value="oracle">Oracle</option>
                      <option value="sqlite">SQLite</option>
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

                {isSQLite ? (
                  <div className="form-group">
                    <label>Database File</label>
                    <div className="form-file-input-row">
                      <input
                        required
                        value={formData.file_path || ''}
                        onChange={e => update('file_path', e.target.value)}
                        placeholder="/absolute/path/to/db.sqlite (or :memory:)"
                      />
                      <button
                        type="button"
                        className="secondary browse-btn"
                        onClick={async () => {
                          if (window.require) {
                            const { ipcRenderer } = window.require('electron');
                            const path = await ipcRenderer.invoke('dialog:openFile');
                            if (path) update('file_path', path);
                          }
                        }}
                      >
                        Browse...
                      </button>
                    </div>
                    <div className="form-help-text" style={{ marginLeft: 0 }}>
                      Use <code>:memory:</code> for an in-memory database.
                    </div>
                  </div>
                ) : (
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
                )}

                {!isSQLite && (
                  <div className="form-group">
                    <label>{isOracle ? 'Service Name' : 'Database (Optional)'}</label>
                    <input
                      value={formData.dbname}
                      onChange={e => update('dbname', e.target.value)}
                      placeholder={isOracle ? 'orclpdb1' : 'postgres'}
                      required={isOracle}
                    />
                  </div>
                )}

                {!isSQLite && (
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
                )}
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

              <div className="form-group">
                <label>Database Type</label>
                <select
                  value={formData.db_type}
                  onChange={e => {
                    const next = e.target.value;
                    update('db_type', next);
                    const defaults = { redis: 6379, mongodb: 27017, cassandra: 9042, elasticsearch: 9200, opensearch: 9200 };
                    update('port', defaults[next] ?? formData.port ?? 0);
                    if (next === 'redis') update('dbname', '');
                  }}
                >
                  <option value="redis">Redis</option>
                  <option value="mongodb">MongoDB</option>
                  <option value="cassandra">Cassandra</option>
                  <option value="elasticsearch">Elasticsearch</option>
                  <option value="opensearch">OpenSearch</option>
                </select>
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

              {formData.db_type !== 'redis' && (
                <div className="form-group">
                  <label>
                    {formData.db_type === 'mongodb' ? 'Database (Optional)' :
                      formData.db_type === 'cassandra' ? 'Keyspace (Optional)' :
                        'Index (Optional)'}
                  </label>
                  <input
                    value={formData.dbname || ''}
                    onChange={e => update('dbname', e.target.value)}
                    placeholder={
                      formData.db_type === 'mongodb' ? 'app' :
                        formData.db_type === 'cassandra' ? 'keyspace' :
                          'logs-*'
                    }
                  />
                </div>
              )}

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

              {formData.db_type === 'redis' && (
                <div className="form-group mt-10">
                  <label className="form-checkbox-label">
                    <input type="checkbox" checked={formData.is_cluster} onChange={e => update('is_cluster', e.target.checked)} />
                    Enable Cluster Mode
                  </label>
                  <div className="form-help-text">
                    Enable if connecting to a Redis Cluster (uses UniversalClient).
                  </div>
                </div>
              )}
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
