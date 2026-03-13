import React, { useState, useEffect, useRef } from "react"
import logoApp from '../assets/darube.png';
import iconAdd from '../assets/add.svg';
import iconAngleDown from '../assets/angle-down.svg';
import iconAngleRight from '../assets/angle-right.svg';
import iconDatabase from '../assets/database.svg';
import iconDelete from '../assets/delete.svg';
import iconMysql from '../assets/mysql.svg';
import iconOracle from '../assets/oracle.svg';
import iconPencil from '../assets/pencil.svg';
import iconPostgres from '../assets/postgre.svg';
import iconQuestion from '../assets/question.svg';
import iconRefresh from '../assets/rotate-right.svg';
import iconRedis from '../assets/redis.svg';
import iconSqlite from '../assets/sqlite.svg';
import iconSqlServer from '../assets/sql-server.svg';
import { groupTables } from '../utils/tableUtils';

function TableNode({ tbl, schemaKey, schemaName, cId, expandedTree, toggleTree, handleTableContextMenu }) {
  const tblKey = `${schemaKey}:${tbl.name}`;
  const isTblOpen = expandedTree[tblKey];
  return (
    <div key={tbl.name}>
      <div
        className="metadata-node"
        onClick={(e) => toggleTree(e, tblKey, 'table', cId, schemaName, tbl.name)}
        onContextMenu={(e) => handleTableContextMenu(e, tbl, schemaName, cId)}
        title={tbl.indexes?.length > 0 ? 'Indexes: ' + tbl.indexes.join(', ') : 'No Indexes'}
      >
        <img src={isTblOpen ? iconAngleDown : iconAngleRight} className="icon-sm icon-light" alt="Toggle" />
        {tbl.type === 'view' ? '👁️' : '📄'} {tbl.name}
      </div>
      {isTblOpen && tbl.columns && (
        <div className="metadata-node table-columns">
          {!tbl.columnsLoaded && <div className="metadata-node" style={{ color: 'var(--text-muted)' }}>Loading...</div>}
          {tbl.columnsLoaded && tbl.columns.length === 0 && <div className="metadata-node" style={{ color: 'var(--text-muted)', fontStyle: 'italic' }}>(Empty)</div>}
          {tbl.columns.map(col => (
            <div key={col.name} className="metadata-node" style={{ color: 'var(--text-muted)' }}>
              <span style={{ color: 'var(--accent)' }}>♦</span> {col.name} <i>({col.type})</i>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function ConnectionItem({ c, activeId, expandedConns, expandedTree, metadata, expandedFolders,
  draggedConnId, handleConnectionClick, handleEditConnection, handleDeleteConnection,
  handleDisconnect, handleReconnect, fetchConnections, handleConnectionContextMenu,
  handleTableContextMenu, toggleTree, setDraggedConnId, setDragOverFolderId, inFolder }) {
  return (
    <div
      key={c.id}
      draggable
      onDragStart={(e) => { e.stopPropagation(); setDraggedConnId(c.id); }}
      onDragEnd={() => { setDraggedConnId(null); setDragOverFolderId(null); }}
      onContextMenu={(e) => handleConnectionContextMenu(e, c)}
      className={draggedConnId === c.id ? 'dragged-item' : ''}
    >
      <div
        className={`connection-item ${activeId === c.id ? 'active' : ''} ${inFolder ? 'in-folder' : ''}`}
        onClick={() => handleConnectionClick(c.id)}
      >
        <div className="connection-item-header">
          <span className={`status-dot ${c.status}`} title={c.status} />
          {c.db_type === 'mysql'     && <img src={iconMysql}     className="icon icon-light" alt="MySQL" />}
          {c.db_type === 'oracle'   && <img src={iconOracle}    className="icon icon-light" alt="Oracle" />}
          {c.db_type === 'postgres'  && <img src={iconPostgres}  className="icon icon-light" alt="PostgreSQL" />}
          {c.db_type === 'sqlite'   && <img src={iconSqlite}    className="icon icon-light" alt="SQLite" />}
          {c.db_type === 'sqlserver' && <img src={iconSqlServer} className="icon icon-light" alt="SQL Server" />}
          {c.db_type === 'redis'     && <img src={iconRedis}     className="icon icon-light" alt="Redis" />}
          <span style={{ fontWeight: 500 }}>{c.connection_name}</span>
        </div>
        <div className="connection-item-controls">
          <img src={iconPencil}  className="icon-sm icon-light" alt="Edit"    onClick={(e) => handleEditConnection(c, e)} title="Edit Connection" />
          <img src={iconDelete}  className="icon-sm icon-light" alt="Delete"  onClick={(e) => handleDeleteConnection(c.id, e)} title="Delete Connection" />
          <img src={iconRefresh} className="icon-sm icon-light" alt="Refresh" onClick={(e) => { e.stopPropagation(); fetchConnections(); }} title="Refresh" />
          <div className="flex-grow" />
          {c.status === 'connected'
            ? <button onClick={(e) => { e.stopPropagation(); handleDisconnect(c.id); }} className="secondary tab-connect-btn">Stop</button>
            : <button onClick={(e) => { e.stopPropagation(); handleReconnect(c.id); }} className="tab-connect-btn">Connect</button>
          }
        </div>
      </div>

      {c.db_type !== 'redis' && expandedConns[c.id] && metadata[c.id] && c.status === 'connected' && (
        <div className="metadata-tree">
            <div className="metadata-group">
              <div className="metadata-title">Databases</div>
              {metadata[c.id].databases.map(db => (
                <div key={db.name} className="metadata-node">
                  <img src={iconDatabase} className="icon-sm icon-light" alt="DB" /> {db.name}
                </div>
              ))}
            </div>
          {metadata[c.id].schemas?.length > 0 && (
            <div className="metadata-group">
              <div className="metadata-title" style={{ marginTop: '10px' }}>Schemas & Entities</div>
              {metadata[c.id].schemas.map(schema => {
                const schemaKey = `${c.id}:${schema.name}`;
                const isSchemaOpen = expandedTree[schemaKey];
                return (
                  <div key={schema.name}>
                    <div className="metadata-node" onClick={(e) => toggleTree(e, schemaKey, 'schema', c.id, schema.name)}>
                      <img src={isSchemaOpen ? iconAngleDown : iconAngleRight} className="icon-sm icon-light" alt="Toggle" />
                      📁 {schema.name} <span className="entity-count">({schema.tables?.length || 0})</span>
                    </div>
                    {isSchemaOpen && schema.tables && (() => {
                      const grouped = groupTables(schema.tables);
                      return (
                        <div className="metadata-node schema-tables">
                          {!schema.tablesLoaded && <div className="metadata-node" style={{ color: 'var(--text-muted)' }}>Loading...</div>}
                          {schema.tablesLoaded && schema.tables.length === 0 && <div className="metadata-node" style={{ color: 'var(--text-muted)', fontStyle: 'italic' }}>(Empty)</div>}
                          {grouped.map(item => {
                            if (item.isGroup) {
                              const groupKey = `${schemaKey}:group:${item.name}`;
                              const isGroupOpen = expandedTree[groupKey];
                              return (
                                <div key={item.name}>
                                  <div className="metadata-node" onClick={(e) => toggleTree(e, groupKey)}>
                                    <img src={isGroupOpen ? iconAngleDown : iconAngleRight} className="icon-sm icon-light" alt="Toggle" />
                                    🗂️ {item.name} <span className="entity-count">({item.tables.length})</span>
                                  </div>
                                  {isGroupOpen && (
                                    <div className="metadata-node schema-tables">
                                      {item.tables.map(tbl => <TableNode key={tbl.name} tbl={tbl} schemaKey={schemaKey} schemaName={schema.name} cId={c.id} expandedTree={expandedTree} toggleTree={toggleTree} handleTableContextMenu={handleTableContextMenu} />)}
                                    </div>
                                  )}
                                </div>
                              );
                            }
                            return <TableNode key={item.name} tbl={item} schemaKey={schemaKey} schemaName={schema.name} cId={c.id} expandedTree={expandedTree} toggleTree={toggleTree} handleTableContextMenu={handleTableContextMenu} />;
                          })}
                        </div>
                      );
                    })()}
                  </div>
                );
              })}
            </div>
          )}
          {!metadata[c.id].databases?.length && !metadata[c.id].schemas?.length && (
            <div className="metadata-node" style={{ color: 'var(--text-muted)' }}>No schemas found.</div>
          )}
        </div>
      )}
    </div>
  );
}

export function Sidebar({
  connections, folders, metadata, activeId,
  expandedConns, expandedTree, expandedFolders, setExpandedFolders,
  draggedConnId, dragOverFolderId, setDraggedConnId, setDragOverFolderId,
  editingFolderId, folderEditName, setFolderEditName,
  showNewFolderInput, newFolderName, setNewFolderName,
  sidebarCollapsed, setSidebarCollapsed,
  handleConnectionClick, handleEditConnection, handleDeleteConnection,
  handleDisconnect, handleReconnect, fetchConnections,
  handleConnectionContextMenu, handleTableContextMenu, toggleTree,
  handleCreateFolder, handleSubmitNewFolder,
  handleRenameFolder, handleSubmitRenameFolder, handleDeleteFolder,
  handleDropOnFolder, onNewConnection, onShowHelp, onNewScript,
}) {
  const connItemProps = {
    activeId, expandedConns, expandedTree, metadata, expandedFolders,
    draggedConnId, handleConnectionClick, handleEditConnection, handleDeleteConnection,
    handleDisconnect, handleReconnect, fetchConnections,
    handleConnectionContextMenu, handleTableContextMenu, toggleTree,
    setDraggedConnId, setDragOverFolderId,
  };


  return (
   <div className={`sidebar-wrapper ${sidebarCollapsed ? 'collapsed' : ''}`}>
      <div className="sidebar">
        {sidebarCollapsed ? (
          <div className="sidebar-icons-only">
            <div className="sidebar-icon-logo" title="Connections">
              <img src={logoApp} alt="Darube" style={{ height: '22px', opacity: 0.85 }} />
            </div>
            {connections.map(c => (
              <div key={c.id} className={`sidebar-icon-conn ${activeId === c.id ? 'active' : ''}`} onClick={() => handleConnectionClick(c.id)} title={`${c.connection_name} (${c.db_type}) — ${c.status}`}>
                <span className={`status-dot ${c.status}`} />
                {c.db_type === 'mysql'     && <img src={iconMysql}     className="icon icon-light" alt="MySQL" />}
                {c.db_type === 'oracle'   && <img src={iconOracle}    className="icon icon-light" alt="Oracle" />}
                {c.db_type === 'postgres'  && <img src={iconPostgres}  className="icon icon-light" alt="PostgreSQL" />}
                {c.db_type === 'sqlite'   && <img src={iconSqlite}    className="icon icon-light" alt="SQLite" />}
                {c.db_type === 'sqlserver' && <img src={iconSqlServer} className="icon icon-light" alt="SQL Server" />}
                {c.db_type === 'redis'     && <img src={iconRedis}     className="icon icon-light" alt="Redis" />}
              </div>
            ))}
          </div>
        ) : (
          <>
            <div className="sidebar-header">
              <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                <span>Connections</span>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
                <button onClick={onShowHelp} className="btn-icon" style={{ padding: '5px' }} title="Help"><img src={iconQuestion} className="icon-sm icon-light" alt="Help" /></button>
                <button onClick={onNewConnection} className="btn-icon" style={{ padding: '5px' }}><img src={iconAdd} className="icon-sm icon-light" alt="Add" /></button>
              </div>
            </div>

            <div className="sidebar-content">
              {showNewFolderInput ? (
                <div className="new-folder-input-row">
                  <input autoFocus value={newFolderName} onChange={e => setNewFolderName(e.target.value)}
                    onKeyDown={e => { if (e.key === 'Enter') handleSubmitNewFolder(); if (e.key === 'Escape') { setNewFolderName(''); } }}
                    placeholder="Folder name..." style={{ flex: 1, fontSize: '12px', padding: '4px 8px', background: 'var(--bg-dark)', border: '1px solid var(--accent)', borderRadius: '4px', color: 'white', outline: 'none' }} />
                  <button onClick={handleSubmitNewFolder} style={{ fontSize: '11px', padding: '3px 8px' }}>Add</button>
                  <button onClick={() => setNewFolderName('')} className="secondary" style={{ fontSize: '11px', padding: '3px 6px' }}>✕</button>
                </div>
              ) : (
                <button className="new-folder-btn" onClick={handleCreateFolder}>+ New Folder</button>
              )}

              <button className="new-script-btn" onClick={onNewScript}>{"</>"} New Script</button>

              {folders.map(folder => {
                const folderConns = connections.filter(c => c.folder_id === folder.id);
                const isOpen = expandedFolders[folder.id] !== false;
                const isRenaming = editingFolderId === folder.id;
                const isDragTarget = dragOverFolderId === folder.id;
                return (
                  <div key={folder.id} className="sidebar-folder"
                    onDragOver={(e) => { e.preventDefault(); setDragOverFolderId(folder.id); }}
                    onDragLeave={() => setDragOverFolderId(null)}
                    onDrop={(e) => { e.preventDefault(); handleDropOnFolder(folder.id); }}
                    style={isDragTarget ? { background: 'rgba(79,140,255,0.12)', borderLeft: '2px solid var(--accent)', borderRadius: '4px' } : {}}
                  >
                    <div className="sidebar-folder-header" onClick={() => !isRenaming && setExpandedFolders(prev => ({ ...prev, [folder.id]: !isOpen }))}>
                      <span style={{ fontSize: '12px', opacity: 0.7 }}>{isOpen ? '▾' : '▸'}</span>
                      <span style={{ fontSize: '13px' }}>📁</span>
                      {isRenaming ? (
                        <input autoFocus value={folderEditName} onChange={e => setFolderEditName(e.target.value)}
                          onKeyDown={e => { if (e.key === 'Enter') handleSubmitRenameFolder(folder.id); if (e.key === 'Escape') { setFolderEditName(''); } }}
                          onClick={e => e.stopPropagation()}
                          style={{ flex: 1, fontSize: '12px', padding: '2px 6px', background: 'var(--bg-dark)', border: '1px solid var(--accent)', borderRadius: '4px', color: 'white', outline: 'none', marginRight: '4px' }} />
                      ) : (
                        <span style={{ flex: 1, fontWeight: 600, fontSize: '12px', color: isDragTarget ? 'var(--accent)' : 'var(--text-muted)', letterSpacing: '0.05em', textTransform: 'uppercase' }}>{folder.name}</span>
                      )}
                      <span style={{ fontSize: '10px', opacity: 0.5, marginRight: '4px' }}>{folderConns.length}</span>
                      <img src={iconPencil} className="icon-sm icon-light" alt="Rename" style={{ opacity: 0.6, cursor: 'pointer', marginRight: '4px' }} onClick={(e) => handleRenameFolder(folder.id, folder.name, e)} title="Rename Folder" />
                      <img src={iconDelete} className="icon-sm icon-light" alt="Delete" style={{ opacity: 0.6, cursor: 'pointer' }} onClick={(e) => handleDeleteFolder(folder.id, e)} title="Delete Folder" />
                    </div>
                    {isOpen && folderConns.map(c => <ConnectionItem key={c.id} c={c} {...connItemProps} inFolder />)}
                  </div>
                );
              })}

              <div className="sidebar-uncategorized-dropzone"
                onDragOver={(e) => { e.preventDefault(); setDragOverFolderId('__uncategorized__'); }}
                onDragLeave={() => setDragOverFolderId(null)}
                onDrop={(e) => { e.preventDefault(); handleDropOnFolder(''); }}
                style={{ flex: 1, minHeight: '100px', display: 'flex', flexDirection: 'column', ...(dragOverFolderId === '__uncategorized__' ? { background: 'rgba(255,255,255,0.04)', borderLeft: '2px solid var(--border)', borderRadius: '4px' } : {}) }}
              >
                {connections.filter(c => !c.folder_id || !folders.find(f => f.id === c.folder_id)).map(c => (
                  <ConnectionItem key={c.id} c={c} {...connItemProps} inFolder={false} />
                ))}
                {draggedConnId && dragOverFolderId === '__uncategorized__' && (
                  <div style={{ padding: '8px 14px', fontSize: '11px', color: 'var(--text-muted)', fontStyle: 'italic' }}>Drop here to remove from folder</div>
                )}
              </div>
            </div>
          </>
        )}
      </div>
      <button className="sidebar-toggle-btn" onClick={() => setSidebarCollapsed(prev => !prev)} title={sidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}>
        {sidebarCollapsed ? '›' : '‹'}
      </button>
    </div>
  );
}
