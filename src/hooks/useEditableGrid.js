import { useState } from 'react';
import { computeWorkingData as computeWorkingDataUtil, consolidateMutations } from '../utils/mutationUtils';

export function useEditableGrid(apiUrl, activeId, activeTab, updateActiveTab, executeQuery, setLoading) {
  const [editingCell, setEditingCell] = useState({ uiId: null, colName: null, tempValue: '' });

  const applyMutation = (action) => {
    if (!activeTab.targetTable) return;
    const newHistory = activeTab.history.slice(0, activeTab.historyIndex + 1);
    newHistory.push(action);
    updateActiveTab({ history: newHistory, historyIndex: newHistory.length - 1 });
  };

  const undoMutation = () => {
    if (activeTab.historyIndex >= 0)
      updateActiveTab({ historyIndex: activeTab.historyIndex - 1 });
  };

  const redoMutation = () => {
    if (activeTab.historyIndex < activeTab.history.length - 1)
      updateActiveTab({ historyIndex: activeTab.historyIndex + 1 });
  };

  const cancelMutations = () => {
    if (window.confirm('Discard all unsaved changes in this tab?'))
      updateActiveTab({ history: [], historyIndex: -1 });
  };

  const handleCellDoubleClick = (uiId, colName, value) => {
    if (!activeTab.targetTable) return;
    setEditingCell({ uiId, colName, tempValue: value ?? '' });
  };

  const handleCellBlur = (uiId, colName, originalValue, originalRow) => {
    const newVal = editingCell.tempValue;
    setEditingCell({ uiId: null, colName: null, tempValue: '' });

    if (newVal !== String(originalValue ?? '')) {
      const dbOrig = activeTab.results.rows.find(r => r._ui_id === uiId) || originalRow;
      const cleanOrig = {};
      activeTab.results.columns.forEach((c, idx) => { cleanOrig[c] = dbOrig[idx]; });
      applyMutation({
        type: 'update', timestamp: Date.now(), uiId, colName,
        originalValue, newValue: newVal, originalRow: cleanOrig,
        newValues: { [colName]: newVal },
      });
    }
  };

  const handleRowAction = (actionType, rowData) => {
    if (!activeTab.targetTable) return;
    const cleanOrig = {};
    activeTab.results.columns.forEach((c, idx) => { cleanOrig[c] = rowData[idx]; });

    if (actionType === 'delete') {
      applyMutation({ type: 'delete', timestamp: Date.now(), uiId: rowData._ui_id, originalRow: cleanOrig });
    } else if (actionType === 'duplicate') {
      applyMutation({ type: 'insert', timestamp: Date.now(), uiId: Math.random().toString(36).substr(2, 9), newValues: cleanOrig });
    } else if (actionType === 'insert_blank') {
      const emptyRow = {};
      activeTab.results.columns.forEach(c => { emptyRow[c] = null; });
      applyMutation({ type: 'insert', timestamp: Date.now(), uiId: Math.random().toString(36).substr(2, 9), newValues: emptyRow });
    }
  };

  const computeWorkingData = () => {
    if (!activeTab.results?.rows) return [];
    return computeWorkingDataUtil(
      activeTab.results.rows,
      activeTab.results.columns,
      activeTab.history,
      activeTab.historyIndex,
    );
  };

  const saveMutations = async () => {
    if (!activeId || !activeTab.targetTable) return;
    const activeHistory = activeTab.history.slice(0, activeTab.historyIndex + 1);
    if (activeHistory.length === 0) return;

    const consolidated = consolidateMutations(activeHistory);
    const stats = consolidated.reduce((acc, curr) => { acc[curr.type] = (acc[curr.type] || 0) + 1; return acc; }, {});

    if (!window.confirm(
      `Save changes to ${activeTab.targetTable}?\n\nUpdates: ${stats.update || 0}\nInserts: ${stats.insert || 0}\nDeletes: ${stats.delete || 0}`
    )) return;

    setLoading(true);
    try {
      const res = await fetch(`${apiUrl}/api/connections/${activeId}/mutate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ table: activeTab.targetTable, mutations: consolidated }),
      });
      const data = await res.json();
      if (data.success) {
        updateActiveTab({ history: [], historyIndex: -1 });
        executeQuery(activeTab.query);
      } else {
        alert('Save failed:\n' + data.error);
      }
    } catch (err) {
      alert('Network error: ' + err.message);
    } finally {
      setLoading(false);
    }
  };

  return {
    editingCell, setEditingCell,
    undoMutation, redoMutation, cancelMutations,
    handleCellDoubleClick, handleCellBlur, handleRowAction,
    computeWorkingData, saveMutations,
  };
}
