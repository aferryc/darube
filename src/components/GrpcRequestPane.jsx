import { useEffect, useMemo, useState } from 'react';
import { MonacoCodeEditor } from './MonacoCodeEditor';

function normalizeKV(rows) {
  const r = Array.isArray(rows) ? rows : [];
  if (r.length === 0) return [{ key: '', value: '', enabled: true }];
  return r.map(x => ({
    key: x?.key ?? '',
    value: x?.value ?? '',
    enabled: x?.enabled !== false,
  }));
}

function parseState(raw) {
  if (!raw) return null;
  try {
    const v = JSON.parse(raw);
    if (v && typeof v === 'object' && (v.service || v.method || v.request)) return v;
    return null;
  } catch {
    return null;
  }
}

function defaultState() {
  return {
    kind: 'grpc',
    service: '',
    method: '',
    request: '{\n  \n}',
    headers: [{ key: '', value: '', enabled: true }],
    auth_mode: 'inherit', // inherit | none | bearer | basic
    bearer_token: '',
    username: '',
    password: '',
    timeout_ms: 30000,
    activeTab: 'request', // request | headers | auth
  };
}

function serializeState(s) {
  return JSON.stringify(s, null, 2);
}

function KvTable({ rows, onChange }) {
  const setRow = (idx, patch) => {
    const next = rows.map((r, i) => (i === idx ? { ...r, ...patch } : r));
    onChange(next);
  };
  const removeRow = (idx) => {
    const next = rows.filter((_, i) => i !== idx);
    onChange(next.length ? next : [{ key: '', value: '', enabled: true }]);
  };
  const addRow = () => onChange([...rows, { key: '', value: '', enabled: true }]);

  return (
    <div>
      <table className="api-kv-table">
        <thead>
          <tr>
            <th style={{ width: 46 }}>On</th>
            <th style={{ width: '35%' }}>Key</th>
            <th>Value</th>
            <th style={{ width: 70 }} />
          </tr>
        </thead>
        <tbody>
          {rows.map((r, idx) => (
            <tr key={idx}>
              <td>
                <input
                  type="checkbox"
                  checked={!!r.enabled}
                  onChange={(e) => setRow(idx, { enabled: e.target.checked })}
                />
              </td>
              <td>
                <input
                  className="api-kv-input"
                  value={r.key}
                  onChange={(e) => setRow(idx, { key: e.target.value })}
                  placeholder="x-header"
                />
              </td>
              <td>
                <input
                  className="api-kv-input"
                  value={r.value}
                  onChange={(e) => setRow(idx, { value: e.target.value })}
                  placeholder="Value"
                />
              </td>
              <td>
                <button type="button" className="secondary" onClick={() => removeRow(idx)} style={{ height: 30, padding: '0 10px' }}>
                  ✕
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      <div style={{ marginTop: 10 }}>
        <button type="button" className="secondary" onClick={addRow} style={{ height: 30, padding: '0 12px' }}>
          + Add
        </button>
      </div>
    </div>
  );
}

export function GrpcRequestPane({ apiUrl, connectionId, activeTab, updateActiveTab, loading, setLoading }) {
  const parsed = useMemo(() => parseState(activeTab.query), [activeTab.id]);
  const [state, setState] = useState(() => {
    const base = parsed || defaultState();
    return {
      ...defaultState(),
      ...base,
      headers: normalizeKV(base?.headers),
    };
  });

  const [services, setServices] = useState([]);
  const [methods, setMethods] = useState([]);
  const [reflectError, setReflectError] = useState('');

  useEffect(() => {
    const base = parsed || defaultState();
    setState({
      ...defaultState(),
      ...base,
      headers: normalizeKV(base?.headers),
    });
    setServices([]);
    setMethods([]);
    setReflectError('');
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeTab.id]);

  const save = (next) => {
    setState(next);
    updateActiveTab({ query: serializeState(next) });
  };

  const reflect = async () => {
    if (!connectionId) return;
    try {
      setReflectError('');
      const res = await fetch(`${apiUrl}/api/grpc/${connectionId}/reflect`, { method: 'POST' });
      const data = await res.json();
      if (data.success) setServices(data.services || []);
      else throw new Error(data.error || 'Reflection failed');
    } catch (err) {
      setServices([]);
      setReflectError(err.message);
    }
  };

  useEffect(() => {
    if (!connectionId) return;
    reflect();
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [connectionId]);

  const fetchMethods = async () => {
    if (!connectionId || !state.service) return;
    try {
      const res = await fetch(`${apiUrl}/api/grpc/${connectionId}/methods`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ service: state.service }),
      });
      const data = await res.json();
      if (data.success) setMethods(data.methods || []);
      else setMethods([]);
    } catch {
      setMethods([]);
    }
  };

  useEffect(() => {
    if (!connectionId || !state.service) {
      setMethods([]);
      return;
    }
    fetchMethods();
  }, [connectionId, state.service]);

  const fetchSampleRequest = async () => {
    if (!connectionId || !state.service || !state.method) return;
    setLoading(true);
    try {
      const res = await fetch(`${apiUrl}/api/grpc/${connectionId}/sample-request`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ service: state.service, method: state.method }),
      });
      const data = await res.json();
      if (data.success && data.sample != null) {
        save({ ...state, request: data.sample });
      } else {
        alert(data.error || 'Failed to generate sample request');
      }
    } catch (err) {
      alert('Error: ' + err.message);
    } finally {
      setLoading(false);
    }
  };

  const invoke = async () => {
    if (!connectionId) return;
    setLoading(true);
    try {
      const payload = {
        service: state.service,
        method: state.method,
        request: state.request,
        headers: state.headers,
        timeout_ms: state.timeout_ms,
      };
      if (state.auth_mode && state.auth_mode !== 'inherit') {
        payload.auth = {
          type: state.auth_mode,
          bearer_token: state.bearer_token,
          username: state.username,
          password: state.password,
        };
      }
      const res = await fetch(`${apiUrl}/api/grpc/${connectionId}/invoke`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      const data = await res.json();
      updateActiveTab({ results: { kind: 'grpc', ...data } });
    } catch (err) {
      updateActiveTab({ results: { kind: 'grpc', success: false, error: err.message } });
    } finally {
      setLoading(false);
    }
  };

  const response = activeTab.results?.kind === 'grpc' ? activeTab.results : null;
  const activeSub = state.activeTab || 'request';
  const responseText = response?.success ? (response.response || '') : '';

  return (
    <div className="api-pane">
      <div className="api-request-bar">
        <select
          className="api-method"
          value={state.service}
          onChange={(e) => save({ ...state, service: e.target.value, method: '' })}
          disabled={loading}
        >
          <option value="">(service)</option>
          {services.map(s => <option key={s} value={s}>{s}</option>)}
        </select>
        {methods.length > 0 ? (
          <>
            <input
              className="api-url"
              list="grpc-method-datalist"
              value={state.method}
              onChange={(e) => save({ ...state, method: e.target.value })}
              placeholder="Method (unary) e.g. GetUser"
              disabled={loading}
              autoComplete="off"
            />
            <datalist id="grpc-method-datalist">
              {methods.map(m => <option key={m} value={m} />)}
            </datalist>
          </>
        ) : (
          <input
            className="api-url"
            value={state.method}
            onChange={(e) => save({ ...state, method: e.target.value })}
            placeholder={state.service ? "Loading methods…" : "Select service first"}
            disabled={loading || !state.service}
            autoComplete="off"
          />
        )}
        <button className="secondary" onClick={reflect} disabled={!connectionId || loading} style={{ height: 34, padding: '0 12px', borderRadius: 10 }}>
          Refresh
        </button>
        <button className="api-send" onClick={invoke} disabled={loading || !connectionId || !state.service || !state.method}>
          {loading ? 'Sending…' : 'Send'}
        </button>
      </div>

      <div className="api-subtabs">
        {[
          ['request','Request'],
          ['headers','Headers'],
          ['auth','Auth'],
        ].map(([k, label]) => (
          <div
            key={k}
            className={`api-subtab ${activeSub === k ? 'active' : ''}`}
            onClick={() => save({ ...state, activeTab: k })}
          >
            {label}
          </div>
        ))}
      </div>

      <div className="api-body">
        <div className="api-section">
          {reflectError && (
            <div className="api-error" style={{ marginBottom: 10 }}>
              Reflection error: {reflectError}
            </div>
          )}

          {activeSub === 'headers' && (
            <KvTable rows={state.headers} onChange={(rows) => save({ ...state, headers: normalizeKV(rows) })} />
          )}

          {activeSub === 'auth' && (
            <div style={{ maxWidth: 560 }}>
              <div className="form-group">
                <label>Auth Mode</label>
                <select
                  className="api-method"
                  style={{ width: '100%' }}
                  value={state.auth_mode}
                  onChange={(e) => save({ ...state, auth_mode: e.target.value })}
                  disabled={loading}
                >
                  <option value="inherit">Inherit connection</option>
                  <option value="none">None</option>
                  <option value="bearer">Bearer Token</option>
                  <option value="basic">Basic</option>
                </select>
              </div>
              {state.auth_mode === 'bearer' && (
                <div className="form-group">
                  <label>Bearer Token</label>
                  <input className="api-url" style={{ width: '100%' }} value={state.bearer_token} onChange={(e) => save({ ...state, bearer_token: e.target.value })} />
                </div>
              )}
              {state.auth_mode === 'basic' && (
                <div className="form-row">
                  <div className="form-group">
                    <label>Username</label>
                    <input className="api-url" style={{ width: '100%' }} value={state.username} onChange={(e) => save({ ...state, username: e.target.value })} />
                  </div>
                  <div className="form-group">
                    <label>Password</label>
                    <input className="api-url" style={{ width: '100%' }} type="password" value={state.password} onChange={(e) => save({ ...state, password: e.target.value })} />
                  </div>
                </div>
              )}
            </div>
          )}

          {activeSub === 'request' && (
            <div style={{ display: 'flex', flexDirection: 'column', flex: 1, minHeight: 280 }}>
              <div style={{ display: 'flex', gap: 8, marginBottom: 10, alignItems: 'center' }}>
                <button
                  type="button"
                  className="secondary"
                  onClick={fetchSampleRequest}
                  disabled={loading || !connectionId || !state.service || !state.method}
                  style={{ height: 30, padding: '0 12px', borderRadius: 8 }}
                >
                  Sample request
                </button>
                <span className="form-help-text" style={{ margin: 0 }}>
                  Generates a sample JSON body from the selected method via reflection.
                </span>
              </div>
              <div style={{ flex: 1, minHeight: 240 }}>
                <MonacoCodeEditor
                  value={state.request}
                  onChange={(text) => save({ ...state, request: text })}
                  language="json"
                  editorRole="grpc-request"
                  className="query-editor-container"
                  style={{ height: '100%' }}
                  placeholder="{\n  \n}"
                />
              </div>
            </div>
          )}
        </div>

        <div className="api-response">
          <div className="api-response-header">
            <strong>Response</strong>
            {response && (
              <>
                <span className="api-pill">{response.success ? 'OK' : 'Error'}</span>
                {response.duration_ms !== undefined && <span className="api-pill">{Math.round(response.duration_ms)} ms</span>}
              </>
            )}
          </div>
          <div className="api-response-body">
            {!response && <span style={{ color: 'var(--text-muted)' }}>Send a request to see results.</span>}
            {response && !response.success && <span className="api-error">{response.error || 'Request failed'}</span>}
            {response && response.success && (responseText || '')}
          </div>
        </div>
      </div>
    </div>
  );
}

