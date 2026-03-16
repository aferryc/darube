import React from 'react';

const NOSQL_TYPES = new Set(['redis', 'mongodb', 'cassandra', 'elasticsearch', 'opensearch']);
const API_TYPES = new Set(['http', 'grpc']);

export function ConnectionModal({
  show, editingId, formData, setFormData, folders,
  onSubmit, onTest, onClose,
}) {
  const [activeTab, setActiveTab] = React.useState('general');
  const [connectionCategory, setConnectionCategory] = React.useState('sql');

  // Sync category with form data if editing
  React.useEffect(() => {
    if (show && editingId) {
      if (API_TYPES.has(formData.db_type)) {
        setConnectionCategory('api');
      } else if (NOSQL_TYPES.has(formData.db_type)) {
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
  const isHttp = formData.db_type === 'http';
  const isGrpc = formData.db_type === 'grpc';

  const onFieldChange = (e) => {
    const { name, type, value, checked } = e.target;
    update(name, type === 'checkbox' ? checked : value);
  };

  const onCategoryClick = (e) => {
    const cat = e.currentTarget?.dataset?.cat;
    if (!cat) return;
    handleCategoryChange(cat);
  };

  const onTabClick = (e) => {
    const tab = e.currentTarget?.dataset?.tab;
    if (!tab) return;
    setActiveTab(tab);
  };

  const onBrowseClick = async (e) => {
    const field = e.currentTarget?.dataset?.field;
    if (!field) return;
    if (window.darube?.openFile) {
      const path = await window.darube.openFile();
      if (path) update(field, path);
    }
  };

  const onSqlDbTypeChange = (e) => {
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
  };

  const onNoSqlDbTypeChange = (e) => {
    const next = e.target.value;
    update('db_type', next);
    const defaults = { redis: 6379, mongodb: 27017, cassandra: 9042, elasticsearch: 9200, opensearch: 9200 };
    update('port', defaults[next] ?? formData.port ?? 0);
    if (next === 'redis') update('dbname', '');
  };

  const handleCategoryChange = (cat) => {
    setConnectionCategory(cat);
    if (cat === 'nosql') {
      if (!NOSQL_TYPES.has(formData.db_type)) {
        update('db_type', 'redis');
        update('port', 6379);
      }
    } else if (cat === 'api') {
      if (!API_TYPES.has(formData.db_type)) {
        update('db_type', 'http');
        update('base_url', '');
        update('address', '');
        update('tls', false);
        update('insecure_tls', false);
        update('server_name', '');
        update('auth_type', 'none');
        update('bearer_token', '');
        update('auth_username', '');
        update('auth_password', '');
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
                data-cat="sql"
                onClick={onCategoryClick}
              >
                SQL
              </span>
              <span 
                className={`category-tab ${connectionCategory === 'nosql' ? 'active' : ''}`}
                data-cat="nosql"
                onClick={onCategoryClick}
              >
                NoSQL
              </span>
              <span
                className={`category-tab ${connectionCategory === 'api' ? 'active' : ''}`}
                data-cat="api"
                onClick={onCategoryClick}
              >
                API
              </span>
            </div>
          </div>
          <div className="modal-tabs">
            {connectionCategory === 'sql' && !isSQLite && (
              <>
                <button 
                  type="button" 
                  className={`modal-tab ${activeTab === 'general' ? 'active' : ''}`}
                  data-tab="general"
                  onClick={onTabClick}
                >
                  General
                </button>
                <button 
                  type="button" 
                  className={`modal-tab ${activeTab === 'ssl' ? 'active' : ''}`}
                  data-tab="ssl"
                  onClick={onTabClick}
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
                  <input name="connection_name" autoFocus required value={formData.connection_name} onChange={onFieldChange} placeholder="Production DB" />
                </div>

                <div className="form-row">
                  <div className="form-group">
                    <label>Database Type</label>
                    <select
                      value={formData.db_type}
                      onChange={onSqlDbTypeChange}
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
                      <select name="folder_id" value={formData.folder_id || ''} onChange={onFieldChange}>
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
                        name="file_path"
                        required
                        value={formData.file_path || ''}
                        onChange={onFieldChange}
                        placeholder="/absolute/path/to/db.sqlite (or :memory:)"
                      />
                      <button
                        type="button"
                        className="secondary browse-btn"
                        data-field="file_path"
                        onClick={onBrowseClick}
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
                      <input name="host" required value={formData.host} onChange={onFieldChange} placeholder="localhost" />
                    </div>
                    <div className="form-group flex-1">
                      <label>Port</label>
                      <input name="port" type="number" required value={formData.port} onChange={onFieldChange} />
                    </div>
                  </div>
                )}

                {!isSQLite && (
                  <div className="form-group">
                    <label>{isOracle ? 'Service Name' : 'Database (Optional)'}</label>
                    <input
                      name="dbname"
                      value={formData.dbname}
                      onChange={onFieldChange}
                      placeholder={isOracle ? 'orclpdb1' : 'postgres'}
                      required={isOracle}
                    />
                  </div>
                )}

                {!isSQLite && (
                  <div className="form-row">
                    <div className="form-group">
                      <label>Username</label>
                      <input name="user" required value={formData.user} onChange={onFieldChange} />
                    </div>
                    <div className="form-group">
                      <label>Password</label>
                      <input name="password" type="password" value={formData.password} onChange={onFieldChange} />
                    </div>
                  </div>
                )}

                {!isSQLite && (
                  <div className="form-group mt-10">
                    <label className="form-checkbox-label">
                      <input
                        name="teleport_enabled"
                        type="checkbox"
                        checked={!!formData.teleport_enabled}
                        onChange={onFieldChange}
                      />
                      Connect via Teleport (tsh)
                    </label>
                    <div className="form-help-text">
                      Requires <code>tsh</code> to be installed and configured on this machine.
                    </div>
                  </div>
                )}

                {formData.teleport_enabled && !isSQLite && (
                  <div className="flex-column gap-12 mt-10">
                    <div className="form-row">
                      <div className="form-group">
                        <label>Teleport Cluster</label>
                        <input
                          name="teleport_cluster"
                          value={formData.teleport_cluster || ''}
                          onChange={onFieldChange}
                          placeholder="example-teleport-cluster"
                        />
                      </div>
                      <div className="form-group">
                        <label>DB Service Name</label>
                        <input
                          name="teleport_db_service"
                          value={formData.teleport_db_service || ''}
                          onChange={onFieldChange}
                          placeholder="postgres-prod"
                        />
                      </div>
                    </div>

                    <div className="form-row">
                      <div className="form-group">
                        <label>Teleport User (Optional)</label>
                        <input
                          name="teleport_user"
                          value={formData.teleport_user || ''}
                          onChange={onFieldChange}
                          placeholder="alice"
                        />
                      </div>
                      <div className="form-group">
                        <label>Teleport Profile (Optional)</label>
                        <input
                          name="teleport_profile"
                          value={formData.teleport_profile || ''}
                          onChange={onFieldChange}
                          placeholder="default"
                        />
                      </div>
                    </div>
                  </div>
                )}
              </>
            ) : (
              <>
                <div className="form-group mb-20">
                  <label className="form-checkbox-label">
                    <input name="enable_ssl" type="checkbox" checked={formData.enable_ssl} onChange={onFieldChange} />
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
                          name="ca_cert_path"
                          value={formData.ca_cert_path || ''}
                          onChange={onFieldChange}
                          placeholder="/absolute/path/to/ca.pem"
                        />
                        <button
                          type="button"
                          className="secondary browse-btn"
                          data-field="ca_cert_path"
                          onClick={onBrowseClick}
                        >
                          Browse...
                        </button>
                      </div>
                    </div>

                    <div className="form-group">
                      <label>Client Certificate</label>
                      <div className="form-file-input-row">
                        <input
                          name="client_cert_path"
                          value={formData.client_cert_path || ''}
                          onChange={onFieldChange}
                          placeholder="/absolute/path/to/client.crt"
                        />
                        <button
                          type="button"
                          className="secondary browse-btn"
                          data-field="client_cert_path"
                          onClick={onBrowseClick}
                        >
                          Browse...
                        </button>
                      </div>
                    </div>

                    <div className="form-group">
                      <label>Client Key</label>
                      <div className="form-file-input-row">
                        <input
                          name="client_key_path"
                          value={formData.client_key_path || ''}
                          onChange={onFieldChange}
                          placeholder="/absolute/path/to/client.key"
                        />
                        <button
                          type="button"
                          className="secondary browse-btn"
                          data-field="client_key_path"
                          onClick={onBrowseClick}
                        >
                          Browse...
                        </button>
                      </div>
                    </div>
                  </div>
                )}
              </>
            )
          ) : connectionCategory === 'api' ? (
            <>
              <div className="form-group">
                <label>Name</label>
                <input name="connection_name" autoFocus required value={formData.connection_name} onChange={onFieldChange} placeholder="Payments API" />
              </div>

              <div className="form-row">
                <div className="form-group">
                  <label>Type</label>
                  <select value={formData.db_type} onChange={(e) => update('db_type', e.target.value)}>
                    <option value="http">HTTP</option>
                    <option value="grpc">gRPC</option>
                  </select>
                </div>
                {folders.length > 0 && (
                  <div className="form-group">
                    <label>Folder (Optional)</label>
                    <select name="folder_id" value={formData.folder_id || ''} onChange={onFieldChange}>
                      <option value="">Uncategorized</option>
                      {folders.map(f => <option key={f.id} value={f.id}>{f.name}</option>)}
                    </select>
                  </div>
                )}
              </div>

              {isHttp && (
                <div className="form-group">
                  <label>Base URL</label>
                  <input name="base_url" required value={formData.base_url || ''} onChange={onFieldChange} placeholder="https://api.example.com" />
                </div>
              )}

              {isGrpc && (
                <>
                  <div className="form-group">
                    <label>Address</label>
                    <input name="address" required value={formData.address || ''} onChange={onFieldChange} placeholder="grpc.example.com:443" />
                  </div>
                  <div className="form-row">
                    <div className="form-group">
                      <label className="form-checkbox-label">
                        <input name="tls" type="checkbox" checked={!!formData.tls} onChange={onFieldChange} />
                        TLS
                      </label>
                    </div>
                    <div className="form-group">
                      <label className="form-checkbox-label">
                        <input name="insecure_tls" type="checkbox" checked={!!formData.insecure_tls} onChange={onFieldChange} />
                        Insecure TLS
                      </label>
                    </div>
                  </div>
                  <div className="form-group">
                    <label>Server Name (Optional)</label>
                    <input name="server_name" value={formData.server_name || ''} onChange={onFieldChange} placeholder="override SNI (rare)" />
                  </div>
                </>
              )}

              <div className="form-row">
                <div className="form-group flex-1">
                  <label>Auth</label>
                  <select name="auth_type" value={formData.auth_type || 'none'} onChange={onFieldChange}>
                    <option value="none">None</option>
                    <option value="bearer">Bearer Token</option>
                    <option value="basic">Basic</option>
                  </select>
                </div>
              </div>

              {formData.auth_type === 'bearer' && (
                <div className="form-group">
                  <label>Bearer Token</label>
                  <input name="bearer_token" value={formData.bearer_token || ''} onChange={onFieldChange} placeholder="eyJhbGciOi..." />
                </div>
              )}

              {formData.auth_type === 'basic' && (
                <div className="form-row">
                  <div className="form-group">
                    <label>Username</label>
                    <input name="auth_username" value={formData.auth_username || ''} onChange={onFieldChange} />
                  </div>
                  <div className="form-group">
                    <label>Password</label>
                    <input name="auth_password" type="password" value={formData.auth_password || ''} onChange={onFieldChange} />
                  </div>
                </div>
              )}
            </>
          ) : (
            <>
              {/* NoSQL / Redis Form */}
              <div className="form-group">
                <label>Connection Name</label>
                <input name="connection_name" autoFocus required value={formData.connection_name} onChange={onFieldChange} placeholder="Redis Production" />
              </div>

              <div className="form-group">
                <label>Database Type</label>
                <select
                  value={formData.db_type}
                  onChange={onNoSqlDbTypeChange}
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
                  <input name="host" required value={formData.host} onChange={onFieldChange} placeholder="localhost" />
                </div>
                <div className="form-group flex-1">
                  <label>Port</label>
                  <input name="port" type="number" value={formData.port || 6379} onChange={onFieldChange} />
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
                    name="dbname"
                    value={formData.dbname || ''}
                    onChange={onFieldChange}
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
                  <input name="user" value={formData.user} onChange={onFieldChange} placeholder="default" />
                </div>
                <div className="form-group">
                  <label>Password (Optional)</label>
                  <input name="password" type="password" value={formData.password} onChange={onFieldChange} />
                </div>
              </div>

              {formData.db_type === 'redis' && (
                <div className="form-group mt-10">
                  <label className="form-checkbox-label">
                    <input name="is_cluster" type="checkbox" checked={formData.is_cluster} onChange={onFieldChange} />
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
