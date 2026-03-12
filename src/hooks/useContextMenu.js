import { useState } from 'react';

export function useContextMenu() {
  const [contextMenu, setContextMenu] = useState({ visible: false, x: 0, y: 0, type: null, data: null });

  const showMenu = (e, type, data) => {
    e.preventDefault();
    e.stopPropagation();
    setContextMenu({ visible: true, x: e.clientX, y: e.clientY, type, data });
  };

  const hideMenu = () => setContextMenu(prev => ({ ...prev, visible: false }));

  const handleConnectionContextMenu = (e, conn) => showMenu(e, 'connection', conn);
  const handleTableContextMenu = (e, tbl, schemaName, cId) => showMenu(e, 'table', { tbl, schemaName, cId });
  const handleTextContextMenu = (e) => {
    const el = e.target;
    const isTextField = el && (
      el.tagName === 'TEXTAREA'
      || (el.tagName === 'INPUT' && (el.type === 'text' || el.type === 'search' || el.type === 'password' || el.type === 'email' || el.type === 'url' || el.type === 'tel'))
    );

    if (isTextField) {
      const start = el.selectionStart ?? 0;
      const end = el.selectionEnd ?? 0;
      const hasBoxSelection = !!el?.dataset?.boxSelection;
      const hasSelection = hasBoxSelection || (end > start);
      const editorRole = el?.dataset?.darubeEditorRole || null;
      const hasText = !!String(el?.value || '').trim();
      showMenu(e, 'text', {
        el,
        kind: 'dom',
        hasSelection,
        readOnly: !!el.readOnly || !!el.disabled,
        boxSelection: el?.dataset?.boxSelection || null,
        editorRole,
        hasText,
      });
      return;
    }

    // Monaco editors render as divs; detect and adapt.
    const monacoRoot = el?.closest?.('[data-darube-monaco="1"]') || null;
    const editor = monacoRoot?.__darubeMonacoEditor || null;
    if (!editor) return;

    const model = editor.getModel?.();
    const selections = editor.getSelections?.() || (editor.getSelection ? [editor.getSelection()] : []);
    const hasSelection = (selections || []).some((s) => {
      if (!s) return false;
      if (typeof s.isEmpty === 'function') return !s.isEmpty();
      const sln = s.startLineNumber ?? s.selectionStartLineNumber;
      const sc = s.startColumn ?? s.selectionStartColumn;
      const eln = s.endLineNumber ?? s.positionLineNumber ?? s.selectionEndLineNumber;
      const ec = s.endColumn ?? s.positionColumn ?? s.selectionEndColumn;
      if (sln == null || sc == null || eln == null || ec == null) return false;
      return (sln !== eln) || (sc !== ec);
    });

    const editorRole = monacoRoot?.dataset?.darubeEditorRole || null;
    const hasText = !!String(model?.getValue?.() || '').trim();
    const readOnly = (monacoRoot?.dataset?.darubeReadOnly === '1');

    showMenu(e, 'text', {
      kind: 'monaco',
      editor,
      hasSelection,
      readOnly,
      boxSelection: null,
      editorRole,
      hasText,
    });
  };
  const handleResultsContextMenu = (e, data) => showMenu(e, 'results', data);

  // Returns which action to take; caller must wire the actual handlers
  const handleMenuAction = (action, { onConnAction, onTableAction, onTextAction, onResultsAction } = {}) => {
    const { type, data } = contextMenu;
    hideMenu();
    if (type === 'connection') onConnAction?.(action, data);
    if (type === 'table') onTableAction?.(action, data);
    if (type === 'text') onTextAction?.(action, data);
    if (type === 'results') onResultsAction?.(action, data);
  };

  return {
    contextMenu, setContextMenu, hideMenu,
    handleConnectionContextMenu, handleTableContextMenu,
    handleTextContextMenu, handleResultsContextMenu,
    handleMenuAction,
  };
}
