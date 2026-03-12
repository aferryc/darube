import { useState } from 'react';

export function useExport(apiUrl, activeId, activeTab, updateActiveTab, setLoading) {
  const [exportConfig, setExportConfig] = useState(null);

  const handleExportClick = (targetType, targetName, queryTarget) => {
    setExportConfig({
      targetType, targetName, queryTarget,
      format: 'csv', headers: true, path: '',
      filename: `${targetName.replace(/\s+/g, '_').toLowerCase()}_export`,
      dataPayload: null, columnsPayload: null,
    });
  };

  const handleRowContextMenu = (e, index) => {
    e.preventDefault();
    let selected = activeTab.selectedRows || [];
    if (!selected.includes(index)) {
      selected = [index];
      updateActiveTab({ selectedRows: selected, lastSelectedIndex: index });
    }
    const selectedData = selected.map(i => activeTab.results?.rows[i]).filter(Boolean);
    setExportConfig({
      targetType: 'data',
      targetName: `Selected Rows (${selected.length})`,
      queryTarget: activeTab.query,
      format: 'csv', headers: true, path: '',
      filename: 'selection_export',
      dataPayload: selectedData,
      columnsPayload: activeTab.results?.columns,
    });
  };

  const handleExportRedisResult = (redisResult, command) => {
    if (!redisResult) return;
    setExportConfig({
      targetType: 'redis',
      targetName: 'Redis Result',
      queryTarget: '',
      format: 'json',
      headers: true,
      path: '',
      filename: 'redis_export',
      dataPayload: null,
      columnsPayload: null,
      redisPayload: {
        data_type: redisResult.data_type,
        value: redisResult.value,
        command: command || '',
      },
    });
  };

  const handleSelectDirectory = async () => {
    if (window.darube?.openDirectory) {
      try {
        const path = await window.darube.openDirectory();
        if (path) setExportConfig(prev => ({ ...prev, path }));
      } catch (err) { console.error('IPC Error:', err); }
    } else {
      alert('Native directory picker is only available in the Electron desktop environment.');
    }
  };

  const handleExecuteExport = async (e) => {
    e.preventDefault();
    if (!exportConfig.path || !exportConfig.filename) return alert('Please specify a directory and filename.');
    setLoading(true);
    try {
      const isRedis = exportConfig.targetType === 'redis';
      const url = isRedis
        ? `${apiUrl}/api/redis/${activeId}/export`
        : `${apiUrl}/api/connections/${activeId}/export`;

      const body = isRedis ? {
        format: exportConfig.format,
        headers: exportConfig.headers,
        destination_path: exportConfig.path,
        filename: exportConfig.filename,
        data_type: exportConfig.redisPayload?.data_type,
        value: exportConfig.redisPayload?.value,
        command: exportConfig.redisPayload?.command,
      } : {
        target_type: exportConfig.targetType,
        target: exportConfig.queryTarget || exportConfig.targetName,
        format: exportConfig.format,
        headers: exportConfig.headers,
        destination_path: exportConfig.path,
        filename: exportConfig.filename,
        columns: exportConfig.columnsPayload,
        data: exportConfig.dataPayload,
      };

      const res = await fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      const data = await res.json();
      if (data.success) {
        alert('Export successful!\nSaved to: ' + data.saved_to);
        setExportConfig(null);
      } else {
        alert('Export failed: ' + data.error);
      }
    } catch (err) {
      alert('Export failed: ' + err.message);
    } finally {
      setLoading(false);
    }
  };

  const handleFileUpload = (e, field, setFormData) => {
    const file = e.target.files[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = (ev) => setFormData(prev => ({ ...prev, [field]: ev.target.result }));
    reader.readAsText(file);
  };

  return {
    exportConfig, setExportConfig,
    handleExportClick, handleRowContextMenu,
    handleExportRedisResult,
    handleSelectDirectory, handleExecuteExport, handleFileUpload,
  };
}
