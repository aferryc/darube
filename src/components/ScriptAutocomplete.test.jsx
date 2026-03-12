import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import React from 'react'

import { registerScriptAutocomplete, ScriptAutocomplete } from './ScriptAutocomplete'

function makeMonacoStub() {
  const registrations = []

  function Range(startLineNumber, startColumn, endLineNumber, endColumn) {
    this.startLineNumber = startLineNumber
    this.startColumn = startColumn
    this.endLineNumber = endLineNumber
    this.endColumn = endColumn
  }

  const monaco = {
    Range,
    languages: {
      registerCompletionItemProvider: vi.fn((langId, provider) => {
        registrations.push({ langId, provider })
        return { dispose: vi.fn() }
      }),
      CompletionItemKind: {
        Function: 1,
        Method: 2,
        Keyword: 3,
      },
      CompletionItemInsertTextRule: {
        InsertAsSnippet: 4,
      },
    },
  }

  return { monaco, registrations }
}

function makeModel(text) {
  return {
    getValue: () => text,
    getOffsetAt: (pos) => pos.__offset,
    getPositionAt: (offset) => ({ lineNumber: 1, column: offset + 1 }),
  }
}

function provide(provider, text, cursor) {
  const model = makeModel(text)
  const position = { lineNumber: 1, column: cursor + 1, __offset: cursor }
  return provider.provideCompletionItems(model, position)
}

describe('ScriptAutocomplete (Monaco completions)', () => {
  it('registers a JS completion provider and updates editor options', () => {
    const { monaco, registrations } = makeMonacoStub()
    const editor = { updateOptions: vi.fn() }

    registerScriptAutocomplete(monaco, editor, () => [])

    expect(monaco.languages.registerCompletionItemProvider).toHaveBeenCalledWith('javascript', expect.any(Object))
    expect(registrations[0]?.langId).toBe('javascript')
    expect(editor.updateOptions).toHaveBeenCalled()
  })

  it('suggests connection names inside db.conn(\"...\") and inserts without extra quotes', () => {
    const { monaco, registrations } = makeMonacoStub()
    const editor = { updateOptions: vi.fn() }
    registerScriptAutocomplete(monaco, editor, () => [
      { id: 'c1', connection_name: 'prod-postgres', db_type: 'postgres' },
      { id: 'c2', connection_name: 'cache', db_type: 'redis' },
    ])

    const provider = registrations[0].provider
    const text = 'db.conn(\"pro'
    const res = provide(provider, text, text.length)

    const item = res.suggestions.find((s) => s.label === 'prod-postgres (postgres)')
    expect(item).toBeTruthy()
    expect(item.insertText).toBe('prod-postgres')
  })

  it('inserts quoted connection name when db.conn( has no opening quote', () => {
    const { monaco, registrations } = makeMonacoStub()
    const editor = { updateOptions: vi.fn() }
    registerScriptAutocomplete(monaco, editor, () => [
      { id: 'c1', connection_name: 'prod-postgres', db_type: 'postgres' },
    ])

    const provider = registrations[0].provider
    const text = 'const pg = db.conn(pro'
    const res = provide(provider, text, text.length)

    const item = res.suggestions.find((s) => s.label === 'prod-postgres (postgres)')
    expect(item).toBeTruthy()
    expect(item.insertText).toBe('\"prod-postgres\"')
  })

  it('suggests utils.* when typing utils.n', () => {
    const { monaco, registrations } = makeMonacoStub()
    const editor = { updateOptions: vi.fn() }
    registerScriptAutocomplete(monaco, editor, () => [])

    const provider = registrations[0].provider
    const text = 'utils.n'
    const res = provide(provider, text, text.length)

    expect(res.suggestions.some((s) => s.label === 'utils.now')).toBe(true)
    expect(res.suggestions.some((s) => s.label === 'utils.nowUnixMs')).toBe(true)
  })

  it('suggests db.conn after typing db. and inserts as a snippet', () => {
    const { monaco, registrations } = makeMonacoStub()
    const editor = { updateOptions: vi.fn() }
    registerScriptAutocomplete(monaco, editor, () => [])

    const provider = registrations[0].provider
    const text = 'db.c'
    const res = provide(provider, text, text.length)
    const item = res.suggestions.find((s) => s.label === 'db.conn')

    expect(item).toBeTruthy()
    expect(item.insertText).toBe('db.conn(\"$0\")')
    expect(item.insertTextRules).toBe(monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet)
  })

  it('suggests methods after var dot and inserts with prefix as a snippet', () => {
    const { monaco, registrations } = makeMonacoStub()
    const editor = { updateOptions: vi.fn() }
    registerScriptAutocomplete(monaco, editor, () => [])

    const provider = registrations[0].provider
    const text = 'pg.q'
    const res = provide(provider, text, text.length)
    const item = res.suggestions.find((s) => s.label === '.query')

    expect(item).toBeTruthy()
    expect(item.insertText).toBe('pg.query(\"$0\")')
    expect(item.insertTextRules).toBe(monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet)
  })

  it('renders in test env using a textarea fallback', () => {
    render(
      <ScriptAutocomplete
        value={'const x = 1'}
        onChange={() => {}}
        placeholder="Script..."
        connections={[]}
      />
    )
    const ta = screen.getByRole('textbox')
    expect(ta).toHaveValue('const x = 1')
  })
})

