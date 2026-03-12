export function normalizeBox(start, end) {
  const a = start || { row: 0, col: 0 };
  const b = end || { row: 0, col: 0 };
  const minRow = Math.min(a.row, b.row);
  const maxRow = Math.max(a.row, b.row);
  const minCol = Math.min(a.col, b.col);
  const maxCol = Math.max(a.col, b.col);
  return { minRow, maxRow, minCol, maxCol };
}

export function getBoxSelectionText(text, start, end) {
  const { minRow, maxRow, minCol, maxCol } = normalizeBox(start, end);
  if (maxCol <= minCol) return '';
  const lines = String(text ?? '').split('\n');
  const hi = Math.min(maxRow, Math.max(0, lines.length - 1));
  const lo = Math.max(0, Math.min(minRow, hi));

  const out = [];
  for (let r = lo; r <= hi; r++) {
    const line = lines[r] ?? '';
    const padded = line.length < maxCol ? line.padEnd(maxCol, ' ') : line;
    out.push(padded.slice(minCol, maxCol));
  }
  return out.join('\n');
}

export function applyBoxCut(text, start, end) {
  const { minRow, maxRow, minCol, maxCol } = normalizeBox(start, end);
  if (maxCol <= minCol) return String(text ?? '');
  const lines = String(text ?? '').split('\n');
  const hi = Math.min(maxRow, Math.max(0, lines.length - 1));
  const lo = Math.max(0, Math.min(minRow, hi));

  const next = [...lines];
  for (let r = lo; r <= hi; r++) {
    const line = next[r] ?? '';
    const padded = line.length < maxCol ? line.padEnd(maxCol, ' ') : line;
    next[r] = padded.slice(0, minCol) + padded.slice(maxCol);
  }
  return next.join('\n');
}

