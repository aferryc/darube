import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { SqlAutocomplete } from './SqlAutocomplete'

describe('SqlAutocomplete', () => {
  const defaultProps = {
    value: '',
    onChange: vi.fn(),
    onKeyDown: vi.fn(),
    disabled: false,
    placeholder: 'Type SQL...',
    apiUrl: 'http://localhost:3000',
    connectionId: null,
  }

  beforeEach(() => {
    vi.clearAllMocks()
    global.fetch = vi.fn()
  })

  it('renders with placeholder', () => {
    render(<SqlAutocomplete {...defaultProps} />)
    expect(screen.getByPlaceholderText('Type SQL...')).toBeInTheDocument()
  })

  it('renders with initial value', () => {
    render(<SqlAutocomplete {...defaultProps} value="SELECT * FROM " />)
    const editor = screen.getByRole('textbox')
    expect(editor).toHaveValue('SELECT * FROM ')
  })

  it('calls onChange when user types', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    render(<SqlAutocomplete {...defaultProps} onChange={onChange} />)
    const editor = screen.getByRole('textbox')
    await user.type(editor, 'x')
    expect(onChange).toHaveBeenCalled()
  })

  it('shows disabled state when disabled prop is true', () => {
    render(<SqlAutocomplete {...defaultProps} disabled />)
    const editor = screen.getByRole('textbox')
    expect(editor).toBeDisabled()
  })
})
