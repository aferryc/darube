/**
 * Parses SQL query context at cursor position for autocomplete.
 * Returns currentWord, clause, tables, aliasMap, dotPrefix.
 */
export function parseContext(query, cursorPos) {
  const before = query.slice(0, cursorPos);

  // Raw word at cursor (can include dot)
  const wordMatch = before.match(/([^\s,();]+)$/);
  const rawWord = wordMatch ? wordMatch[1] : '';

  // Dot prefix: "alias.col" → dotPrefix='alias', currentWord='col'
  let dotPrefix = null;
  let currentWord = rawWord;
  const dotIdx = rawWord.lastIndexOf('.');
  if (dotIdx !== -1) {
    dotPrefix = rawWord.slice(0, dotIdx);
    currentWord = rawWord.slice(dotIdx + 1);
  }

  // Determine active clause (last keyword before cursor)
  const clauseRe = /\b(SELECT|FROM|WHERE|ON|HAVING|SET|JOIN|GROUP\s+BY|ORDER\s+BY)\b/gi;
  let lastClause = 'SELECT';
  let lastIdx = -1;
  let m;
  while ((m = clauseRe.exec(before)) !== null) {
    if (m.index > lastIdx) {
      lastIdx = m.index;
      lastClause = m[1].toUpperCase().replace(/\s+/g, '_');
    }
  }

  // Parse all table references from the FULL query
  const tables = [];
  const tableRe = /(?:FROM|JOIN)\s+([\w.]+)(?:\s+AS\s+(\w+)|\s+(\w+))?/gi;
  let tr;
  while ((tr = tableRe.exec(query)) !== null) {
    const name = tr[1];
    const alias = tr[2] || tr[3] || null;
    if (!tables.find((t) => t.name === name)) {
      tables.push({ name, alias, schema: null });
    }
  }

  const aliasMap = {};
  tables.forEach((t) => {
    if (t.alias) aliasMap[t.alias] = t.name;
  });

  return { currentWord, rawWord, clause: lastClause, tables, aliasMap, dotPrefix };
}
