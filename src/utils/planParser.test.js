import { describe, it, expect } from 'vitest'
import { parsePostgresNode } from './planParser'

describe('parsePostgresNode', () => {
  it('returns null for null/undefined', () => {
    expect(parsePostgresNode(null)).toBeNull()
    expect(parsePostgresNode(undefined)).toBeNull()
  })

  it('parses basic node', () => {
    const node = {
      'Node Type': 'Seq Scan',
      'Relation Name': 'users',
      'Alias': 'u',
      'Total Cost': 10.5,
      'Plan Rows': 100,
      'Actual Rows': 95,
      'Actual Total Time': 2.345,
      'Actual Loops': 1,
    }
    const result = parsePostgresNode(node)
    expect(result.type).toBe('Seq Scan')
    expect(result.relation).toBe('users')
    expect(result.alias).toBe('u')
    expect(result.cost).toBe('10.50')
    expect(result.rows).toBe(95) // Actual Rows takes precedence
    expect(result.time).toBe('2.345 ms')
    expect(result.loops).toBe(1)
    expect(result.children).toEqual([])
  })

  it('uses defaults for missing fields', () => {
    const node = {}
    const result = parsePostgresNode(node)
    expect(result.type).toBe('Unknown Node')
    expect(result.relation).toBe('')
    expect(result.alias).toBe('')
    expect(result.cost).toBe('0.00')
    expect(result.rows).toBe(0)
    expect(result.time).toBe('')
    expect(result.loops).toBe(1)
  })

  it('parses nested children', () => {
    const node = {
      'Node Type': 'Nested Loop',
      'Total Cost': 20,
      'Plan Rows': 50,
      Plans: [
        {
          'Node Type': 'Seq Scan',
          'Relation Name': 'a',
          'Total Cost': 5,
          'Plan Rows': 10,
        },
        {
          'Node Type': 'Index Scan',
          'Relation Name': 'b',
          'Total Cost': 15,
          'Plan Rows': 5,
        },
      ],
    }
    const result = parsePostgresNode(node)
    expect(result.children).toHaveLength(2)
    expect(result.children[0].type).toBe('Seq Scan')
    expect(result.children[0].relation).toBe('a')
    expect(result.children[1].type).toBe('Index Scan')
    expect(result.children[1].relation).toBe('b')
  })

  it('uses Plan Rows when Actual Rows missing', () => {
    const node = { 'Node Type': 'Seq Scan', 'Plan Rows': 42 }
    const result = parsePostgresNode(node)
    expect(result.rows).toBe(42)
  })
})
