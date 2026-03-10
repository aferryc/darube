import { describe, it, expect } from 'vitest'
import { computeWorkingData, consolidateMutations } from './mutationUtils'

describe('computeWorkingData', () => {
  const columns = ['id', 'name']

  it('returns empty for null rows', () => {
    expect(computeWorkingData(null, columns, [], 0)).toEqual([])
  })

  it('returns empty for null columns', () => {
    expect(computeWorkingData([], null, [], 0)).toEqual([])
  })

  it('returns rows unchanged when no history', () => {
    const rows = [
      [1, 'Alice'],
      [2, 'Bob'],
    ]
    rows[0]._ui_id = 'id1'
    rows[1]._ui_id = 'id2'
    const result = computeWorkingData(rows, columns, [], -1)
    expect(result).toHaveLength(2)
    expect(result[0][0]).toBe(1)
    expect(result[0][1]).toBe('Alice')
  })

  it('applies update mutation', () => {
    const rows = [[1, 'Alice']]
    rows[0]._ui_id = 'id1'
    const history = [
      {
        type: 'update',
        uiId: 'id1',
        colName: 'name',
        newValue: 'Alicia',
      },
    ]
    const result = computeWorkingData(rows, columns, history, 0)
    expect(result).toHaveLength(1)
    expect(result[0][1]).toBe('Alicia')
    expect(result[0]._mutatedCols?.name).toBe(true)
  })

  it('applies delete mutation', () => {
    const rows = [[1, 'Alice'], [2, 'Bob']]
    rows[0]._ui_id = 'id1'
    rows[1]._ui_id = 'id2'
    const history = [
      {
        type: 'delete',
        uiId: 'id1',
      },
    ]
    const result = computeWorkingData(rows, columns, history, 0)
    expect(result).toHaveLength(1)
    expect(result[0][1]).toBe('Bob')
  })

  it('applies insert mutation', () => {
    const rows = [[1, 'Alice']]
    rows[0]._ui_id = 'id1'
    const history = [
      {
        type: 'insert',
        uiId: 'new1',
        newValues: { id: 2, name: 'Bob' },
      },
    ]
    const result = computeWorkingData(rows, columns, history, 0)
    expect(result).toHaveLength(2)
    expect(result[1][0]).toBe(2)
    expect(result[1][1]).toBe('Bob')
    expect(result[1]._isInserted).toBe(true)
  })

  it('respects historyIndex', () => {
    const rows = [[1, 'Alice']]
    rows[0]._ui_id = 'id1'
    const history = [
      { type: 'update', uiId: 'id1', colName: 'name', newValue: 'Alicia' },
      { type: 'update', uiId: 'id1', colName: 'name', newValue: 'Alyssa' },
    ]
    const result = computeWorkingData(rows, columns, history, 0)
    expect(result[0][1]).toBe('Alicia')
    const resultFull = computeWorkingData(rows, columns, history, 1)
    expect(resultFull[0][1]).toBe('Alyssa')
  })
})

describe('consolidateMutations', () => {
  it('consolidates multiple updates to same row', () => {
    const history = [
      {
        type: 'update',
        uiId: 'id1',
        originalRow: { id: 1, name: 'Alice' },
        newValues: { name: 'Alicia' },
      },
      {
        type: 'update',
        uiId: 'id1',
        originalRow: { id: 1, name: 'Alice' },
        newValues: { name: 'Alyssa' },
      },
    ]
    const result = consolidateMutations(history)
    expect(result).toHaveLength(1)
    expect(result[0].type).toBe('update')
    expect(result[0].newValues.name).toBe('Alyssa')
  })

  it('delete overrides previous operations', () => {
    const history = [
      {
        type: 'insert',
        uiId: 'new1',
        originalRow: undefined,
        newValues: { id: 1, name: 'New' },
      },
      {
        type: 'delete',
        uiId: 'new1',
        originalRow: undefined,
      },
    ]
    const result = consolidateMutations(history)
    expect(result).toHaveLength(0) // insert with no originalRow deleted
  })

  it('insert + update merges newValues', () => {
    const history = [
      {
        type: 'insert',
        uiId: 'new1',
        newValues: { id: 1, name: 'Alice' },
      },
      {
        type: 'update',
        uiId: 'new1',
        newValues: { name: 'Alicia' },
      },
    ]
    const result = consolidateMutations(history)
    expect(result).toHaveLength(1)
    expect(result[0].newValues).toEqual({ id: 1, name: 'Alicia' })
  })

  it('keeps delete when originalRow exists', () => {
    const history = [
      {
        type: 'update',
        uiId: 'id1',
        originalRow: { id: 1, name: 'Alice' },
        newValues: { name: 'Alicia' },
      },
      {
        type: 'delete',
        uiId: 'id1',
        originalRow: { id: 1, name: 'Alice' },
      },
    ]
    const result = consolidateMutations(history)
    expect(result).toHaveLength(1)
    expect(result[0].type).toBe('delete')
  })
})
