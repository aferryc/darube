/* @refresh reset */
import { useCallback, useEffect, useRef } from 'react';

import { MonacoCodeEditor } from './MonacoCodeEditor';

import { parseContext } from '../utils/sqlContext';

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

function completionKind(monaco, kind) {
  if (kind === 'column') return monaco.languages.CompletionItemKind.Field;
  if (kind === 'table') return monaco.languages.CompletionItemKind.Class;
  return monaco.languages.CompletionItemKind.Keyword;
}

export function SqlAutocomplete({
  value,
  onChange,
  onKeyDown,
  onContextMenu,
  onSelectionChange,
  disabled,
  placeholder,
  style,
  apiUrl,
  connectionId,
}) {
  const apiUrlRef = useRef(apiUrl);
  const connectionIdRef = useRef(connectionId);
  useEffect(() => { apiUrlRef.current = apiUrl; }, [apiUrl]);
  useEffect(() => { connectionIdRef.current = connectionId; }, [connectionId]);

  const colCache = useRef({});
  const tableCache = useRef({});

  // Invalidate caches on connection change.
  useEffect(() => {
    colCache.current = {};
    tableCache.current = {};
  }, [connectionId]);

  const fetchTables = useCallback(async () => {
    const cId = connectionIdRef.current;
    if (!cId) return [];
    if (tableCache.current[cId]) return tableCache.current[cId];

    const base = apiUrlRef.current;
    try {
      const schemRes = await fetch(`${base}/api/connections/${cId}/metadata/schemas`);
      const schemData = await schemRes.json();
      const schemaNames = (schemData.schemas || []).map(s => (typeof s === 'string' ? s : s.name));
      const all = [];
      await Promise.all(schemaNames.map(async (schemaName) => {
        try {
          const tRes = await fetch(`${base}/api/connections/${cId}/metadata/schemas/${encodeURIComponent(schemaName)}/tables`);
          const tData = await tRes.json();
          (tData.tables || []).forEach(t => {
            const tname = typeof t === 'string' ? t : (t.name || t);
            all.push({ name: tname, schema: schemaName });
          });
        } catch { /* skip */ }
      }));
      tableCache.current[cId] = all;
      return all;
    } catch {
      return [];
    }
  }, []);

  const fetchColumns = useCallback(async (schema, tableName) => {
    const cId = connectionIdRef.current;
    if (!cId) return [];
    const key = `${cId}:${schema}.${tableName}`;
    if (colCache.current[key]) return colCache.current[key];

    const base = apiUrlRef.current;
    try {
      const res = await fetch(`${base}/api/connections/${cId}/metadata/schemas/${encodeURIComponent(schema)}/tables/${encodeURIComponent(tableName)}/columns`);
      const data = await res.json();
      const cols = (data.columns || []).map(c => (typeof c === 'string' ? c : c.name));
      colCache.current[key] = cols;
      return cols;
    } catch {
      return [];
    }
  }, []);

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
        .filter(c => c.toLowerCase().includes((currentWord || '').toLowerCase()))
        .map(c => ({ label: `${dotPrefix}.${c}`, insert: `${dotPrefix}.${c}`, kind: 'column' }));
    }

    if (!currentWord) return [];

    // FROM / JOIN → table suggestions
    if (clause === 'FROM' || clause === 'JOIN') {
      const allTables = await fetchTables();
      const filtered = allTables.filter(t => t.name.toLowerCase().includes(currentWord.toLowerCase()));
      return filtered.map(t => ({ label: t.name, insert: t.name, kind: 'table', sub: t.schema }));
    }

    // SELECT / WHERE / ON etc. + known tables → column suggestions
    const colSuggestions = [];
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
  }, [fetchColumns, fetchTables]);

  const handleMount = useCallback((monaco, editor) => {
    // Provide completions using the same context parser logic as before.
    const disposable = monaco.languages.registerCompletionItemProvider('sql', {
      triggerCharacters: ['.', ' ', '"', "'", '_'],
      provideCompletionItems: async (model, position) => {
        const query = model.getValue();
        const cursorPos = model.getOffsetAt(position);

        const list = await buildSuggestions(query, cursorPos);
        if (!list.length) return { suggestions: [] };

        const before = query.slice(0, cursorPos);
        const wMatch = before.match(/([^\s,();]+)$/);
        const rawWord = wMatch ? wMatch[1] : '';
        const startOffset = cursorPos - rawWord.length;
        const startPos = model.getPositionAt(startOffset);
        const range = new monaco.Range(startPos.lineNumber, startPos.column, position.lineNumber, position.column);

        return {
          suggestions: list.map((s) => ({
            label: s.label,
            kind: completionKind(monaco, s.kind),
            insertText: s.insert,
            range,
            detail: s.sub || undefined,
          })),
        };
      },
    });

    // Make sure the editor is in a good state for "type-to-suggest".
    try {
      editor.updateOptions({
        quickSuggestions: { other: true, comments: true, strings: true },
        suggestOnTriggerCharacters: true,
      });
    } catch { /* ignore */ }

    return () => disposable?.dispose?.();
  }, [buildSuggestions]);

  return (
    <MonacoCodeEditor
      value={value}
      onChange={onChange}
      language="sql"
      disabled={disabled}
      placeholder={placeholder}
      style={style}
      className="query-editor-container"
      editorRole="sql"
      onKeyDown={onKeyDown}
      onContextMenu={onContextMenu}
      onSelectionChange={onSelectionChange}
      onMount={handleMount}
    />
  );
}
