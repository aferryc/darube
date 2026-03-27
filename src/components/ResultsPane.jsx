import iconExport from '../assets/file-export.svg';
import { VisualPlan } from './VisualPlan';

export function ResultsPane({
  activeTab, layoutDirection, loading,
  editingCell, setEditingCell,
  updateActiveTab,
  undoMutation, redoMutation, cancelMutations, saveMutations,
  handleCellDoubleClick, handleCellBlur, handleRowAction,
  handleRowClick, handleRowContextMenu,
  handleExportClick,
  onRefreshTableSizes,
  computeWorkingData,
}) {
  const results = activeTab.results;
  const plan = activeTab.plan;
  const tableSizesUpdatedAt = results?.meta?.updated_at || '';

  const renderDataGrid = () => {
    if (!results?.success || !results.columns) return null;
    const workingData = computeWorkingData();
    const totalRows = workingData.length;
    const totalPages = Math.ceil(totalRows / activeTab.rowsPerPage);
    const startIndex = (activeTab.currentPage - 1) * activeTab.rowsPerPage;
    const paginatedRows = workingData.slice(startIndex, startIndex + activeTab.rowsPerPage);

    return (
      <div className="flex-grow flex-column min-h-0">
        {activeTab.historyIndex > -1 && (
          <div className="mutation-action-bar">
            <span className="mutation-count">{activeTab.historyIndex + 1} unsaved edits</span>
            <button className="secondary" onClick={undoMutation} disabled={activeTab.historyIndex < 0}>↩ Undo</button>
            <button className="secondary" onClick={redoMutation} disabled={activeTab.historyIndex >= activeTab.history.length - 1}>↪ Redo</button>
            <button className="secondary" onClick={cancelMutations}>Cancel</button>
            <button onClick={saveMutations} disabled={loading} className="flex-grow-none ml-auto">Make Changes</button>
          </div>
        )}

        <div className="results-table-wrapper">
          <table className="results-table">
            <thead>
              <tr>
                {activeTab.targetTable && <th className="row-actions-cell" />}
                {results.columns.map((col, i) => <th key={i}>{col}</th>)}
              </tr>
            </thead>
            <tbody>
              {paginatedRows.map((row, i) => {
                const absoluteIndex = startIndex + i;
                const isSelected = activeTab.selectedRows?.includes(absoluteIndex);
                const uiId = row._ui_id;
                return (
                  <tr
                    key={uiId}
                    className={`${isSelected ? 'selected-row' : ''} ${row._isInserted ? 'inserted-row' : ''}`}
                    onClick={(e) => handleRowClick(e, absoluteIndex)}
                    onContextMenu={(e) => handleRowContextMenu(e, absoluteIndex)}
                  >
                    {activeTab.targetTable && (
                      <td className="row-actions-cell" onClick={e => e.stopPropagation()}>
                        <div className="row-actions-inline">
                          <span title="Duplicate row" onClick={() => handleRowAction('duplicate', row)}>⧉</span>
                          <span title="Delete row" className="text-danger" onClick={() => handleRowAction('delete', row)}>×</span>
                        </div>
                      </td>
                    )}
                    {results.columns.map((col, j) => {
                      const cellValue = row[j];
                      const isMutated = row._mutatedCols?.[col] || row._isInserted;
                      const isEditing = editingCell.uiId === uiId && editingCell.colName === col;
                      return (
                        <td
                          key={`${uiId}-${col}`}
                          className={isMutated ? 'mutated-cell' : ''}
                          onDoubleClick={() => handleCellDoubleClick(uiId, col, cellValue)}
                          onContextMenu={(e) => {
                            e.stopPropagation();
                            handleRowContextMenu(e, absoluteIndex, { colIndex: j, colName: col, cellValue });
                          }}
                        >
                          {isEditing ? (
                            <input
                              autoFocus
                              className="cell-editor-input"
                              value={editingCell.tempValue}
                              onChange={e => setEditingCell({ ...editingCell, tempValue: e.target.value })}
                              onBlur={() => handleCellBlur(uiId, col, cellValue, row)}
                              onKeyDown={e => { if (e.key === 'Enter' || e.key === 'Escape') e.target.blur(); }}
                            />
                          ) : String(cellValue ?? '')}
                        </td>
                      );
                    })}
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>

        {totalRows > 0 && (
          <div className="pagination-bar">
            <div>Showing {startIndex + 1} to {Math.min(startIndex + activeTab.rowsPerPage, totalRows)} of {totalRows} rows</div>
            <div className="pagination-controls">
              <button className="secondary" disabled={activeTab.currentPage === 1} onClick={() => updateActiveTab({ currentPage: 1 })}>«</button>
              <button className="secondary" disabled={activeTab.currentPage === 1} onClick={() => updateActiveTab({ currentPage: activeTab.currentPage - 1 })}>‹</button>
              <span>Page {activeTab.currentPage} of {totalPages}</span>
              <button className="secondary" disabled={activeTab.currentPage === totalPages} onClick={() => updateActiveTab({ currentPage: activeTab.currentPage + 1 })}>›</button>
              <button className="secondary" disabled={activeTab.currentPage === totalPages} onClick={() => updateActiveTab({ currentPage: totalPages })}>»</button>
              <select
                style={{ marginLeft: '10px', padding: '4px', background: 'var(--bg-dark)', color: 'white', border: '1px solid var(--border)' }}
                value={activeTab.rowsPerPage}
                onChange={e => updateActiveTab({ rowsPerPage: parseInt(e.target.value), currentPage: 1 })}
              >
                <option value={50}>50 / page</option>
                <option value={100}>100 / page</option>
                <option value={500}>500 / page</option>
              </select>
            </div>
          </div>
        )}
      </div>
    );
  };

  return (
    <div className={`pane results-section layout-${layoutDirection}`}>
      {/* View toggle bar */}
      {activeTab.type === 'query' && (results?.columns || plan) && (
        <div className="results-header-bar">
          <div className="results-view-tabs">
            <div onClick={() => updateActiveTab({ activeView: 'results' })} className={`results-view-tab ${activeTab.activeView !== 'plan' ? 'active' : ''}`}>
              Data Results
            </div>
            {plan && (
              <div onClick={() => updateActiveTab({ activeView: 'plan' })} className={`results-view-tab ${activeTab.activeView === 'plan' ? 'active' : ''}`}>
                Execution Plan
              </div>
            )}
          </div>
          {activeTab.activeView !== 'plan' && results?.success && results?.columns && (
            <button className="secondary btn-icon export-btn" onClick={() => handleExportClick('query', activeTab.name, activeTab.query)} disabled={loading}>
              <img src={iconExport} className="icon-sm icon-light" alt="Export" /> Export
            </button>
          )}
        </div>
      )}
      {activeTab.type === 'indexes' && results?.columns && (
        <div className="special-tab-header">
          <span className="special-tab-title">Table Indexes</span>
        </div>
      )}
      {activeTab.type === 'table_sizes' && results?.columns && (
        <div className="special-tab-header">
          <span className="special-tab-title">Table Size Cache</span>
          <div className="special-tab-actions">
            {tableSizesUpdatedAt && (
              <span className="special-tab-meta">
                Updated: {new Date(tableSizesUpdatedAt).toLocaleString()}
              </span>
            )}
            <button
              className="secondary"
              onClick={() => onRefreshTableSizes?.(activeTab.connectionId)}
              disabled={loading}
            >
              Refresh
            </button>
          </div>
        </div>
      )}
      {activeTab.type === 'dml' && (
        <div className="special-tab-header">
          <span className="special-tab-title">Database DML Structure</span>
        </div>
      )}

      {activeTab.type === 'dml' ? (
        <div className="dml-viewer">
          {results?.dml}
        </div>
      ) : activeTab.activeView === 'plan' ? (
        <VisualPlan plan={plan} />
      ) : (
        <>
          {results?.success === false && (
            <div className="execution-error">
              <h4>Execution Error</h4>
              <p className="error-message">{results.error}</p>
            </div>
          )}
          {results?.success && results.rows_affected !== undefined && (
            <div className="execution-success">
              <h4>Success</h4>
              <p>{results.message}</p>
            </div>
          )}
          {renderDataGrid()}
        </>
      )}
    </div>
  );
}
