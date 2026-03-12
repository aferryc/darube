import { fireEvent, render, screen } from '@testing-library/react';
import React from 'react';
import { describe, expect, it } from 'vitest';
import { QueryTabs } from './QueryTabs';

function Wrapper({ initialTabs, initialActive = 't2' }) {
  const [tabs, setTabs] = React.useState(initialTabs);
  const [activeTabId, setActiveTabId] = React.useState(initialActive);
  const [editingTabId, setEditingTabId] = React.useState(null);

  const closeTab = (e, idToClose) => {
    e.stopPropagation();
    if (tabs.length === 1) return;
    const next = tabs.filter((t) => t.id !== idToClose);
    setTabs(next);
    if (activeTabId === idToClose) setActiveTabId(next[0].id);
  };

  return (
    <QueryTabs
      tabs={tabs}
      activeTabId={activeTabId}
      setActiveTabId={setActiveTabId}
      editingTabId={editingTabId}
      setEditingTabId={setEditingTabId}
      setTabs={setTabs}
      addNewTab={() => {}}
      closeTab={closeTab}
      activeId="conn-1"
      loading={false}
      executeQuery={() => {}}
      executeExplain={() => {}}
      activeConnType="sql"
    />
  );
}

const makeTabs = () => ([
  { id: 't1', name: 'Query 1', type: 'query', query: 'select 1', connectionId: 'c', lastExecutedQuery: '', results: null, plan: null, activeView: 'results', currentPage: 1, rowsPerPage: 50, selectedRows: [], lastSelectedIndex: null, history: [], historyIndex: -1, targetTable: null },
  { id: 't2', name: 'Query 2', type: 'query', query: 'select 2', connectionId: 'c', lastExecutedQuery: '', results: null, plan: null, activeView: 'results', currentPage: 1, rowsPerPage: 50, selectedRows: [], lastSelectedIndex: null, history: [], historyIndex: -1, targetTable: null },
  { id: 't3', name: 'Script 1', type: 'script', query: 'console.log(1)', connectionId: 'c', lastExecutedQuery: '', results: null, plan: null, activeView: 'results', currentPage: 1, rowsPerPage: 50, selectedRows: [], lastSelectedIndex: null, history: [], historyIndex: -1, targetTable: null },
]);

function openMenu(tabName) {
  const tab = screen.getByText(tabName);
  fireEvent.contextMenu(tab);
}

describe('QueryTabs context menu', () => {
  it('closes tabs to the right', () => {
    render(<Wrapper initialTabs={makeTabs()} initialActive="t2" />);
    openMenu('Query 2');
    fireEvent.click(screen.getByText('Close Tab to the right'));
    expect(screen.getByText('Query 1')).toBeInTheDocument();
    expect(screen.getByText('Query 2')).toBeInTheDocument();
    expect(screen.queryByText('Script 1')).toBeNull();
  });

  it('closes tabs to the left', () => {
    render(<Wrapper initialTabs={makeTabs()} initialActive="t2" />);
    openMenu('Query 2');
    fireEvent.click(screen.getByText('Close Tab to the left'));
    expect(screen.queryByText('Query 1')).toBeNull();
    expect(screen.getByText('Query 2')).toBeInTheDocument();
    expect(screen.getByText('Script 1')).toBeInTheDocument();
  });

  it('closes all except this tab', () => {
    render(<Wrapper initialTabs={makeTabs()} initialActive="t2" />);
    openMenu('Query 2');
    fireEvent.click(screen.getByText('Close all except this tab'));
    expect(screen.queryByText('Query 1')).toBeNull();
    expect(screen.getByText('Query 2')).toBeInTheDocument();
    expect(screen.queryByText('Script 1')).toBeNull();
  });

  it('duplicates tab and activates copy', () => {
    render(<Wrapper initialTabs={makeTabs()} initialActive="t2" />);
    openMenu('Query 2');
    fireEvent.click(screen.getByText('Duplicate Tab'));
    expect(screen.getByText('Query 2 copy')).toBeInTheDocument();
  });
});

