import { describe, it, expect } from 'vitest'
import { parseContext } from './sqlContext'

describe('parseContext', () => {
  it('returns empty currentWord for empty query', () => {
    const result = parseContext('', 0)
    expect(result.currentWord).toBe('')
    expect(result.rawWord).toBe('')
    expect(result.clause).toBe('SELECT')
  })

  it('detects clause at cursor', () => {
    const q = 'SELECT id, name FROM users WHERE '
    const result = parseContext(q, q.length)
    expect(result.clause).toBe('WHERE')
  })

  it('detects FROM clause', () => {
    const q = 'SELECT * FROM '
    const result = parseContext(q, q.length)
    expect(result.clause).toBe('FROM')
    expect(result.currentWord).toBe('')
  })

  it('extracts current word at cursor', () => {
    const q = 'SELECT us'
    const result = parseContext(q, q.length)
    expect(result.currentWord).toBe('us')
    expect(result.rawWord).toBe('us')
  })

  it('parses dot prefix for alias.column', () => {
    const q = 'SELECT u.'
    const result = parseContext(q, q.length)
    expect(result.dotPrefix).toBe('u')
    expect(result.currentWord).toBe('')
  })

  it('parses dot prefix with partial column', () => {
    const q = 'SELECT u.na'
    const result = parseContext(q, q.length)
    expect(result.dotPrefix).toBe('u')
    expect(result.currentWord).toBe('na')
  })

  it('extracts tables from FROM and JOIN', () => {
    const q = 'SELECT * FROM users u JOIN orders o ON u.id = o.user_id'
    const result = parseContext(q, q.length)
    expect(result.tables).toHaveLength(2)
    expect(result.tables.find((t) => t.name === 'users')).toBeDefined()
    expect(result.tables.find((t) => t.name === 'orders')).toBeDefined()
    expect(result.aliasMap.u).toBe('users')
    expect(result.aliasMap.o).toBe('orders')
  })

  it('handles GROUP BY clause', () => {
    const q = 'SELECT id FROM users GROUP BY '
    const result = parseContext(q, q.length)
    expect(result.clause).toBe('GROUP_BY')
  })
})
