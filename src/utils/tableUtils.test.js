import { describe, it, expect } from 'vitest'
import { groupTables } from './tableUtils'

describe('groupTables', () => {
  it('returns empty array for null/undefined', () => {
    expect(groupTables(null)).toEqual([])
    expect(groupTables(undefined)).toEqual([])
  })

  it('returns empty array for empty tables', () => {
    expect(groupTables([])).toEqual([])
  })

  it('returns single table as non-group when no suffix match', () => {
    const tables = [{ name: 'users', type: 'table' }]
    expect(groupTables(tables)).toEqual([{ name: 'users', type: 'table', isGroup: false }])
  })

  it('groups tables with _YYYYMMDD suffix', () => {
    const tables = [
      { name: 'logs_20191225', type: 'table' },
      { name: 'logs_20200101', type: 'table' },
      { name: 'logs_20240315', type: 'table' },
    ]
    const result = groupTables(tables)
    expect(result).toHaveLength(1)
    expect(result[0].isGroup).toBe(true)
    expect(result[0].name).toBe('logs_*')
    expect(result[0].tables).toHaveLength(3)
  })

  it('keeps single suffix table as non-group', () => {
    const tables = [{ name: 'logs_20191225', type: 'table' }]
    const result = groupTables(tables)
    expect(result).toHaveLength(1)
    expect(result[0].isGroup).toBe(false)
    expect(result[0].name).toBe('logs_20191225')
  })

  it('handles mixed grouped and ungrouped tables', () => {
    const tables = [
      { name: 'users', type: 'table' },
      { name: 'events_20240101', type: 'table' },
      { name: 'events_20240201', type: 'table' },
      { name: 'orders', type: 'table' },
    ]
    const result = groupTables(tables)
    // users -> single, events_* -> group of 2, orders -> single = 3 items
    expect(result).toHaveLength(3)
    const names = result.map((r) => r.name).sort()
    expect(names).toContain('users')
    expect(names).toContain('orders')
    expect(names).toContain('events_*')
  })

  it('sorts result by name', () => {
    const tables = [
      { name: 'zebra', type: 'table' },
      { name: 'alpha', type: 'table' },
      { name: 'beta', type: 'table' },
    ]
    const result = groupTables(tables)
    expect(result[0].name).toBe('alpha')
    expect(result[1].name).toBe('beta')
    expect(result[2].name).toBe('zebra')
  })

  it('matches _YYYY pattern (4+ digits)', () => {
    const tables = [
      { name: 'data_2024', type: 'table' },
      { name: 'data_2025', type: 'table' },
    ]
    const result = groupTables(tables)
    expect(result[0].isGroup).toBe(true)
    expect(result[0].name).toBe('data_*')
  })
})
