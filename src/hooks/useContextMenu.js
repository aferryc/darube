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
    const isTextField = el && (el.tagName === 'TEXTAREA' || (el.tagName === 'INPUT' && (el.type === 'text' || el.type === 'search' || el.type === 'password' || el.type === 'email' || el.type === 'url' || el.type === 'tel')));
    if (!isTextField) return;
    const start = el.selectionStart ?? 0;
    const end = el.selectionEnd ?? 0;
    const hasBoxSelection = !!el?.dataset?.boxSelection;
    const hasSelection = hasBoxSelection || (end > start);
    const editorRole = el?.dataset?.darubeEditorRole || null;
    const hasText = !!String(el?.value || '').trim();
    showMenu(e, 'text', {
      el,
      hasSelection,
      readOnly: !!el.readOnly || !!el.disabled,
      boxSelection: el?.dataset?.boxSelection || null,
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
