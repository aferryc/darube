/**
 * Groups tables by suffix patterns like _20191225 or _202403.
 * Tables matching the pattern are grouped under prefix_*; others stay as single items.
 */
export function groupTables(tables) {
  if (!tables) return [];
  const groups = {};
  const result = [];

  // Regex to match suffix patterns like _20191225 or _202403
  const regex = /^(.*)_([0-9]{4,})$/;

  tables.forEach((tbl) => {
    const match = tbl.name.match(regex);
    if (match) {
      const prefix = match[1];
      if (!groups[prefix]) {
        groups[prefix] = [];
      }
      groups[prefix].push(tbl);
    } else {
      result.push({ ...tbl, isGroup: false });
    }
  });

  Object.keys(groups).forEach((prefix) => {
    if (groups[prefix].length > 1) {
      result.push({
        name: prefix + '_*',
        isGroup: true,
        tables: groups[prefix],
      });
    } else {
      result.push({ ...groups[prefix][0], isGroup: false });
    }
  });

  result.sort((a, b) => a.name.localeCompare(b.name));
  return result;
}
