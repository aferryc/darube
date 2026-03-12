import { describe, expect, it } from 'vitest';
import { applyBoxCut, getBoxSelectionText } from './boxSelection';

describe('boxSelection utils', () => {
  it('extracts rectangular text with padding', () => {
    const text = 'abc\n12\nXYZ';
    const sel = getBoxSelectionText(text, { row: 0, col: 1 }, { row: 2, col: 3 });
    expect(sel).toBe('bc\n2 \nYZ');
  });

  it('returns empty when width is zero', () => {
    const text = 'abc\ndef';
    const sel = getBoxSelectionText(text, { row: 0, col: 2 }, { row: 1, col: 2 });
    expect(sel).toBe('');
  });

  it('cuts rectangular text from each affected line', () => {
    const text = 'abcd\n12\nXYZ';
    const next = applyBoxCut(text, { row: 0, col: 1 }, { row: 2, col: 3 });
    expect(next).toBe('ad\n1\nX');
  });
});

