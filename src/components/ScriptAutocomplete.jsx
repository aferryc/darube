import { useCallback, useEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import Editor from 'react-simple-code-editor';
import Prism from 'prismjs';
import 'prismjs/components/prism-javascript';

import { getTextareaCaretViewportPosition } from '../utils/textareaCaret';

const SUGGESTIONS = [
  { label: 'db.conn', insert: 'db.conn("', tail: '")', kind: 'api', desc: 'Get a connection by ID (or name)' },

  { label: 'sleep', insert: 'sleep(', tail: ')', kind: 'api', desc: 'Pause script (ms)' },
  { label: 'utils.sleep', insert: 'utils.sleep(', tail: ')', kind: 'api', desc: 'Pause script (ms)' },
  { label: 'utils.uuidv7', insert: 'utils.uuidv7()', tail: '', kind: 'api', desc: 'Generate UUIDv7' },
  { label: 'utils.now', insert: 'utils.now()', tail: '', kind: 'api', desc: 'Time now (RFC3339)' },
  { label: 'utils.nowUnixMs', insert: 'utils.nowUnixMs()', tail: '', kind: 'api', desc: 'Time now (unix ms)' },

  { label: '.query', insert: 'query("', tail: '")', kind: 'sql', desc: 'Run a query, returns array of objects' },
  { label: '.exec', insert: 'exec("', tail: '")', kind: 'sql', desc: 'Execute SQL, returns rows affected' },
  { label: '.one', insert: 'one("', tail: '")', kind: 'sql', desc: 'Run a query, expect exactly one row' },
  { label: '.scalar', insert: 'scalar("', tail: '")', kind: 'sql', desc: 'Return first column of first row' },

  { label: '.set', insert: 'set("', tail: '", "")', kind: 'redis', desc: 'SET key value (redis)' },
  { label: '.get', insert: 'get("', tail: '")', kind: 'redis', desc: 'GET key (redis)' },
  { label: '.del', insert: 'del("', tail: '")', kind: 'redis', desc: 'DEL key (redis)' },

  { label: 'const', insert: 'const ', tail: '', kind: 'kw', desc: 'Declare variable' },
  { label: 'for (const ... of ...)', insert: 'for (const item of items) {\n  \n}', tail: '', kind: 'kw', desc: 'Iterate array' },
];

function getTokenContext(text, cursor) {
  const before = text.slice(0, cursor);

  // Capture "foo.barBaz" or "barBaz" token right before cursor.
  const m = before.match(/([A-Za-z0-9_.$]+)$/);
  const token = m ? m[1] : '';
  const tokenStart = m ? cursor - token.length : cursor;

  const dotIdx = token.lastIndexOf('.');
  const hasDot = dotIdx !== -1;
  const prefix = hasDot ? token.slice(0, dotIdx) : '';
  const fragment = hasDot ? token.slice(dotIdx + 1) : token;

  return { token, tokenStart, prefix, fragment, hasDot };
}

export function ScriptAutocomplete({ value, onChange, disabled, placeholder, style, connections }) {
  const [open, setOpen] = useState(false);
  const [suggestions, setSuggestions] = useState([]);
  const [selectedIdx, setSelectedIdx] = useState(0);
  const [dropdownPos, setDropdownPos] = useState(null);

  const containerRef = useRef(null);
  const connectionsRef = useRef([]);
  const valueRef = useRef(value);
  useEffect(() => { valueRef.current = value; }, [value]);
  useEffect(() => { connectionsRef.current = connections || []; }, [connections]);

  const buildSuggestions = useCallback((text, cursor) => {
    // Context: db.conn(<here>)
    const before = text.slice(0, cursor);
    const connCall = before.match(/db\s*\.\s*conn\s*\(\s*(["']?)([^"']*)$/);
    if (connCall) {
      const frag = (connCall[2] || '').toLowerCase();
      return (connectionsRef.current || [])
        .map(c => ({
          label: `${c.connection_name} (${c.db_type})`,
          insert: c.connection_name,
          tail: '',
          kind: 'api',
          desc: c.id,
          _ctx: 'dbconn',
        }))
        .filter(s => s.insert.toLowerCase().includes(frag))
        .slice(0, 12);
    }

    const { prefix, fragment, hasDot } = getTokenContext(text, cursor);
    const frag = (fragment || '').toLowerCase();

    // If user typed "db." then suggest conn
    if (hasDot && prefix === 'db') {
      return SUGGESTIONS.filter(s => s.label === 'db.conn');
    }

    // If user typed "utils." then suggest utilities
    if (hasDot && prefix === 'utils') {
      return SUGGESTIONS
        .filter(s => s.label.startsWith('utils.') && s.label.slice('utils.'.length).toLowerCase().startsWith(frag))
        .slice(0, 10);
    }

    // If user typed "<var>." suggest connection methods
    if (hasDot) {
      return SUGGESTIONS.filter(s => s.label.startsWith('.') && s.label.slice(1).toLowerCase().startsWith(frag)).slice(0, 10);
    }

    // Otherwise suggest keywords and db.conn
    return SUGGESTIONS
      .filter(s => !s.label.startsWith('.') && s.label.toLowerCase().includes(frag))
      .slice(0, 10);
  }, []);

  const refresh = useCallback((e) => {
    const ignored = ['ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight', 'Enter', 'Tab', 'Escape', 'Shift', 'Control', 'Alt', 'Meta'];
    if (ignored.includes(e.key)) return;
    const ta = e.target;
    const cursor = ta.selectionStart ?? 0;
    const text = valueRef.current || '';
    const list = buildSuggestions(text, cursor);
    if (!list.length) {
      setOpen(false);
      setDropdownPos(null);
      return;
    }

    setSuggestions(list);
    setSelectedIdx(0);
    setOpen(true);

    const caret = getTextareaCaretViewportPosition(ta, cursor);
    if (caret) {
      const desiredTop = caret.top + caret.height + 6;
      const desiredLeft = caret.left;
      const maxLeft = Math.max(8, window.innerWidth - 460);
      const maxTop = Math.max(8, window.innerHeight - 320);
      setDropdownPos({
        top: Math.min(desiredTop, maxTop),
        left: Math.min(desiredLeft, maxLeft),
      });
    }
  }, [buildSuggestions]);

  const insertSuggestion = useCallback((s) => {
    const ta = containerRef.current?.querySelector('textarea');
    if (!ta) return;

    const cursor = ta.selectionStart ?? 0;
    const text = valueRef.current || '';

    // Insert connection name inside db.conn(...) argument.
    if (s._ctx === 'dbconn') {
      const before = text.slice(0, cursor);
      const m = before.match(/db\s*\.\s*conn\s*\(\s*(["']?)([^"']*)$/);
      if (m) {
        const quote = m[1] || '';
        const frag = m[2] || '';
        const fragStart = cursor - frag.length;
        const insert = quote ? s.insert : `"${s.insert}"`;
        const next = text.slice(0, fragStart) + insert + text.slice(cursor);
        onChange(next);
        setOpen(false);
        setDropdownPos(null);
        setTimeout(() => {
          const newPos = fragStart + insert.length;
          ta.setSelectionRange(newPos, newPos);
          ta.focus();
        }, 0);
        return;
      }
    }

    const { tokenStart, hasDot, prefix } = getTokenContext(text, cursor);

    let insert = s.insert;
    // For dot-suggestions, preserve "<var>." prefix and insert method name.
    if (s.label.startsWith('.') && hasDot && prefix) {
      insert = `${prefix}.${s.insert}`;
    }

    const next = text.slice(0, tokenStart) + insert + s.tail + text.slice(cursor);
    onChange(next);

    setOpen(false);
    setDropdownPos(null);

    setTimeout(() => {
      const newPos = tokenStart + insert.length;
      ta.setSelectionRange(newPos, newPos);
      ta.focus();
    }, 0);
  }, [onChange]);

  const onKeyDown = useCallback((e) => {
    if (open && suggestions.length) {
      if (e.key === 'ArrowDown') { e.preventDefault(); setSelectedIdx(i => (i + 1) % suggestions.length); return; }
      if (e.key === 'ArrowUp') { e.preventDefault(); setSelectedIdx(i => (i - 1 + suggestions.length) % suggestions.length); return; }
      if (e.key === 'Tab' || (e.key === 'Enter' && !e.metaKey && !e.ctrlKey)) {
        e.preventDefault();
        insertSuggestion(suggestions[selectedIdx]);
        return;
      }
      if (e.key === 'Escape') { setOpen(false); setDropdownPos(null); return; }
    }
  }, [open, suggestions, selectedIdx, insertSuggestion]);

  // Attach keyup listener directly to textarea (same approach as SqlAutocomplete).
  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;
    const attach = () => {
      const ta = container.querySelector('textarea');
      if (ta) {
        ta.addEventListener('keyup', refresh);
        return ta;
      }
      return null;
    };
    let ta = attach();
    if (!ta) {
      const obs = new MutationObserver(() => {
        ta = attach();
        if (ta) obs.disconnect();
      });
      obs.observe(container, { childList: true, subtree: true });
      return () => obs.disconnect();
    }
    return () => { if (ta) ta.removeEventListener('keyup', refresh); };
  }, [refresh]);

  // Close on click outside
  useEffect(() => {
    const close = (e) => {
      // Dropdown is rendered in a portal; clicking it should not be treated as an outside click.
      if (e?.target?.closest?.('.script-ac-dropdown')) return;
      if (containerRef.current && !containerRef.current.contains(e.target)) {
        setOpen(false);
        setDropdownPos(null);
      }
    };
    document.addEventListener('mousedown', close);
    return () => document.removeEventListener('mousedown', close);
  }, []);

  const kindBadge = (kind) => {
    if (kind === 'api') return <span className="ac-badge ac-badge-kw">api</span>;
    if (kind === 'sql') return <span className="ac-badge ac-badge-tbl">sql</span>;
    if (kind === 'redis') return <span className="ac-badge ac-badge-col">redis</span>;
    return <span className="ac-badge ac-badge-kw">js</span>;
  };

  return (
    <div ref={containerRef} style={{ position: 'relative', display: 'flex', flexDirection: 'column', flex: 1, width: '100%' }}>
      <Editor
        value={value}
        onValueChange={onChange}
        highlight={(code) => Prism.highlight(code, Prism.languages.javascript, 'javascript')}
        padding={16}
        className="query-editor-container"
        textareaClassName="query-editor-textarea"
        onKeyDown={onKeyDown}
        disabled={disabled}
        placeholder={placeholder}
        style={style}
      />

      {open && dropdownPos && createPortal(
        <ul
          className="ac-dropdown script-ac-dropdown"
          style={{ position: 'fixed', top: `${dropdownPos.top}px`, left: `${dropdownPos.left}px` }}
          onMouseDown={(e) => e.preventDefault()}
        >
          {suggestions.map((s, i) => (
            <li
              key={s.label + i}
              className={`ac-item ${i === selectedIdx ? 'ac-selected' : ''}`}
              onMouseEnter={() => setSelectedIdx(i)}
              onClick={() => insertSuggestion(s)}
            >
              {kindBadge(s.kind)}
              <span className="ac-label">{s.label}</span>
              {s.desc && <span className="ac-sub">{s.desc}</span>}
            </li>
          ))}
        </ul>,
        document.body
      )}
    </div>
  );
}
