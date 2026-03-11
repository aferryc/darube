import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import React from 'react'
import { ScriptAutocomplete } from './ScriptAutocomplete'

vi.mock('../utils/textareaCaret', () => ({
  getTextareaCaretViewportPosition: () => ({ top: 10, left: 10, height: 18 }),
}))

function Stateful({ initialValue, connections }) {
  const [value, setValue] = React.useState(initialValue)
  return (
    <div>
      <div data-testid="val">{value}</div>
      <ScriptAutocomplete
        value={value}
        onChange={setValue}
        disabled={false}
        placeholder="Script..."
        connections={connections || []}
      />
    </div>
  )
}

describe('ScriptAutocomplete', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('suggests db.conn connection names inside db.conn(', async () => {
    const user = userEvent.setup()
    render(
      <Stateful
        initialValue={'db.conn("'}
        connections={[
          { id: 'c1', connection_name: 'prod-postgres', db_type: 'postgres' },
          { id: 'c2', connection_name: 'cache', db_type: 'redis' },
        ]}
      />
    )

    const ta = screen.getByRole('textbox')
    await user.type(ta, 'pro')
    fireEvent.keyUp(ta, { key: 'o' })

    // Dropdown should include prod-postgres suggestion.
    expect(screen.getByText('prod-postgres (postgres)')).toBeInTheDocument()
    await user.click(screen.getByText('prod-postgres (postgres)'))

    // Inserts connection name (keeps quotes as user already typed a quote).
    expect(screen.getByTestId('val').textContent).toContain('prod-postgres')
  })

  it('suggests utils.* when typing utils.', async () => {
    const user = userEvent.setup()
    render(
      <Stateful initialValue={'utils.'} connections={[]} />
    )

    const ta = screen.getByRole('textbox')
    // With "utils.n" fragment, we should see now/nowUnixMs.
    await user.type(ta, 'n')
    fireEvent.keyUp(ta, { key: 'n' })
    expect(screen.getByText('utils.now')).toBeInTheDocument()
    expect(screen.getByText('utils.nowUnixMs')).toBeInTheDocument()
  })

  it('suggests db.conn after typing db.', async () => {
    const user = userEvent.setup()
    render(
      <Stateful initialValue={'db.'} connections={[]} />
    )

    const ta = screen.getByRole('textbox')
    await user.type(ta, 'c')
    expect(screen.getByText('db.conn')).toBeInTheDocument()
  })

  it('inserts quoted connection name when db.conn( has no opening quote', async () => {
    const user = userEvent.setup()
    render(
      <Stateful
        initialValue={'const pg = db.conn('}
        connections={[
          { id: 'c1', connection_name: 'prod-postgres', db_type: 'postgres' },
        ]}
      />
    )

    const ta = screen.getByRole('textbox')
    await user.type(ta, 'pro')
    fireEvent.keyUp(ta, { key: 'o' })
    await user.click(screen.getByText('prod-postgres (postgres)'))

    expect(screen.getByTestId('val').textContent).toContain('"prod-postgres"')
  })

  it('suggests methods after var dot and inserts with prefix', async () => {
    const user = userEvent.setup()
    render(
      <Stateful initialValue={'pg.q'} connections={[]} />
    )

    const ta = screen.getByRole('textbox')
    await user.type(ta, 'u')
    fireEvent.keyUp(ta, { key: 'u' })
    expect(screen.getByText('.query')).toBeInTheDocument()
    await user.click(screen.getByText('.query'))

    expect(screen.getByTestId('val').textContent).toContain('pg.query("')
  })

  it('closes dropdown on outside click', async () => {
    const user = userEvent.setup()
    render(<Stateful initialValue={'db.'} connections={[]} />)
    const ta = screen.getByRole('textbox')
    await user.type(ta, 'c')
    fireEvent.keyUp(ta, { key: 'c' })
    expect(screen.getByText('db.conn')).toBeInTheDocument()

    fireEvent.mouseDown(document.body)
    expect(screen.queryByText('db.conn')).not.toBeInTheDocument()
  })

  it('supports Tab to accept selected suggestion and Escape to close', async () => {
    const user = userEvent.setup()
    render(<Stateful initialValue={'db.'} connections={[]} />)
    const ta = screen.getByRole('textbox')
    await user.type(ta, 'c')
    fireEvent.keyUp(ta, { key: 'c' })
    expect(screen.getByText('db.conn')).toBeInTheDocument()

    fireEvent.keyDown(ta, { key: 'Tab' })
    expect(screen.getByTestId('val').textContent).toContain('db.conn("')

    // Open again and close with Escape.
    await user.type(ta, ' ')
    fireEvent.keyUp(ta, { key: ' ' })
    expect(screen.getByText('db.conn')).toBeInTheDocument()
    fireEvent.keyDown(ta, { key: 'Escape' })
    expect(screen.queryByText('db.conn')).not.toBeInTheDocument()
  })

  it('does not treat mousedown on dropdown as outside click', async () => {
    const user = userEvent.setup()
    render(<Stateful initialValue={'db.'} connections={[]} />)
    const ta = screen.getByRole('textbox')
    await user.type(ta, 'c')
    fireEvent.keyUp(ta, { key: 'c' })
    const item = screen.getByText('db.conn')
    expect(item).toBeInTheDocument()

    // This would close before our bugfix; now it should remain open on mousedown.
    fireEvent.mouseDown(item)
    expect(screen.getByText('db.conn')).toBeInTheDocument()
  })

  it('suggests JS keywords when not using dot context', async () => {
    const user = userEvent.setup()
    render(<Stateful initialValue={''} connections={[]} />)
    const ta = screen.getByRole('textbox')
    await user.type(ta, 'co')
    fireEvent.keyUp(ta, { key: 'o' })
    expect(screen.getByText('const')).toBeInTheDocument()
  })
})
