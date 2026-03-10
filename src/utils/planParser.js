/**
 * Recursively parses Postgres JSON EXPLAIN plan node.
 */
export function parsePostgresNode(node) {
  if (!node) return null;
  return {
    type: node['Node Type'] || 'Unknown Node',
    relation: node['Relation Name'] || '',
    alias: node['Alias'] || '',
    cost: node['Total Cost'] ? node['Total Cost'].toFixed(2) : '0.00',
    rows: node['Actual Rows'] || node['Plan Rows'] || 0,
    time: node['Actual Total Time']
      ? `${node['Actual Total Time'].toFixed(3)} ms`
      : '',
    loops: node['Actual Loops'] || 1,
    children: node.Plans ? node.Plans.map(parsePostgresNode) : [],
  };
}
