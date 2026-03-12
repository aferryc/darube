import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { ContextMenu } from './Modals';

describe('ContextMenu (text + results)', () => {
  it('renders text actions and disables cut/copy without selection', () => {
    const onAction = vi.fn();
    const el = document.createElement('textarea');
    el.value = 'select 1;';
    el.dataset.darubeEditorRole = 'sql';
    render(
      <ContextMenu
        contextMenu={{ visible: true, x: 10, y: 10, type: 'text', data: { el, hasSelection: false, hasText: true, readOnly: false, editorRole: 'sql' } }}
        onAction={onAction}
      />
    );

    expect(screen.getByText('Run Query')).not.toHaveClass('disabled');
    expect(screen.getByText('Cut')).toHaveClass('disabled');
    expect(screen.getByText('Copy')).toHaveClass('disabled');
    expect(screen.getByText('Paste')).not.toHaveClass('disabled');
    expect(screen.getByText('Select All')).toBeInTheDocument();

    fireEvent.click(screen.getByText('Copy'));
    expect(onAction).not.toHaveBeenCalled();
  });

  it('fires actions when enabled', () => {
    const onAction = vi.fn();
    render(
      <ContextMenu
        contextMenu={{ visible: true, x: 10, y: 10, type: 'text', data: { hasSelection: true, readOnly: false } }}
        onAction={onAction}
      />
    );

    fireEvent.click(screen.getByText('Copy'));
    expect(onAction).toHaveBeenCalledWith('copy');
  });

  it('renders results actions and disables selected-row items when none selected', () => {
    const onAction = vi.fn();
    render(
      <ContextMenu
        contextMenu={{
          visible: true,
          x: 10,
          y: 10,
          type: 'results',
          data: { rowIndex: 0, colIndex: null, selectedRows: [] },
        }}
        onAction={onAction}
      />
    );

    expect(screen.getByText('Copy Cell')).toHaveClass('disabled');
    expect(screen.getByText('Copy Row (TSV)')).not.toHaveClass('disabled');
    expect(screen.getByText('Copy Selected Rows (TSV)')).toHaveClass('disabled');
    expect(screen.getByText('Export Selected Rows...')).toHaveClass('disabled');
  });
});
