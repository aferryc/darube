import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import React from 'react'
import { ScriptHelpModal } from './ScriptHelpModal'

describe('ScriptHelpModal', () => {
  beforeEach(() => {
    global.navigator.clipboard = { writeText: vi.fn().mockResolvedValue(undefined) }
  })

  it('switches tabs and expands/copies example', async () => {
    render(<ScriptHelpModal show onClose={() => {}} />)

    // Default is Overview
    expect(screen.getByText('Basics')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Syntax' }))
    expect(screen.getByText('Connection Methods')).toBeInTheDocument()

    // Example collapsed by default
    expect(screen.getByRole('button', { name: 'Expand' })).toBeInTheDocument()
    const copyBtn = screen.getByRole('button', { name: 'Copy' })
    expect(copyBtn).toBeDisabled()

    fireEvent.click(screen.getByRole('button', { name: 'Expand' }))
    expect(copyBtn).not.toBeDisabled()
    expect(screen.getByText(/console\.log\("hello world"\)/)).toBeInTheDocument()

    fireEvent.click(copyBtn)
    expect(global.navigator.clipboard.writeText).toHaveBeenCalled()
    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Copied' })).toBeInTheDocument()
    })
  })

  it('does not crash when clipboard copy fails', async () => {
    global.navigator.clipboard.writeText = vi.fn().mockRejectedValue(new Error('nope'))
    render(<ScriptHelpModal show onClose={() => {}} />)

    fireEvent.click(screen.getByRole('button', { name: 'Syntax' }))
    fireEvent.click(screen.getByRole('button', { name: 'Expand' }))
    fireEvent.click(screen.getByRole('button', { name: 'Copy' }))

    await waitFor(() => {
      expect(global.navigator.clipboard.writeText).toHaveBeenCalled()
    })
    // Still rendered and interactive
    expect(screen.getByRole('button', { name: 'Collapse' })).toBeInTheDocument()
  })
})
