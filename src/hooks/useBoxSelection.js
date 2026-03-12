import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { applyBoxCut, getBoxSelectionText, normalizeBox } from '../utils/boxSelection';

function getLineHeightPx(cs) {
  const lh = cs.lineHeight;
  const fs = parseFloat(cs.fontSize || '12') || 12;
  if (!lh || lh === 'normal') return Math.round(fs * 1.4);
  const v = parseFloat(lh);
  return Number.isFinite(v) && v > 0 ? v : Math.round(fs * 1.4);
}

function measureCharWidthPx(cs) {
  try {
    const span = document.createElement('span');
    span.textContent = 'M';
    span.style.position = 'absolute';
    span.style.left = '-9999px';
    span.style.top = '-9999px';
    span.style.visibility = 'hidden';
    span.style.whiteSpace = 'pre';
    span.style.fontFamily = cs.fontFamily;
    span.style.fontSize = cs.fontSize;
    span.style.fontWeight = cs.fontWeight;
    span.style.letterSpacing = cs.letterSpacing;
    document.body.appendChild(span);
    const w = span.getBoundingClientRect().width;
    document.body.removeChild(span);
    return w > 0 ? w : 8;
  } catch {
    return 8;
  }
}

function getScrollable(ta) {
  if (!ta) return null;
  const parent = ta.parentElement;
  const useParent = parent && (parent.scrollHeight > parent.clientHeight || parent.scrollWidth > parent.clientWidth);
  return useParent ? parent : ta;
}

function getMousePos(ta, clientX, clientY, metrics, lineCount) {
  const rect = ta.getBoundingClientRect();
  const scrollEl = getScrollable(ta);
  const scrollTop = scrollEl?.scrollTop || 0;
  const scrollLeft = scrollEl?.scrollLeft || 0;
  const x = clientX - rect.left - metrics.offsetLeft + scrollLeft;
  const y = clientY - rect.top - metrics.offsetTop + scrollTop;
  const col = Math.max(0, Math.floor(x / metrics.charWidth));
  const row = Math.min(Math.max(0, Math.floor(y / metrics.lineHeight)), Math.max(0, lineCount - 1));
  return { row, col };
}

export function useBoxSelection({
  containerRef,
  valueRef,
  onChange,
  disabled,
  onActivate,
}) {
  const [start, setStart] = useState(null);
  const [end, setEnd] = useState(null);
  const [metrics, setMetrics] = useState({ charWidth: 8, lineHeight: 16, offsetLeft: 0, offsetTop: 0 });

  const draggingRef = useRef(false);
  const metricsRef = useRef(metrics);
  const onChangeRef = useRef(onChange);
  const onActivateRef = useRef(onActivate);
  const disabledRef = useRef(disabled);
  const textareaRef = useRef(null);
  const startRef = useRef(null);
  const endRef = useRef(null);
  useEffect(() => { startRef.current = start; }, [start]);
  useEffect(() => { endRef.current = end; }, [end]);
  useEffect(() => { metricsRef.current = metrics; }, [metrics]);
  useEffect(() => { onChangeRef.current = onChange; }, [onChange]);
  useEffect(() => { onActivateRef.current = onActivate; }, [onActivate]);
  useEffect(() => { disabledRef.current = disabled; }, [disabled]);

  const active = !!(start && end);

  const rect = useMemo(() => normalizeBox(start, end), [start, end]);

  useEffect(() => {
    const ta = textareaRef.current;
    if (!ta) return;
    if (!active) {
      delete ta.dataset.boxSelection;
      return;
    }
    const { minRow, minCol, maxRow, maxCol } = normalizeBox(startRef.current, endRef.current);
    ta.dataset.boxSelection = `${minRow},${minCol},${maxRow},${maxCol}`;
  }, [active, start, end]);

  const clear = useCallback(() => {
    draggingRef.current = false;
    setStart(null);
    setEnd(null);
    const ta = textareaRef.current;
    if (ta) delete ta.dataset.boxSelection;
  }, []);

  // Clear box selection when clicking anywhere outside the editor.
  useEffect(() => {
    if (!active) return;
    const onDown = (e) => {
      const container = containerRef.current;
      if (!container) return;
      if (container.contains(e.target)) return;
      clear();
    };
    document.addEventListener('mousedown', onDown, true);
    return () => document.removeEventListener('mousedown', onDown, true);
  }, [active, clear, containerRef]);

  const overlay = useMemo(() => {
    if (!active) return null;
    const text = String(valueRef.current ?? '');
    const lines = text.split('\n');
    const lineCount = lines.length || 1;
    const { minRow, maxRow, minCol, maxCol } = rect;
    if (maxCol <= minCol) return null;

    const loRow = Math.max(0, Math.min(minRow, lineCount - 1));
    const hiRow = Math.max(0, Math.min(maxRow, lineCount - 1));

    const pieces = [];
    for (let r = loRow; r <= hiRow; r++) {
      pieces.push({
        row: r,
        top: metrics.offsetTop + r * metrics.lineHeight,
        left: metrics.offsetLeft + minCol * metrics.charWidth,
        width: (maxCol - minCol) * metrics.charWidth,
        height: metrics.lineHeight,
      });
    }
    return pieces;
  }, [active, metrics, rect, valueRef]);

  const attachToTextarea = useCallback((container) => {
    if (!container) return null;
    const ta = container.querySelector('textarea');
    return ta || null;
  }, []);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const attach = () => {
      const ta = attachToTextarea(container);
      if (!ta) return null;
      textareaRef.current = ta;

      const cs = window.getComputedStyle(ta);
      const padLeft = parseFloat(cs.paddingLeft || '0') || 0;
      const padTop = parseFloat(cs.paddingTop || '0') || 0;
      const measured = {
        charWidth: measureCharWidthPx(cs),
        lineHeight: getLineHeightPx(cs),
        offsetLeft: padLeft,
        offsetTop: padTop,
      };
      metricsRef.current = measured;
      setMetrics(measured);

      const scrollEl = getScrollable(ta);

      const onMouseDown = (e) => {
        if (disabledRef.current) return;
        const hasSelection = !!(startRef.current && endRef.current);
        if (!(e.altKey && e.shiftKey) || e.button !== 0) {
          if (hasSelection && e.button === 0) clear();
          return;
        }
        onActivateRef.current?.();
        e.preventDefault();
        e.stopPropagation();
        const text = String(valueRef.current ?? '');
        const lineCount = text.split('\n').length || 1;
        const pos = getMousePos(ta, e.clientX, e.clientY, metricsRef.current, lineCount);
        draggingRef.current = true;
        setStart({ row: pos.row, col: pos.col });
        setEnd({ row: pos.row, col: pos.col });
      };

      const onMouseMove = (e) => {
        if (!draggingRef.current) return;
        const text = String(valueRef.current ?? '');
        const lineCount = text.split('\n').length || 1;
        const pos = getMousePos(ta, e.clientX, e.clientY, metricsRef.current, lineCount);
        setEnd({ row: pos.row, col: pos.col });
      };

      const onMouseUp = () => {
        if (!draggingRef.current) return;
        draggingRef.current = false;
      };

      const onKeyDown = async (e) => {
        if (!startRef.current || !endRef.current) return;
        if (e.key === 'Escape') {
          clear();
          return;
        }

        const isCopy = (e.key.toLowerCase() === 'c') && (e.metaKey || e.ctrlKey);
        const isCut = (e.key.toLowerCase() === 'x') && (e.metaKey || e.ctrlKey);
        if (!isCopy && !isCut) return;
        if (disabledRef.current) return;

        const text = String(valueRef.current ?? '');
        const selText = getBoxSelectionText(text, startRef.current, endRef.current);
        if (!selText) return;

        e.preventDefault();
        e.stopPropagation();
        if (navigator.clipboard?.writeText) {
          await navigator.clipboard.writeText(selText);
        } else if (window.darube?.clipboard?.writeText) {
          try {
            window.darube.clipboard.writeText(selText);
          } catch { /* ignore */ }
        }

        if (isCut) {
          const next = applyBoxCut(text, startRef.current, endRef.current);
          onChangeRef.current?.(next);
          clear();
        }
      };

      const onBlur = () => {
        if (!draggingRef.current) return;
        draggingRef.current = false;
      };

      ta.addEventListener('mousedown', onMouseDown);
      document.addEventListener('mousemove', onMouseMove);
      document.addEventListener('mouseup', onMouseUp);
      ta.addEventListener('keydown', onKeyDown);
      ta.addEventListener('blur', onBlur);

      return () => {
        ta.removeEventListener('mousedown', onMouseDown);
        document.removeEventListener('mousemove', onMouseMove);
        document.removeEventListener('mouseup', onMouseUp);
        ta.removeEventListener('keydown', onKeyDown);
        ta.removeEventListener('blur', onBlur);
        if (textareaRef.current === ta) textareaRef.current = null;
      };
    };

    let cleanup = attach();
    if (!cleanup) {
      const obs = new MutationObserver(() => {
        cleanup = attach();
        if (cleanup) obs.disconnect();
      });
      obs.observe(container, { childList: true, subtree: true });
      return () => obs.disconnect();
    }
    return () => cleanup?.();
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [attachToTextarea, clear, containerRef]);

  return { active, rect, overlay, clear };
}
