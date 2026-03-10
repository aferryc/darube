import React, { useMemo, useState } from 'react';

function quoteRedisArg(s) {
  const str = String(s ?? '');
  // Quote only when needed (spaces/newlines/quotes). Keep simple and robust.
  if (!/[\\s"\\n\\r\\t]/.test(str)) return str;
  return '"' + str
    .replaceAll('\\', '\\\\')
    .replaceAll('"', '\\"')
    .replaceAll('\n', '\\n')
    .replaceAll('\r', '\\r')
    .replaceAll('\t', '\\t') + '"';
}

function parseRedisCommand(cmd) {
  const parts = String(cmd || '').trim().split(/\\s+/).filter(Boolean);
  return { name: (parts[0] || '').toUpperCase(), args: parts.slice(1) };
}

export function RedisPane({ activeTab, loading, connectionId, onQuery, onExport }) {
  const result = activeTab.results?.data;
  const lastCmd = activeTab.lastExecutedQuery || activeTab.query || '';
  const parsedCmd = useMemo(() => parseRedisCommand(lastCmd), [lastCmd]);
  const [editing, setEditing] = useState(null); // { kind: 'get'|'hget'|'hash', key, field?, draft }

  const tryParseJson = (s) => {
    if (typeof s !== 'string') return null;
    const t = s.trim();
    if (!(t.startsWith('{') || t.startsWith('['))) return null;
    try { return JSON.parse(t); } catch { return null; }
  };
  
  const renderValue = (val, type) => {
    if (type === 'nil' || val === null || val === undefined) {
      return <div className="redis-nil">(nil)</div>;
    }
    // Allow editing for GET/HGET scalar results.
    const canEditScalar = !loading && !!onQuery && !!connectionId &&
      ((parsedCmd.name === 'GET' && parsedCmd.args.length >= 1) || (parsedCmd.name === 'HGET' && parsedCmd.args.length >= 2));
    const onDbl = canEditScalar ? () => {
      if (parsedCmd.name === 'GET') {
        const key = parsedCmd.args[0];
        setEditing({ kind: 'get', key, draft: typeof val === 'string' ? val : JSON.stringify(val, null, 2) });
      } else if (parsedCmd.name === 'HGET') {
        const [key, field] = parsedCmd.args;
        setEditing({ kind: 'hget', key, field, draft: typeof val === 'string' ? val : JSON.stringify(val, null, 2) });
      }
    } : null;
    if (type === 'json' || typeof val === 'object') {
      return <pre className="redis-json-view redis-editable" onDoubleClick={onDbl}>{JSON.stringify(val, null, 2)}</pre>;
    }
    return <div className="redis-raw-value redis-editable" onDoubleClick={onDbl}>{String(val)}</div>;
  };

  const renderHash = (val) => {
    if (!val || typeof val !== 'object' || Array.isArray(val)) {
      return renderValue(val, 'json');
    }
    const entries = Object.entries(val);
    if (entries.length === 0) return <div className="redis-nil">(empty)</div>;

    const allowHashEdit = parsedCmd.name === 'HGETALL' && parsedCmd.args.length >= 1 && !loading && !!onQuery && !!connectionId;
    const hashKey = allowHashEdit ? parsedCmd.args[0] : null;

    return (
      <div className="results-table-wrapper redis-kv-wrapper">
        <table className="results-table redis-kv-table">
          <thead>
            <tr>
              <th style={{ width: '260px' }}>Field</th>
              <th>Value</th>
            </tr>
          </thead>
          <tbody>
            {entries.map(([k, v]) => {
              const parsed = tryParseJson(v);
              const isEditing = editing?.kind === 'hash' && editing?.field === k;
              return (
                <tr key={k}>
                  <td className="redis-kv-key">{k}</td>
                  <td className="redis-kv-value">
                    {isEditing ? (
                      <textarea
                        className="redis-inline-editor"
                        value={editing.draft}
                        autoFocus
                        onChange={(e) => setEditing(prev => ({ ...prev, draft: e.target.value }))}
                        onKeyDown={async (e) => {
                          if (e.key === 'Escape') { e.preventDefault(); setEditing(null); return; }
                          if (e.key === 'Enter' && !e.shiftKey) {
                            e.preventDefault();
                            try {
                              const cmd = `HSET ${quoteRedisArg(hashKey)} ${quoteRedisArg(k)} ${quoteRedisArg(editing.draft)}`;
                              await onQuery(cmd, connectionId, 'redis');
                              await onQuery(`HGETALL ${quoteRedisArg(hashKey)}`, connectionId, 'redis');
                            } finally {
                              setEditing(null);
                            }
                          }
                        }}
                      />
                    ) : (
                      <div
                        className="redis-editable"
                        title={allowHashEdit ? 'Double-click to edit (Enter to save, Shift+Enter for newline)' : ''}
                        onDoubleClick={() => {
                          if (!allowHashEdit) return;
                          setEditing({
                            kind: 'hash',
                            key: hashKey,
                            field: k,
                            draft: typeof v === 'string' ? v : (parsed !== null ? JSON.stringify(parsed) : String(v)),
                          });
                        }}
                      >
                        {parsed !== null ? (
                          <pre className="redis-json-view">{JSON.stringify(parsed, null, 2)}</pre>
                        ) : (
                          <div className="redis-raw-value">{String(v)}</div>
                        )}
                      </div>
                    )}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    );
  };

  const renderInlineEditor = () => {
    if (!editing || (editing.kind !== 'get' && editing.kind !== 'hget')) return null;
    const title = editing.kind === 'get'
      ? <>Edit value for <span className="mono">{editing.key}</span></>
      : <>Edit field <span className="mono">{editing.field}</span> in <span className="mono">{editing.key}</span></>;
    return (
      <div className="redis-inline-editor-overlay">
        <div className="redis-inline-editor-card">
          <div className="redis-inline-editor-title">{title}</div>
          <textarea
            className="redis-inline-editor"
            value={editing.draft}
            autoFocus
            onChange={(e) => setEditing(prev => ({ ...prev, draft: e.target.value }))}
            onKeyDown={async (e) => {
              if (e.key === 'Escape') { e.preventDefault(); setEditing(null); return; }
              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                try {
                  if (editing.kind === 'get') {
                    const cmd = `SET ${quoteRedisArg(editing.key)} ${quoteRedisArg(editing.draft)}`;
                    await onQuery?.(cmd, connectionId, 'redis');
                    await onQuery?.(`GET ${quoteRedisArg(editing.key)}`, connectionId, 'redis');
                  } else {
                    const cmd = `HSET ${quoteRedisArg(editing.key)} ${quoteRedisArg(editing.field)} ${quoteRedisArg(editing.draft)}`;
                    await onQuery?.(cmd, connectionId, 'redis');
                    await onQuery?.(`HGET ${quoteRedisArg(editing.key)} ${quoteRedisArg(editing.field)}`, connectionId, 'redis');
                  }
                } finally {
                  setEditing(null);
                }
              }
            }}
          />
          <div className="redis-inline-editor-hint">Enter to save, Shift+Enter newline, Esc cancel</div>
        </div>
      </div>
    );
  };

  return (
    <div className="pane redis-section">
      {activeTab.results?.message && (
        <div className="redis-warning">
          ⚠️ {activeTab.results.message}
        </div>
      )}
      
      {!activeTab.results && !loading && (
        <div className="empty-state">Execute a Redis command to see results</div>
      )}

      {activeTab.results?.success === false && (
        <div className="execution-error">
          <h4>Redis Error</h4>
          <p className="error-message">{activeTab.results.error}</p>
        </div>
      )}

      {result && (
        <div className="redis-results-container">
          <div className="redis-result-meta">
            <span className="redis-type-badge">{result.data_type}</span>
            <div style={{ flex: 1 }} />
            <button
              className="secondary"
              type="button"
              disabled={loading}
              onClick={() => onExport?.(result, lastCmd)}
              title="Export this Redis result"
              style={{ height: '26px', padding: '4px 10px', fontSize: '11px' }}
            >
              Export...
            </button>
          </div>
          <div className="redis-result-content">
            {result.data_type === 'hash'
              ? renderHash(result.value)
              : renderValue(result.value, result.data_type)}
          </div>
        </div>
      )}

      {renderInlineEditor()}
    </div>
  );
}
