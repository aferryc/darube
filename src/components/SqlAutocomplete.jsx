import { useState, useEffect, useRef, useCallback } from 'react';
import { createPortal } from 'react-dom';
import Editor from 'react-simple-code-editor';
import Prism from 'prismjs';
import 'prismjs/components/prism-sql';

// ──────────────────────────────────────────
// Static lists
// ──────────────────────────────────────────
const SQL_KEYWORDS = [
  'SELECT', 'FROM', 'WHERE', 'AND', 'OR', 'NOT', 'IN', 'IS NULL', 'IS NOT NULL',
  'LIKE', 'ILIKE', 'BETWEEN', 'EXISTS', 'CASE', 'WHEN', 'THEN', 'ELSE', 'END',
  'INNER JOIN', 'LEFT JOIN', 'RIGHT JOIN', 'FULL OUTER JOIN', 'CROSS JOIN', 'ON',
  'GROUP BY', 'ORDER BY', 'HAVING', 'LIMIT', 'OFFSET', 'DISTINCT', 'AS',
  'INSERT INTO', 'UPDATE', 'DELETE FROM', 'SET', 'VALUES',
  'CREATE TABLE', 'DROP TABLE', 'ALTER TABLE', 'TRUNCATE',
  'UNION', 'UNION ALL', 'INTERSECT', 'EXCEPT', 'WITH', 'RETURNING',
];

const SQL_FUNCTIONS = [
  'COUNT(*)', 'COUNT(', 'SUM(', 'AVG(', 'MIN(', 'MAX(', 'ROUND(', 'FLOOR(', 'CEIL(',
  'COALESCE(', 'NULLIF(', 'GREATEST(', 'LEAST(',
  'LENGTH(', 'UPPER(', 'LOWER(', 'TRIM(', 'LTRIM(', 'RTRIM(', 'SUBSTRING(', 'POSITION(',
  'CONCAT(', 'REPLACE(', 'SPLIT_PART(', 'REGEXP_REPLACE(', 'TO_TIMESTAMP(',
  'NOW()', 'CURRENT_DATE', 'CURRENT_TIMESTAMP', 'DATE_TRUNC(', 'EXTRACT(',
  'CAST(', 'TO_CHAR(', 'TO_DATE(',
  'ROW_NUMBER()', 'RANK()', 'DENSE_RANK()', 'LAG(', 'LEAD(', 'FIRST_VALUE(', 'LAST_VALUE(',
  'ARRAY_AGG(', 'STRING_AGG(', 'JSON_AGG(', 'JSON_BUILD_OBJECT(',
];

import { parseContext } from '../utils/sqlContext'
import { getTextareaCaretViewportPosition } from '../utils/textareaCaret'

// ──────────────────────────────────────────
// Main component
// ──────────────────────────────────────────
export function SqlAutocomplete({ value, onChange, onKeyDown, disabled, placeholder, style, apiUrl, connectionId }) {
  const [suggestions, setSuggestions] = useState([]);
  const [selectedIdx, setSelectedIdx] = useState(0);
  const [open, setOpen] = useState(false);
  const [dropdownPos, setDropdownPos] = useState(null);

  const colCache = useRef({});
  const tableCache = useRef({});
  const containerRef = useRef(null);
  // Keep latest value in a ref for the keyup listener
  const valueRef = useRef(value);
  useEffect(() => { valueRef.current = value; }, [value]);

  // ── Fetch & cache tables ──
  const fetchTables = useCallback(async () => {
    if (!connectionId) return [];
    if (tableCache.current[connectionId]) return tableCache.current[connectionId];
    try {
      const schemRes = await fetch(`${apiUrl}/api/connections/${connectionId}/metadata/schemas`);
      const schemData = await schemRes.json();
      // schemas is [{name, tables}] or ['name']
      const schemaNames = (schemData.schemas || []).map(s => (typeof s === 'string' ? s : s.name));
      const all = [];
      await Promise.all(schemaNames.map(async (schemaName) => {
        try {
          const tRes = await fetch(`${apiUrl}/api/connections/${connectionId}/metadata/schemas/${encodeURIComponent(schemaName)}/tables`);
          const tData = await tRes.json();
          (tData.tables || []).forEach(t => {
            const tname = typeof t === 'string' ? t : (t.name || t);
            all.push({ name: tname, schema: schemaName });
          });
        } catch { /* skip */ }
      }));
      tableCache.current[connectionId] = all;
      return all;
    } catch { return []; }
  }, [apiUrl, connectionId]);

  // ── Fetch & cache columns ──
  const fetchColumns = useCallback(async (schema, tableName) => {
    if (!connectionId) return [];
    const key = `${connectionId}:${schema}.${tableName}`;
    if (colCache.current[key]) return colCache.current[key];
    try {
      const res = await fetch(`${apiUrl}/api/connections/${connectionId}/metadata/schemas/${encodeURIComponent(schema)}/tables/${encodeURIComponent(tableName)}/columns`);
      const data = await res.json();
      const cols = (data.columns || []).map(c => (typeof c === 'string' ? c : c.name));
      colCache.current[key] = cols;
      return cols;
    } catch { return []; }
  }, [apiUrl, connectionId]);

  // ── Build suggestion list ──
  const buildSuggestions = useCallback(async (query, cursorPos) => {
    const { currentWord, clause, tables, aliasMap, dotPrefix } = parseContext(query, cursorPos);

    // alias.col → show columns scoped to the table
    if (dotPrefix !== null) {
      const tableName = aliasMap[dotPrefix] || dotPrefix;
      const tableRef = tables.find(t => t.name === tableName || t.name.split('.').pop() === tableName);
      const schema = (tableRef && tableRef.schema) || 'public';
      const realTable = tableName.split('.').pop();
      const cols = await fetchColumns(schema, realTable);
      return cols
        .filter(c => c.toLowerCase().includes(currentWord.toLowerCase()))
        .map(c => ({ label: `${dotPrefix}.${c}`, insert: `${dotPrefix}.${c}`, kind: 'column' }));
    }

    // No prefix typed yet — nothing to suggest  
    if (!currentWord) return [];

    // FROM / JOIN → table suggestions
    if (clause === 'FROM' || clause === 'JOIN') {
      const allTables = await fetchTables();
      const filtered = allTables.filter(t => t.name.toLowerCase().includes(currentWord.toLowerCase()));
      return filtered.map(t => ({ label: t.name, insert: t.name, kind: 'table', sub: t.schema }));
    }

    // SELECT / WHERE / ON etc. + known tables → column suggestions
    let colSuggestions = [];
    if (tables.length > 0) {
      for (const tbl of tables) {
        const schema = tbl.schema || 'public';
        const realTable = tbl.name.split('.').pop();
        const cols = await fetchColumns(schema, realTable);
        for (const c of cols) {
          if (c.toLowerCase().includes(currentWord.toLowerCase())) {
            colSuggestions.push({ label: c, insert: c, kind: 'column', sub: tbl.name });
          }
        }
      }
    }

    // Keywords / functions
    const kwSuggestions = [...SQL_KEYWORDS, ...SQL_FUNCTIONS]
      .filter(k => k.toLowerCase().includes(currentWord.toLowerCase()))
      .map(k => ({ label: k, insert: k, kind: 'keyword' }));

    const seen = new Set();
    const merged = [];
    [...colSuggestions, ...kwSuggestions].forEach(s => {
      if (!seen.has(s.label)) { seen.add(s.label); merged.push(s); }
    });
    return merged.slice(0, 20);
  }, [fetchTables, fetchColumns]);

  // ── keyup handler attached directly to textarea ──
  const refreshSuggestions = useCallback(async (e) => {
    // Ignore pure modifier / navigation keys
    const ignoredKeys = ['ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight', 'Enter', 'Tab', 'Escape', 'Shift', 'Control', 'Alt', 'Meta'];
    if (ignoredKeys.includes(e.key)) return;

    const ta = e.target;
    const cursorPos = ta.selectionStart;
    const query = valueRef.current;

    const list = await buildSuggestions(query, cursorPos);
    if (list.length === 0) {
      setOpen(false);
    } else {
      setSuggestions(list);
      setSelectedIdx(0);
      setOpen(true);

      const caret = getTextareaCaretViewportPosition(ta, cursorPos);
      if (caret) {
        const desiredTop = caret.top + caret.height + 6;
        const desiredLeft = caret.left;
        const maxLeft = Math.max(8, window.innerWidth - 420);
        const maxTop = Math.max(8, window.innerHeight - 300);
        setDropdownPos({
          top: Math.min(desiredTop, maxTop),
          left: Math.min(desiredLeft, maxLeft),
        });
      }
    }
  }, [buildSuggestions]);

  // Attach listener once textarea is mounted
  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;
    // react-simple-code-editor wraps a textarea
    const attach = () => {
      const ta = container.querySelector('textarea');
      if (ta) {
        ta.addEventListener('keyup', refreshSuggestions);
        return ta;
      }
      return null;
    };
    let ta = attach();
    // If textarea not yet there, retry
    if (!ta) {
      const obs = new MutationObserver(() => {
        ta = attach();
        if (ta) obs.disconnect();
      });
      obs.observe(container, { childList: true, subtree: true });
      return () => obs.disconnect();
    }
    return () => { if (ta) ta.removeEventListener('keyup', refreshSuggestions); };
  }, [refreshSuggestions]);

  // Invalidate caches on connection change
  useEffect(() => {
    colCache.current = {};
    tableCache.current = {};
    setOpen(false);
    setDropdownPos(null);
  }, [connectionId]);

  // ── Insert suggestion ──
  const insertSuggestion = useCallback((suggestion) => {
    const ta = containerRef.current?.querySelector('textarea');
    if (!ta) return;

    const pos = ta.selectionStart;
    const query = valueRef.current;
    const before = query.slice(0, pos);
    // Find start of current raw word (may include "alias.")
    const wMatch = before.match(/([^\s,();]+)$/);
    const rawWord = wMatch ? wMatch[1] : '';
    const wordStart = pos - rawWord.length;

    const newQuery = query.slice(0, wordStart) + suggestion.insert + query.slice(pos);
    onChange(newQuery);

    setTimeout(() => {
      const newPos = wordStart + suggestion.insert.length;
      ta.setSelectionRange(newPos, newPos);
      ta.focus();
    }, 0);

    setOpen(false);
  }, [onChange]);

  // ── Keyboard navigation (intercepted before the editor) ──
  const handleKeyDown = useCallback((e) => {
    if (open && suggestions.length > 0) {
      if (e.key === 'ArrowDown') { e.preventDefault(); setSelectedIdx(i => Math.min(i + 1, suggestions.length - 1)); return; }
      if (e.key === 'ArrowUp')   { e.preventDefault(); setSelectedIdx(i => Math.max(i - 1, 0)); return; }
      if (e.key === 'Tab') { e.preventDefault(); insertSuggestion(suggestions[selectedIdx]); return; }
      if (e.key === 'Escape') { setOpen(false); return; }
      // Enter: only intercept if dropdown is open (don't block Cmd+Enter)
      if (e.key === 'Enter' && !e.metaKey && !e.ctrlKey) { e.preventDefault(); insertSuggestion(suggestions[selectedIdx]); return; }
    }
    onKeyDown?.(e);
  }, [open, suggestions, selectedIdx, insertSuggestion, onKeyDown]);

  // Close on click outside
  useEffect(() => {
    const close = (e) => {
      if (containerRef.current && !containerRef.current.contains(e.target)) {
        setOpen(false);
      }
    };
    document.addEventListener('mousedown', close);
    return () => document.removeEventListener('mousedown', close);
  }, []);

  const kindLabel = (kind) => {
    if (kind === 'column') return <span className="ac-badge ac-badge-col">col</span>;
    if (kind === 'table')  return <span className="ac-badge ac-badge-tbl">tbl</span>;
    return <span className="ac-badge ac-badge-kw">kw</span>;
  };

  return (
    <div ref={containerRef} style={{ position: 'relative', display: 'flex', flexDirection: 'column', flex: 1, width: '100%' }}>
      <Editor
        value={value}
        onValueChange={onChange}
        highlight={code => Prism.highlight(code, Prism.languages.sql, 'sql')}
        padding={16}
        className="query-editor-container"
        textareaClassName="query-editor-textarea"
        onKeyDown={handleKeyDown}
        disabled={disabled}
        placeholder={placeholder}
        style={style}
      />

      {open && suggestions.length > 0 && dropdownPos && createPortal(
        <ul
          className="ac-dropdown"
          style={{ position: 'fixed', top: `${dropdownPos.top}px`, left: `${dropdownPos.left}px` }}
          onMouseDown={e => e.preventDefault()}
        >
          {suggestions.map((s, i) => (
            <li
              key={s.label + i}
              className={`ac-item ${i === selectedIdx ? 'ac-selected' : ''}`}
              onMouseEnter={() => setSelectedIdx(i)}
              onClick={() => insertSuggestion(s)}
            >
              {kindLabel(s.kind)}
              <span className="ac-label">{s.label}</span>
              {s.sub && <span className="ac-sub">{s.sub}</span>}
            </li>
          ))}
        </ul>,
        document.body
      )}
    </div>
  );
}
