import React from 'react';
import { act, cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import App from './App';

vi.mock('./components/MonacoCodeEditor', () => ({
  MonacoCodeEditor: () => <div />,
}));
vi.mock('react-split', () => ({ default: ({ children }) => <div>{children}</div> }));

describe('manual connection lifecycle', () => {
  let connected;
  let connect;
  let fetchMock;

  beforeEach(() => {
    vi.useFakeTimers();
    connected = false;
    connect = vi.fn(async () => ({ success: false, error: 'Database unavailable' }));
    fetchMock = vi.fn(async (url, options) => {
      let data;
      if (url.endsWith('/api/connections')) {
        data = { connections: [{
          id: 'saved', connection_name: 'Saved database', db_type: 'postgres',
          status: connected ? 'connected' : 'disconnected',
        }] };
      } else if (url.endsWith('/api/connections/connect')) {
        data = await connect();
        if (data.success) connected = true;
      } else if (url.endsWith('/api/workspace') && options?.method !== 'POST') {
        data = { tabs: [{ id: 'restored', name: 'Restored query', query: 'SELECT 1', connection_id: 'saved' }] };
      } else if (url.endsWith('/api/folders')) {
        data = { success: true, folders: [] };
      } else {
        data = { success: true, databases: [], schemas: [], status: 'idle' };
      }
      return { ok: true, json: async () => data };
    });
    vi.stubGlobal('fetch', fetchMock);
    vi.stubGlobal('alert', vi.fn());
  });

  afterEach(() => {
    cleanup();
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  const start = async () => {
    await act(async () => { render(<App />); });
    return screen.getByText('Saved database').closest('.connection-item');
  };

  it('restores tabs and polls saved connections without opening them', async () => {
    const row = await start();
    expect(screen.getByText('Restored query')).toBeInTheDocument();
    expect(row.querySelector('.status-dot')).toHaveClass('disconnected');
    await act(async () => { vi.advanceTimersByTime(10000); });
    fireEvent.click(row);
    expect(connect).not.toHaveBeenCalled();
    expect(fetchMock.mock.calls.some(([url]) => url.includes('/metadata/'))).toBe(false);
  });

  it('attempts once on double-click and waits for another user action after failure', async () => {
    const row = await start();
    await act(async () => { fireEvent.doubleClick(row); });
    expect(connect).toHaveBeenCalledTimes(1);
    expect(alert).toHaveBeenCalledWith('Failed to connect: Database unavailable');
    await act(async () => { vi.advanceTimersByTime(10000); });
    expect(connect).toHaveBeenCalledTimes(1);
    expect(row.querySelector('.status-dot')).toHaveClass('disconnected');

    connect.mockResolvedValue({ success: true });
    await act(async () => { fireEvent.click(screen.getByRole('button', { name: 'Connect' })); });
    expect(connect).toHaveBeenCalledTimes(2);
    expect(row.querySelector('.status-dot')).toHaveClass('connected');
    await act(async () => { fireEvent.doubleClick(row); });
    expect(connect).toHaveBeenCalledTimes(2);
  });

  it('does not retry a failed Teleport connection', async () => {
    connect.mockResolvedValue({ success: false, error: 'Teleport session expired. Run tsh login' });
    const row = await start();
    await act(async () => { fireEvent.doubleClick(row); });
    await act(async () => { vi.advanceTimersByTime(10000); });
    expect(connect).toHaveBeenCalledTimes(1);
    expect(fetchMock.mock.calls.some(([url]) => url.includes('/api/teleport/'))).toBe(false);
  });

  it('ignores duplicate triggers while a connection attempt is pending', async () => {
    let finish;
    connect.mockImplementation(() => new Promise(resolve => { finish = resolve; }));
    const row = await start();
    await act(async () => { fireEvent.doubleClick(row); });
    await act(async () => { fireEvent.doubleClick(row); });
    expect(connect).toHaveBeenCalledTimes(1);
    await act(async () => { finish({ success: false, error: 'Database unavailable' }); });
    await act(async () => { fireEvent.doubleClick(row); });
    expect(connect).toHaveBeenCalledTimes(2);
    await act(async () => { finish({ success: false, error: 'Database unavailable' }); });
  });

  it('supports double-click to connect with the sidebar collapsed', async () => {
    await start();
    fireEvent.click(screen.getByTitle('Collapse sidebar'));
    const icon = screen.getByTitle('Saved database (postgres) — disconnected');
    fireEvent.click(icon);
    expect(connect).not.toHaveBeenCalled();
    await act(async () => { fireEvent.doubleClick(icon); });
    expect(connect).toHaveBeenCalledTimes(1);
  });
});
