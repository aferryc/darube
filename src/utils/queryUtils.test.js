import { describe, it, expect } from 'vitest'
import { getTargetTable } from './queryUtils'

describe('getTargetTable', () => {
  it('returns null for null/undefined', () => {
    expect(getTargetTable(null)).toBeNull()
    expect(getTargetTable('')).toBeNull()
  })

  it('extracts simple table name from FROM', () => {
    expect(getTargetTable('SELECT * FROM users')).toBe('users')
    expect(getTargetTable('select * from users')).toBe('users')
    expect(getTargetTable('SELECT id, name FROM products')).toBe('products')
  })

  it('extracts schema.table', () => {
    expect(getTargetTable('SELECT * FROM public.users')).toBe('public.users')
    expect(getTargetTable('SELECT * FROM myschema.mytable')).toBe('myschema.mytable')
  })

  it('strips quotes and backticks', () => {
    expect(getTargetTable('SELECT * FROM "users"')).toBe('users')
    expect(getTargetTable('SELECT * FROM `users`')).toBe('users')
    expect(getTargetTable('SELECT * FROM [users]')).toBe('users')
  })

  it('returns null when query has JOIN', () => {
    expect(getTargetTable('SELECT * FROM users JOIN orders ON users.id = orders.user_id')).toBeNull()
    expect(getTargetTable('SELECT * FROM users INNER JOIN orders')).toBeNull()
  })

  it('returns null when query has GROUP BY', () => {
    expect(getTargetTable('SELECT * FROM users GROUP BY id')).toBeNull()
  })

  it('returns null when query has UNION', () => {
    expect(getTargetTable('SELECT * FROM a UNION SELECT * FROM b')).toBeNull()
  })

  it('handles semicolon at end', () => {
    expect(getTargetTable('SELECT * FROM users;')).toBe('users')
  })

  it('handles FROM with trailing space', () => {
    expect(getTargetTable('SELECT * FROM users ')).toBe('users')
  })
})
