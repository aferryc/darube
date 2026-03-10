// Minimal caret position helper for textarea elements.
// Returns viewport-relative coordinates suitable for `position: fixed` dropdowns.
export function getTextareaCaretViewportPosition(textarea, caretPos) {
  if (!textarea) return null;
  const pos = typeof caretPos === 'number' ? caretPos : textarea.selectionEnd || 0;

  const style = window.getComputedStyle(textarea);

  const mirror = document.createElement('div');
  mirror.style.position = 'absolute';
  mirror.style.visibility = 'hidden';
  mirror.style.whiteSpace = 'pre-wrap';
  mirror.style.wordWrap = 'break-word';
  mirror.style.top = '0';
  mirror.style.left = '-9999px';

  // Copy the relevant typography/box props so wrapping matches the textarea.
  const props = [
    'boxSizing',
    'width',
    'paddingTop',
    'paddingRight',
    'paddingBottom',
    'paddingLeft',
    'borderTopWidth',
    'borderRightWidth',
    'borderBottomWidth',
    'borderLeftWidth',
    'fontFamily',
    'fontSize',
    'fontWeight',
    'fontStyle',
    'letterSpacing',
    'textTransform',
    'textIndent',
    'lineHeight',
    'tabSize',
  ];
  props.forEach((p) => {
    mirror.style[p] = style[p];
  });

  // Ensure width matches rendered textarea (style.width can be "auto" in some cases).
  mirror.style.width = `${textarea.getBoundingClientRect().width}px`;

  const value = textarea.value || '';
  mirror.textContent = value.slice(0, pos);

  const marker = document.createElement('span');
  // Marker needs some content to get a box; a zero-width char is unreliable across browsers.
  marker.textContent = value.slice(pos) || '.';
  mirror.appendChild(marker);

  document.body.appendChild(mirror);

  const markerTop = marker.offsetTop;
  const markerLeft = marker.offsetLeft;
  document.body.removeChild(mirror);

  const rect = textarea.getBoundingClientRect();
  const lineHeight = parseFloat(style.lineHeight) || 18;

  return {
    top: rect.top + (markerTop - textarea.scrollTop),
    left: rect.left + (markerLeft - textarea.scrollLeft),
    height: lineHeight,
  };
}

