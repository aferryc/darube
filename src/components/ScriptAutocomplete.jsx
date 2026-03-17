/* @refresh reset */
import { useCallback, useEffect, useRef } from 'react';

import { MonacoCodeEditor } from './MonacoCodeEditor';

const SUGGESTIONS = [
  { label: 'db.conn', insert: 'db.conn("', tail: '")', kind: 'api', desc: 'Get a connection by ID (or name)' },
  { label: 'http.request', insert: 'http.request({ method: "GET", url: "', tail: '" })', kind: 'api', desc: 'HTTP request (curl-like)' },
  { label: 'http.conn', insert: 'http.conn("', tail: '")', kind: 'api', desc: 'Get an HTTP connection by ID (or name)' },
  { label: 'grpc.conn', insert: 'grpc.conn("', tail: '")', kind: 'api', desc: 'Get a gRPC connection by ID (or name)' },

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
  const m = before.match(/([A-Za-z0-9_.$]+)$/);
  const token = m ? m[1] : '';
  const tokenStart = m ? cursor - token.length : cursor;

  const dotIdx = token.lastIndexOf('.');
  const hasDot = dotIdx !== -1;
  const prefix = hasDot ? token.slice(0, dotIdx) : '';
  const fragment = hasDot ? token.slice(dotIdx + 1) : token;

  return { token, tokenStart, prefix, fragment, hasDot };
}

export function buildScriptSuggestions(text, cursor, connections) {
  const before = text.slice(0, cursor);

  // Context: db.conn(<here>)
  const connCall = before.match(/db\s*\.\s*conn\s*\(\s*(["']?)([^"']*)$/);
  if (connCall) {
    const frag = (connCall[2] || '').toLowerCase();
    return (connections || [])
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
    return SUGGESTIONS
      .filter(s => s.label.startsWith('.') && s.label.slice(1).toLowerCase().startsWith(frag))
      .slice(0, 10);
  }

  // Otherwise suggest keywords and db.conn
  return SUGGESTIONS
    .filter(s => !s.label.startsWith('.') && s.label.toLowerCase().includes(frag))
    .slice(0, 10);
}

function completionKind(monaco, kind) {
  if (kind === 'api') return monaco.languages.CompletionItemKind.Function;
  if (kind === 'sql') return monaco.languages.CompletionItemKind.Method;
  if (kind === 'redis') return monaco.languages.CompletionItemKind.Method;
  return monaco.languages.CompletionItemKind.Keyword;
}

function asSnippetText(insert, tail) {
  if (!tail) return insert;
  return `${insert}$0${tail}`;
}

export function registerScriptAutocomplete(monaco, editor, getConnections) {
  const disposable = monaco.languages.registerCompletionItemProvider('javascript', {
    triggerCharacters: ['.', '"', "'", '_'],
    provideCompletionItems: (model, position) => {
      const text = model.getValue();
      const cursor = model.getOffsetAt(position);
      const before = text.slice(0, cursor);

      const list = buildScriptSuggestions(text, cursor, getConnections?.() || []);
      if (!list.length) return { suggestions: [] };

      // Replacement range: either the db.conn("...") fragment, or the current token.
      const connCall = before.match(/db\s*\.\s*conn\s*\(\s*(["']?)([^"']*)$/);
      const tokenCtx = connCall ? null : getTokenContext(text, cursor);
      const quote = connCall ? (connCall[1] || '') : '';
      const frag = connCall ? (connCall[2] || '') : '';
      const startOffset = connCall ? (cursor - frag.length) : (tokenCtx?.tokenStart ?? cursor);

      const startPos = model.getPositionAt(startOffset);
      const range = new monaco.Range(startPos.lineNumber, startPos.column, position.lineNumber, position.column);

      return {
        suggestions: list.map((s) => {
          if (s._ctx === 'dbconn') {
            const insertText = quote ? s.insert : `"${s.insert}"`;
            return {
              label: s.label,
              kind: completionKind(monaco, s.kind),
              insertText,
              range,
              detail: s.desc || undefined,
            };
          }

          let insertBase = s.insert;
          // For dot-suggestions, preserve "<var>." prefix and insert method name.
          if (s.label.startsWith('.') && tokenCtx?.hasDot && tokenCtx?.prefix) {
            insertBase = `${tokenCtx.prefix}.${s.insert}`;
          }

          const hasTail = !!s.tail;
          return {
            label: s.label,
            kind: completionKind(monaco, s.kind),
            insertText: hasTail ? asSnippetText(insertBase, s.tail) : insertBase,
            insertTextRules: hasTail ? monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet : undefined,
            range,
            detail: s.desc || undefined,
          };
        }),
      };
    },
  });

  try {
    editor.updateOptions({
      quickSuggestions: { other: true, comments: true, strings: true },
      suggestOnTriggerCharacters: true,
    });
  } catch { /* ignore */ }

  return disposable;
}

export function ScriptAutocomplete({ value, onChange, onContextMenu, disabled, placeholder, style, connections }) {
  const connectionsRef = useRef([]);
  useEffect(() => { connectionsRef.current = connections || []; }, [connections]);

  const handleMount = useCallback((monaco, editor) => {
    const disposable = registerScriptAutocomplete(monaco, editor, () => connectionsRef.current);
    return () => disposable?.dispose?.();
  }, []);

  return (
    <MonacoCodeEditor
      value={value}
      onChange={onChange}
      language="javascript"
      disabled={disabled}
      placeholder={placeholder}
      style={style}
      className="query-editor-container"
      editorRole="script"
      onContextMenu={onContextMenu}
      onMount={handleMount}
    />
  );
}
