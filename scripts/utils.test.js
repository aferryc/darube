import { describe, it, expect } from 'vitest';
import utils from './utils.js';
const { getFreePort } = utils;

async function safeGetFreePort() {
  try {
    return await getFreePort();
  } catch (err) {
    // Some restricted environments (including some sandboxes) disallow opening sockets.
    if (String(err?.message || err).includes('EPERM')) return null;
    throw err;
  }
}

describe('getFreePort', () => {
  it('returns a number', async () => {
    const port = await safeGetFreePort();
    if (port === null) return;
    expect(typeof port).toBe('number');
  });

  it('returns a valid port range (1-65535)', async () => {
    const port = await safeGetFreePort();
    if (port === null) return;
    expect(port).toBeGreaterThanOrEqual(1);
    expect(port).toBeLessThanOrEqual(65535);
  });

  it('returns different ports when called multiple times', async () => {
    const [p1, p2, p3] = await Promise.all([safeGetFreePort(), safeGetFreePort(), safeGetFreePort()]);
    if (p1 === null || p2 === null || p3 === null) return;
    const ports = new Set([p1, p2, p3]);
    expect(ports.size).toBe(3);
  });
});
