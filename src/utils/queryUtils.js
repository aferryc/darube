/**
 * Extracts the target table name from a simple SELECT query's FROM clause.
 * Returns null if the query has JOIN, GROUP BY, or UNION, or if no FROM is found.
 */
export function getTargetTable(query) {
  if (!query) return null;
  const match = query.match(/from\s+([a-zA-Z0-9_."`\[\]]+)(\s|$|;)/i);
  if (match && !query.match(/join|group by|union/i)) {
    return match[1].replace(/["`\[\]]/g, '');
  }
  return null;
}
