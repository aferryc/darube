import { useEffect, useMemo, useState } from 'react';

export function ScriptHelpModal({ show, onClose }) {
  if (!show) return null;
  const [tab, setTab] = useState('overview');
  const [showExample, setShowExample] = useState(false);
  const [copied, setCopied] = useState(false);

  const example = useMemo(() => (
`console.log("hello world")

const pg = db.conn("prod-postgres")
const redis = db.conn("cache")

const users = pg.query("SELECT id FROM users")
for (const u of users) {
  redis.set(\`user:\${u.id}\`, "active")
}

sleep(250)
console.log(utils.uuidv7(), utils.now(), utils.nowUnixMs())
`
  ), []);

  useEffect(() => {
    if (!show) return;
    setShowExample(false);
    setCopied(false);
    setTab(t => (t === 'overview' || t === 'syntax' ? t : 'overview'));
  }, [show]);

  useEffect(() => {
    if (!copied) return;
    const t = setTimeout(() => setCopied(false), 900);
    return () => clearTimeout(t);
  }, [copied]);

  const copyExample = async () => {
    try {
      await navigator.clipboard.writeText(example);
      setCopied(true);
    } catch {
      // Ignore clipboard failures (permissions, older Electron builds).
    }
  };

  return (
    <div className="modal-overlay">
      <div className="modal-content w-500">
        <div className="modal-header" style={{ marginBottom: '12px' }}>
          <div className="modal-header-left">
            <h3>Scripting Help</h3>
          </div>
          <div className="modal-tabs" aria-label="Help tabs">
            <button
              type="button"
              className={`modal-tab ${tab === 'overview' ? 'active' : ''}`}
              onClick={() => setTab('overview')}
            >
              Overview
            </button>
            <button
              type="button"
              className={`modal-tab ${tab === 'syntax' ? 'active' : ''}`}
              onClick={() => setTab('syntax')}
            >
              Syntax
            </button>
          </div>
        </div>

        <div className="help-content">
          {tab === 'overview' && (
            <>
              <p>Darube scripts run JavaScript (goja). You have <code>db</code>, <code>console</code>, and a small set of <code>utils</code>.</p>

              <p style={{ marginTop: '12px' }}><strong>Basics</strong></p>
              <ul className="help-list">
                <li><code>db.conn(id)</code> returns a connection object.</li>
                <li>Connections are resolved by saved connection ID (and also by connection name if unique).</li>
                <li>Scripts are synchronous (no <code>await</code>).</li>
              </ul>
            </>
          )}

          {tab === 'syntax' && (
            <>
              <div className="script-help-grid">
                <div className="script-help-card">
                  <div className="script-help-card-title">Connection Methods</div>
                  <div className="script-help-rows">
                    <div className="script-help-row"><code>query(sql)</code><span className="script-help-hint">array of objects</span></div>
                    <div className="script-help-row"><code>exec(sql)</code><span className="script-help-hint">rows affected</span></div>
                    <div className="script-help-row"><code>one(sql)</code><span className="script-help-hint">exactly one row</span></div>
                    <div className="script-help-row"><code>scalar(sql)</code><span className="script-help-hint">primitive</span></div>
                    <div className="script-help-row"><code>set(key, value)</code><span className="script-help-hint">redis</span></div>
                    <div className="script-help-row"><code>get(key)</code><span className="script-help-hint">redis</span></div>
                    <div className="script-help-row"><code>del(key)</code><span className="script-help-hint">redis</span></div>
                  </div>
                </div>

                <div className="script-help-card">
                  <div className="script-help-card-title">Utilities</div>
                  <div className="script-help-rows">
                    <div className="script-help-row"><code>sleep(ms)</code><span className="script-help-hint">pause script</span></div>
                    <div className="script-help-row"><code>utils.sleep(ms)</code><span className="script-help-hint">same</span></div>
                    <div className="script-help-row"><code>utils.uuidv7()</code><span className="script-help-hint">string</span></div>
                    <div className="script-help-row"><code>utils.now()</code><span className="script-help-hint">RFC3339</span></div>
                    <div className="script-help-row"><code>utils.nowUnixMs()</code><span className="script-help-hint">number</span></div>
                  </div>
                </div>
              </div>

              <div className="script-help-example">
                <div className="script-help-example-head">
                  <div className="script-help-card-title" style={{ margin: 0 }}>Example</div>
                  <div className="script-help-example-actions">
                    <button
                      type="button"
                      className="script-help-mini-btn"
                      onClick={() => setShowExample(v => !v)}
                    >
                      {showExample ? 'Collapse' : 'Expand'}
                    </button>
                    <button
                      type="button"
                      className="script-help-mini-btn"
                      onClick={copyExample}
                      disabled={!showExample}
                      title={showExample ? 'Copy example' : 'Expand to enable copy'}
                    >
                      {copied ? 'Copied' : 'Copy'}
                    </button>
                  </div>
                </div>

                {!showExample && (
                  <div className="script-help-example-preview">
                    <code>const pg = db.conn("prod-postgres")</code>
                    <span className="script-help-hint">then run queries across connections</span>
                  </div>
                )}

                {showExample && (
                  <pre className="script-output-pre script-help-pre">
                    {example}
                  </pre>
                )}
              </div>
            </>
          )}
        </div>
        <div className="modal-footer">
          <button type="button" onClick={onClose}>OK</button>
        </div>
      </div>
    </div>
  );
}
