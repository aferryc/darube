import { useState, useEffect, useRef } from 'react';

export function useConnections(apiUrl) {
  const [connections, setConnections] = useState([]);
  const [folders, setFolders] = useState([]);
  const [expandedConns, setExpandedConns] = useState({});
  const [expandedTree, setExpandedTree] = useState({});
  const [metadata, setMetadata] = useState({});
  const [tableSizes, setTableSizes] = useState({});
  const [expandedFolders, setExpandedFolders] = useState({});
  const tableSizesLoadingRef = useRef(new Set());

  // Folder form state
  const [editingFolderId, setEditingFolderId] = useState(null);
  const [folderEditName, setFolderEditName] = useState('');
  const [showNewFolderInput, setShowNewFolderInput] = useState(false);
  const [newFolderName, setNewFolderName] = useState('');

  // Drag-and-drop
  const [draggedConnId, setDraggedConnId] = useState(null);
  const [dragOverFolderId, setDragOverFolderId] = useState(null);

  const fetchConnections = async () => {
    try {
      const res = await fetch(`${apiUrl}/api/connections`);
      const data = await res.json();
      if (data.connections) setConnections(data.connections);
    } catch (e) {
      console.error('Engine not ready or error:', e);
    }
  };

  const fetchFolders = async () => {
    try {
      const res = await fetch(`${apiUrl}/api/folders`);
      const data = await res.json();
      if (data.success) setFolders(data.folders || []);
    } catch (e) {
      console.error('Failed to load folders:', e);
    }
  };

  const fetchMetadata = async (id) => {
    try {
      const conn = connections.find(c => c.id === id);
      // Non-SQL connections don't have SQL metadata endpoints in the engine.
      if (conn?.db_type === 'redis' || conn?.db_type === 'http' || conn?.db_type === 'grpc') {
        setMetadata(prev => ({
          ...prev,
          [id]: { databases: [], schemas: [] },
        }));
        return;
      }
      const [dbRes, entRes] = await Promise.all([
        fetch(`${apiUrl}/api/connections/${id}/metadata/databases`),
        fetch(`${apiUrl}/api/connections/${id}/metadata/schemas`),
      ]);
      const dbData = await dbRes.json();
      const entData = await entRes.json();
      setMetadata(prev => ({
        ...prev,
        [id]: {
          databases: dbData.success ? dbData.databases : [],
          schemas: entData.success ? entData.schemas : [],
        },
      }));
    } catch (err) {
      console.error('Failed to load metadata', err);
    }
  };

  const normalizeTableKey = (schema, table) => {
    const s = String(schema || '').trim().toLowerCase();
    const t = String(table || '').trim().toLowerCase();
    if (!s) return t;
    return `${s}.${t}`;
  };

  const fetchTableSizes = async (id) => {
    try {
      const res = await fetch(`${apiUrl}/api/connections/${id}/table-sizes`);
      const data = await res.json();
      if (!data?.success) return;

      const sizes = Array.isArray(data.sizes) ? data.sizes : [];
      const byKey = {};
      for (const s of sizes) {
        const key = normalizeTableKey(s.schema, s.table);
        if (!key) continue;
        byKey[key] = s.size_bytes;
      }
      setTableSizes(prev => ({
        ...prev,
        [id]: { byKey, updated_at: data.updated_at || '' },
      }));
    } catch (err) {
      console.error('Failed to fetch table sizes', err);
    }
  };

  const ensureTableSizesLoaded = async (id) => {
    if (!id) return;
    if (tableSizesLoadingRef.current.has(id)) return;
    if (tableSizes[id]?.byKey) return;
    tableSizesLoadingRef.current.add(id);
    try {
      const res = await fetch(`${apiUrl}/api/connections/${id}/table-sizes/status`);
      const data = await res.json().catch(() => ({}));
      if (!res.ok || !data?.success) return;
      const status = data.status || (typeof data.count === 'number' && data.count > 0 ? 'ready' : 'idle');
      if (status === 'ready') {
        await fetchTableSizes(id);
      }
    } catch (err) {
      console.error('Failed to check table size status', err);
    } finally {
      tableSizesLoadingRef.current.delete(id);
    }
  };

  // Auto-expand connected connections
  useEffect(() => {
    let toFetch = false;
    let newExpanded = null;
    connections.forEach(c => {
      if (c.db_type !== 'redis' && c.db_type !== 'http' && c.db_type !== 'grpc' && c.status === 'connected' && !metadata[c.id] && expandedConns[c.id] !== true) {
        if (!newExpanded) newExpanded = { ...expandedConns };
        newExpanded[c.id] = true;
        toFetch = true;
        fetchMetadata(c.id);
      }
    });
    if (toFetch) setExpandedConns(prev => ({ ...prev, ...newExpanded }));
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [connections, metadata, expandedConns]);

  // When a connection is expanded & connected, try to load table sizes (once cache is ready).
  useEffect(() => {
    connections.forEach(c => {
      const isSQL = c.db_type !== 'redis' && c.db_type !== 'http' && c.db_type !== 'grpc';
      if (!isSQL) return;
      if (c.status !== 'connected') return;
      if (!expandedConns[c.id]) return;
      // Only useful once we have schemas/tables visible.
      if (!metadata[c.id]) return;
      ensureTableSizesLoaded(c.id);
    });
    // Prune entries for disconnected connections.
    setTableSizes(prev => {
      const connected = new Set(connections.filter(c => c.status === 'connected').map(c => c.id));
      let changed = false;
      const next = { ...prev };
      for (const id of Object.keys(next)) {
        if (!connected.has(id)) {
          delete next[id];
          changed = true;
        }
      }
      return changed ? next : prev;
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [connections, expandedConns, metadata]);

  const handleConnectionClick = async (id, forceExpand = false) => {
    const isExpanded = forceExpand || !expandedConns[id];
    setExpandedConns(prev => ({ ...prev, [id]: isExpanded }));
    if (isExpanded && !metadata[id]) await fetchMetadata(id);
    return id; // let caller do setActiveId
  };

  const handleDisconnect = async (id, activeId, setActiveId) => {
    const conn = connections.find(c => c.id === id);
    if (conn?.db_type === 'http' || conn?.db_type === 'grpc') return;
    const base = conn?.db_type === 'redis' ? '/api/redis' : '/api/connections';
    await fetch(`${apiUrl}${base}/${id}/disconnect`, { method: 'POST' });
    if (activeId === id) setActiveId(null);
    fetchConnections();
  };

  const handleReconnect = async (id) => {
    const conn = connections.find(c => c.id === id);
    if (conn?.db_type === 'http' || conn?.db_type === 'grpc') return;
    const url = conn?.db_type === 'redis' ? `${apiUrl}/api/redis/reconnect` : `${apiUrl}/api/connections/connect`;
    await fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id }),
    });
    fetchConnections();
  };

  const handleDeleteConnection = async (id, activeId, setActiveId) => {
    if (!window.confirm('Are you sure you want to delete this connection?')) return;
    try {
      const conn = connections.find(c => c.id === id);
      const base =
        conn?.db_type === 'redis' ? '/api/redis'
          : (conn?.db_type === 'http' ? '/api/http'
            : (conn?.db_type === 'grpc' ? '/api/grpc' : '/api/connections'));
      const res = await fetch(`${apiUrl}${base}/${id}`, { method: 'DELETE' });
      const data = await res.json();
      if (data.success) {
        if (activeId === id) setActiveId(null);
        fetchConnections();
      } else {
        alert(data.error);
      }
    } catch (err) {
      alert('Failed to delete: ' + err.message);
    }
  };

  const handleDropOnFolder = async (targetFolderId) => {
    if (!draggedConnId) return;
    const conn = connections.find(c => c.id === draggedConnId);
    if (!conn) return;
    if ((conn.folder_id || '') === (targetFolderId || '')) {
      setDraggedConnId(null); setDragOverFolderId(null); return;
    }
    try {
      const base =
        conn.db_type === 'redis' ? '/api/redis'
          : (conn.db_type === 'http' ? '/api/http'
            : (conn.db_type === 'grpc' ? '/api/grpc' : '/api/connections'));
      await fetch(`${apiUrl}${base}/${draggedConnId}/folder`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ folder_id: targetFolderId || '' }),
      });
      fetchConnections();
    } catch (err) {
      console.error('Error moving connection:', err);
    } finally {
      setDraggedConnId(null); setDragOverFolderId(null);
    }
  };

  const handleCreateFolder = () => { setNewFolderName(''); setShowNewFolderInput(true); };

  const handleSubmitNewFolder = async () => {
    if (!newFolderName.trim()) { setShowNewFolderInput(false); return; }
    try {
      const res = await fetch(`${apiUrl}/api/folders`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: newFolderName.trim() }),
      });
      const data = await res.json();
      if (data.success) {
        fetchFolders();
        setExpandedFolders(prev => ({ ...prev, [data.folder.id]: true }));
      }
    } catch (err) {
      console.error('Error creating folder:', err);
    } finally {
      setShowNewFolderInput(false); setNewFolderName('');
    }
  };

  const handleRenameFolder = (folderId, currentName, e) => {
    e.stopPropagation();
    setEditingFolderId(folderId);
    setFolderEditName(currentName);
  };

  const handleSubmitRenameFolder = async (folderId) => {
    if (folderEditName.trim()) {
      try {
        await fetch(`${apiUrl}/api/folders/${folderId}`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ name: folderEditName.trim() }),
        });
        fetchFolders();
      } catch (err) { console.error('Error renaming folder:', err); }
    }
    setEditingFolderId(null); setFolderEditName('');
  };

  const handleDeleteFolder = async (folderId, e) => {
    e.stopPropagation();
    if (!window.confirm('Delete this folder? Connections inside will become uncategorized.')) return;
    try {
      await fetch(`${apiUrl}/api/folders/${folderId}`, { method: 'DELETE' });
      fetchFolders(); fetchConnections();
    } catch (err) { console.error('Error deleting folder:', err); }
  };

  const toggleTree = async (e, key, type, id, schemaName, tableName) => {
    e.stopPropagation();
    const isExpanding = !expandedTree[key];
    setExpandedTree(prev => ({ ...prev, [key]: isExpanding }));

    if (!isExpanding || !type || !id || !schemaName) return;

    if (type === 'schema') {
      const schema = metadata[id]?.schemas?.find(s => s.name === schemaName);
      if (schema && !schema.tablesLoaded) {
        try {
          const res = await fetch(`${apiUrl}/api/connections/${id}/metadata/schemas/${schemaName}/tables`);
          const data = await res.json();
          if (data.success) {
            setMetadata(prev => {
              const m = { ...prev };
              if (!m[id]?.schemas) return prev;
              const schemas = [...m[id].schemas];
              const si = schemas.findIndex(s => s.name === schemaName);
              if (si > -1) schemas[si] = { ...schemas[si], tables: data.tables || [], tablesLoaded: true };
              return { ...m, [id]: { ...m[id], schemas } };
            });
          }
        } catch (err) { console.error('Failed to fetch tables', err); }
      }
    } else if (type === 'table') {
      const schema = metadata[id]?.schemas?.find(s => s.name === schemaName);
      const table = schema?.tables?.find(t => t.name === tableName);
      if (table && !table.columnsLoaded) {
        try {
          const res = await fetch(`${apiUrl}/api/connections/${id}/metadata/schemas/${schemaName}/tables/${tableName}/columns`);
          const data = await res.json();
          if (data.success) {
            setMetadata(prev => {
              const m = { ...prev };
              if (!m[id]?.schemas) return prev;
              const schemas = [...m[id].schemas];
              const si = schemas.findIndex(s => s.name === schemaName);
              if (si > -1) {
                const tables = [...schemas[si].tables];
                const ti = tables.findIndex(t => t.name === tableName);
                if (ti > -1) tables[ti] = { ...tables[ti], columns: data.columns || [], columnsLoaded: true };
                schemas[si] = { ...schemas[si], tables };
                m[id] = { ...m[id], schemas };
              }
              return m;
            });
          }
        } catch (err) { console.error('Failed to fetch columns', err); }
      }
    }
  };

  return {
    connections, folders, metadata, tableSizes, expandedConns, expandedTree, expandedFolders,
    editingFolderId, folderEditName, showNewFolderInput, newFolderName,
    draggedConnId, dragOverFolderId,
    setExpandedFolders, setFolderEditName, setNewFolderName, setDraggedConnId, setDragOverFolderId,
    fetchConnections, fetchFolders,
    handleConnectionClick, handleDisconnect, handleReconnect, handleDeleteConnection,
    handleDropOnFolder, handleCreateFolder, handleSubmitNewFolder,
    handleRenameFolder, handleSubmitRenameFolder, handleDeleteFolder,
    toggleTree,
  };
}
