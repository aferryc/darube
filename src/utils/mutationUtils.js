/**
 * Applies mutation history to rows and returns the working dataset (excluding deleted rows).
 * @param {Array} rows - Original rows (array format with _ui_id)
 * @param {Array<string>} columns - Column names
 * @param {Array} history - Mutation history
 * @param {number} historyIndex - Index up to which history is applied
 */
export function computeWorkingData(rows, columns, history, historyIndex) {
  if (!rows || !columns) return [];

  let result = rows.map((r) => {
    const copy = [...r];
    Object.keys(r).forEach((k) => {
      if (k.startsWith('_')) copy[k] = r[k];
    });
    return copy;
  });

  const activeHistory = history.slice(0, historyIndex + 1);

  for (const h of activeHistory) {
    if (h.type === 'update') {
      const row = result.find((r) => r._ui_id === h.uiId);
      if (row) {
        const colIndex = columns.indexOf(h.colName);
        if (colIndex > -1) {
          row[colIndex] = h.newValue;
        }
        if (!row._mutatedCols) row._mutatedCols = {};
        row._mutatedCols[h.colName] = true;
      }
    } else if (h.type === 'delete') {
      const row = result.find((r) => r._ui_id === h.uiId);
      if (row) {
        row._isDeleted = true;
      }
    } else if (h.type === 'insert') {
      const newRow = columns.map((c) => h.newValues[c]);
      newRow._ui_id = h.uiId;
      newRow._isInserted = true;
      newRow._mutatedCols = {};
      columns.forEach((c) => (newRow._mutatedCols[c] = true));
      result.push(newRow);
    }
  }

  return result.filter((r) => !r._isDeleted);
}

/**
 * Consolidates mutation history per uiId so multiple edits become one UPDATE.
 * Filters out inserts that were subsequently deleted.
 */
export function consolidateMutations(history) {
  const mutationMap = {};
  for (const h of history) {
    if (!mutationMap[h.uiId]) {
      mutationMap[h.uiId] = {
        type: h.type,
        originalRow: h.originalRow,
        newValues: h.newValues ? { ...h.newValues } : {},
      };
    } else {
      const existing = mutationMap[h.uiId];
      if (existing.type === 'insert' && h.type === 'update') {
        existing.newValues = { ...existing.newValues, ...h.newValues };
      } else if (existing.type === 'update' && h.type === 'update') {
        existing.newValues = { ...existing.newValues, ...h.newValues };
      } else if (h.type === 'delete') {
        existing.type = 'delete';
      }
    }
  }
  return Object.values(mutationMap).filter(
    (m) => !(m.type === 'delete' && !m.originalRow)
  );
}
