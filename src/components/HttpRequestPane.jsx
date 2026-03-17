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
    if (v && typeof v === 'object' && (v.url || v.method || v.body)) return v;
    return null;
  } catch {
    return null;
  }
}

function defaultState() {
  return {
    kind: 'http',
    method: 'GET',
    url: '',
    query_params: [{ key: '', value: '', enabled: true }],
    headers: [{ key: '', value: '', enabled: true }],
    body: { type: 'none', text: '' },
    auth_mode: 'inherit', // inherit | none | bearer | basic
    bearer_token: '',
    username: '',
    password: '',
    timeout_ms: 30000,
    activeTab: 'params', // params | headers | auth | body
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
                  placeholder="Header"
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

export function HttpRequestPane({ apiUrl, connectionId, activeTab, updateActiveTab, loading, setLoading }) {
  const parsed = useMemo(() => parseState(activeTab.query), [activeTab.id]); // only reset on tab change
  const [state, setState] = useState(() => {
    const base = parsed || defaultState();
    return {
      ...defaultState(),
      ...base,
      query_params: normalizeKV(base?.query_params),
      headers: normalizeKV(base?.headers),
      body: base?.body && typeof base.body === 'object' ? { type: base.body.type || 'none', text: base.body.text || '' } : { type: 'none', text: '' },
    };
  });

  useEffect(() => {
    const base = parsed || defaultState();
    setState({
      ...defaultState(),
      ...base,
      query_params: normalizeKV(base?.query_params),
      headers: normalizeKV(base?.headers),
      body: base?.body && typeof base.body === 'object' ? { type: base.body.type || 'none', text: base.body.text || '' } : { type: 'none', text: '' },
    });
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeTab.id]);

  const save = (next) => {
    setState(next);
    updateActiveTab({ query: serializeState(next) });
  };

  const send = async () => {
    if (!connectionId) return;
    setLoading(true);
    try {
      const payload = {
        method: state.method,
        url: state.url,
        query_params: state.query_params,
        headers: state.headers,
        body: state.body,
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

      const res = await fetch(`${apiUrl}/api/http/${connectionId}/request`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      const data = await res.json();
      updateActiveTab({ results: { kind: 'http', ...data } });
    } catch (err) {
      updateActiveTab({ results: { kind: 'http', success: false, error: err.message } });
    } finally {
      setLoading(false);
    }
  };

  const response = activeTab.results?.kind === 'http' ? activeTab.results : null;
  const activeSub = state.activeTab || 'params';

  const responseText = useMemo(() => {
    const txt = response?.body_text ?? '';
    if (!txt) return '';
    try {
      const obj = JSON.parse(txt);
      return JSON.stringify(obj, null, 2);
    } catch {
      return txt;
    }
  }, [response?.body_text]);

  return (
    <div className="api-pane">
      <div className="api-request-bar">
        <select
          className="api-method"
          value={state.method}
          onChange={(e) => save({ ...state, method: e.target.value })}
          disabled={loading}
        >
          {['GET','POST','PUT','PATCH','DELETE','HEAD','OPTIONS'].map(m => <option key={m} value={m}>{m}</option>)}
        </select>
        <input
          className="api-url"
          value={state.url}
          onChange={(e) => save({ ...state, url: e.target.value })}
          placeholder="(optional) /path or https://example.com/path"
          disabled={loading}
        />
        <button className="api-send" onClick={send} disabled={loading || !connectionId}>
          {loading ? 'Sending…' : 'Send'}
        </button>
      </div>

      <div className="api-subtabs">
        {[
          ['params','Params'],
          ['headers','Headers'],
          ['auth','Auth'],
          ['body','Body'],
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
          {activeSub === 'params' && (
            <KvTable rows={state.query_params} onChange={(rows) => save({ ...state, query_params: normalizeKV(rows) })} />
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

          {activeSub === 'body' && (
            <div style={{ display: 'flex', flexDirection: 'column', flex: 1, minHeight: 0 }}>
              <div style={{ display: 'flex', gap: 10, alignItems: 'center' }}>
                <span className="api-pill">Body</span>
                <select
                  className="api-method"
                  value={state.body.type}
                  onChange={(e) => save({ ...state, body: { ...state.body, type: e.target.value } })}
                  disabled={loading}
                >
                  <option value="none">None</option>
                  <option value="json">JSON</option>
                  <option value="raw">Raw</option>
                </select>
              </div>
              {state.body.type !== 'none' && (
                <div style={{ flex: 1, minHeight: 240 }}>
                  <MonacoCodeEditor
                    value={state.body.text}
                    onChange={(text) => save({ ...state, body: { ...state.body, text } })}
                    language={state.body.type === 'json' ? 'json' : 'plaintext'}
                    editorRole="http-body"
                    className="query-editor-container"
                    style={{ height: '100%' }}
                    placeholder={state.body.type === 'json' ? '{\n  \n}' : ''}
                  />
                </div>
              )}
            </div>
          )}
        </div>

        <div className="api-response">
          <div className="api-response-header">
            <strong>Response</strong>
            {response && (
              <>
                <span className="api-pill">{response.success ? response.status : 'Error'}</span>
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

