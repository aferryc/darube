import { useEffect, useMemo, useRef, useState } from 'react';

function isTestEnv() {
  return !!(import.meta?.env?.MODE === 'test' || import.meta?.env?.VITEST);
}

function normalizeNewlines(v) {
  return String(v ?? '').replace(/\r\n/g, '\n');
}

export function MonacoCodeEditor({
  value,
  onChange,
  language = 'plaintext',
  disabled = false,
  placeholder,
  className,
  style,
  editorRole,
  onKeyDown,
  onContextMenu,
  onMount,
  options,
}) {
  const containerRef = useRef(null);
  const editorRef = useRef(null);
  const modelRef = useRef(null);
  const applyingRef = useRef(false);
  const onChangeRef = useRef(onChange);
  useEffect(() => { onChangeRef.current = onChange; }, [onChange]);

  const [focused, setFocused] = useState(false);
  const showPlaceholder = !!placeholder && !focused && !String(value || '');
  const testEnv = isTestEnv();

  const defaultOptions = useMemo(() => ({
    theme: 'darube-dark',
    language,
    readOnly: !!disabled,
    minimap: { enabled: false },
    lineNumbers: 'off',
    glyphMargin: false,
    folding: false,
    lineDecorationsWidth: 10,
    lineNumbersMinChars: 0,
    renderLineHighlight: 'none',
    overviewRulerLanes: 0,
    hideCursorInOverviewRuler: true,
    overviewRulerBorder: false,
    scrollBeyondLastLine: false,
    wordWrap: 'off',
    fontFamily: '"JetBrains Mono", "Menlo", monospace',
    fontSize: 12,
    lineHeight: 18,
    padding: { top: 14, bottom: 14 },
    contextmenu: false,
    quickSuggestions: { other: true, comments: true, strings: true },
    suggestOnTriggerCharacters: true,
    tabCompletion: 'on',
    automaticLayout: false,
  }), [disabled, language]);

  // ── Monaco runtime ───────────────────────────────────────────────────────
  useEffect(() => {
    if (testEnv) return;
    let cancelled = false;
    let resizeObs = null;
    let cleanupMount = null;
    let dispose = null;

    (async () => {
      const el = containerRef.current;
      if (!el) return;

      // Ensure Monaco is configured (workers + language contributions) before editor creation.
      const setup = await import('../monaco/setup');
      const monaco = await import('monaco-editor/esm/vs/editor/editor.main.js');
      setup.applyDarubeTheme?.(monaco);

      if (cancelled) return;

      // Mark container so context-menu code can detect Monaco editors.
      el.dataset.darubeMonaco = '1';
      if (editorRole) el.dataset.darubeEditorRole = editorRole;
      el.dataset.darubeReadOnly = defaultOptions.readOnly ? '1' : '0';

      const model = monaco.editor.createModel(normalizeNewlines(value), language);
      modelRef.current = model;

      const editor = monaco.editor.create(el, {
        model,
        ...defaultOptions,
        ...(options || {}),
      });
      editorRef.current = editor;

      // Expose editor for Darube context-menu handlers.
      el.__darubeMonacoEditor = editor;

      const layout = () => {
        try { editor.layout(); } catch { /* ignore */ }
      };
      layout();

      if (typeof ResizeObserver !== 'undefined') {
        resizeObs = new ResizeObserver(() => layout());
        resizeObs.observe(el);
      } else {
        window.addEventListener('resize', layout);
      }

      const d0 = editor.onDidFocusEditorText(() => setFocused(true));
      const d1 = editor.onDidBlurEditorText(() => setFocused(false));

      const d2 = editor.onDidChangeModelContent(() => {
        if (applyingRef.current) return;
        onChangeRef.current?.(model.getValue());
      });

      // Cmd/Ctrl + Enter support (keeps existing behavior via the passed onKeyDown).
      if (onKeyDown) {
        editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter, () => {
          const selection = editor.getSelection();
          const m = editor.getModel();
          if (!selection || !m) return;
          const selectionStart = m.getOffsetAt(selection.getStartPosition());
          const selectionEnd = m.getOffsetAt(selection.getEndPosition());
          onKeyDown({
            key: 'Enter',
            metaKey: true,
            ctrlKey: true,
            preventDefault: () => {},
            target: { selectionStart, selectionEnd },
          });
        });
      }

      // Forward contextmenu events to Darube handler.
      const onDomContextMenu = (e) => onContextMenu?.(e);
      if (onContextMenu) el.addEventListener('contextmenu', onDomContextMenu, true);

      if (onMount) cleanupMount = onMount(monaco, editor) || null;

      // Cleanups
      const disposeAll = () => {
        d0?.dispose?.();
        d1?.dispose?.();
        d2?.dispose?.();
        cleanupMount?.();
        if (onContextMenu) el.removeEventListener('contextmenu', onDomContextMenu, true);
        if (resizeObs) resizeObs.disconnect();
        else window.removeEventListener('resize', layout);
        try { editor.dispose(); } catch { /* ignore */ }
        try { model.dispose(); } catch { /* ignore */ }
        editorRef.current = null;
        modelRef.current = null;
        delete el.__darubeMonacoEditor;
      };

      dispose = disposeAll;
    })();

    return () => {
      cancelled = true;
      dispose?.();
      dispose = null;
    };
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [testEnv]);

  // Keep external value in sync.
  useEffect(() => {
    if (testEnv) return;
    const model = modelRef.current;
    if (!model) return;
    const next = normalizeNewlines(value);
    if (model.getValue() === next) return;
    applyingRef.current = true;
    model.setValue(next);
    applyingRef.current = false;
  }, [value]);

  // Update language / readonly.
  useEffect(() => {
    if (testEnv) return;
    const editor = editorRef.current;
    const model = modelRef.current;
    if (!editor || !model) return;
    (async () => {
      const monaco = await import('monaco-editor/esm/vs/editor/editor.main.js');
      try { monaco.editor.setModelLanguage(model, language); } catch { /* ignore */ }
      try { editor.updateOptions({ readOnly: !!disabled }); } catch { /* ignore */ }
      const el = containerRef.current;
      if (el) el.dataset.darubeReadOnly = disabled ? '1' : '0';
    })();
  }, [language, disabled]);

  // ── Test fallback (keeps unit tests fast + reliable) ─────────────────────
  if (testEnv) {
    return (
      <textarea
        className={className}
        style={style}
        value={value}
        onChange={(e) => onChange?.(e.target.value)}
        onKeyDown={onKeyDown}
        onContextMenu={onContextMenu}
        disabled={disabled}
        placeholder={placeholder}
        data-darube-editor-role={editorRole || undefined}
      />
    );
  }

  return (
    <div className={className} style={{ ...style, position: 'relative', overflow: 'hidden' }}>
      <div ref={containerRef} style={{ position: 'absolute', inset: 0 }} />
      {showPlaceholder && (
        <div
          style={{
            position: 'absolute',
            inset: 0,
            padding: '16px',
            whiteSpace: 'pre-wrap',
            fontFamily: '"JetBrains Mono", "Menlo", monospace',
            fontSize: '12px',
            lineHeight: '18px',
            color: 'rgba(148, 163, 184, 0.65)',
            pointerEvents: 'none',
            userSelect: 'none',
          }}
        >
          {placeholder}
        </div>
      )}
    </div>
  );
}
