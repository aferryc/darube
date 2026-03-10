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

  // Returns which action to take; caller must wire the actual handlers
  const handleMenuAction = (action, { onConnAction, onTableAction } = {}) => {
    const { type, data } = contextMenu;
    hideMenu();
    if (type === 'connection') onConnAction?.(action, data);
    if (type === 'table') onTableAction?.(action, data);
  };

  return {
    contextMenu, setContextMenu, hideMenu,
    handleConnectionContextMenu, handleTableContextMenu, handleMenuAction,
  };
}
