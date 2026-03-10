import { useMemo, useState } from 'react';

import iconQuestion from '../assets/question.svg';

import { ScriptAutocomplete } from './ScriptAutocomplete';
import { ScriptHelpModal } from './ScriptHelpModal';

export function ScriptPane({ apiUrl, activeTab, updateActiveTab, loading, setLoading, connections }) {
  const output = activeTab.results;
  const [showHelp, setShowHelp] = useState(false);

  const pretty = useMemo(() => {
    if (!output) return '';
    if (output.success === false) return output.error || 'Unknown error';
    try {
      return JSON.stringify(output.result, null, 2);
    } catch {
      return String(output.result);
    }
  }, [output]);

  const logsText = useMemo(() => {
    if (!output?.logs?.length) return '';
    return output.logs.join('\n');
  }, [output]);

  const run = async () => {
    const script = activeTab.query || '';
    if (!script.trim()) return;
    setLoading(true);
    const t0 = performance.now();
    try {
      const res = await fetch(`${apiUrl}/api/scripts/run`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ script, timeout_ms: 15000 }),
      });
      const text = await res.text();
      let data;
      try {
        data = JSON.parse(text);
      } catch (e) {
        throw new Error(text?.slice(0, 300) || e.message);
      }
      updateActiveTab({
        results: {
          ...data,
          durationMs: data.duration_ms ?? (performance.now() - t0),
        },
      });
    } catch (err) {
      updateActiveTab({
        results: { success: false, error: err.message, durationMs: performance.now() - t0 },
      });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="pane script-pane">
      <div className="script-header">
        <div className="script-title">Darube Scripting</div>
        <div style={{ flex: 1 }} />
        <button
          type="button"
          className="btn-icon script-help-btn"
          onClick={() => setShowHelp(true)}
          title="Scripting help"
        >
          <img src={iconQuestion} className="icon-sm icon-light" alt="Help" />
        </button>
        <button type="button" className="script-run-btn" disabled={loading} onClick={run} title="Run script">
          {loading ? 'Running...' : 'Run Script'}
        </button>
      </div>

      <div className="script-split">
        <div className="script-editor">
          <ScriptAutocomplete
            value={activeTab.query}
            onChange={(code) => updateActiveTab({ query: code })}
            placeholder={'// Example:\n// const pg = db.conn(\"prod-postgres\")\n// const users = pg.query(\"SELECT id FROM users\")\n'}
            connections={connections}
          />
        </div>

        <div className="script-output">
          {!output && <div className="empty-state">Run a script to see output</div>}
          {output?.success === false && (
            <div className="execution-error">
              <h4>Script Error</h4>
              <p className="error-message">{pretty}</p>
            </div>
          )}
          {output?.success && (
            <>
              {logsText && (
                <pre className="script-output-pre" style={{ marginBottom: '12px' }}>{logsText}</pre>
              )}
              <pre className="script-output-pre">{pretty || '(no return value)'}</pre>
            </>
          )}
        </div>
      </div>

      <ScriptHelpModal show={showHelp} onClose={() => setShowHelp(false)} />
    </div>
  );
}
