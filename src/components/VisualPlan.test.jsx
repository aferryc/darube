import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { VisualPlan } from './VisualPlan'

describe('VisualPlan', () => {
  it('renders "No plan available" when plan is null', () => {
    render(<VisualPlan plan={null} />)
    expect(screen.getByText(/No plan available/)).toBeInTheDocument()
  })

  it('renders "No plan available" when plan is undefined', () => {
    render(<VisualPlan />)
    expect(screen.getByText(/No plan available/)).toBeInTheDocument()
  })

  it('renders Postgres plan with node type and relation', () => {
    const plan = {
      Plan: {
        'Node Type': 'Seq Scan',
        'Relation Name': 'users',
        'Alias': 'u',
        'Total Cost': 10.5,
        'Plan Rows': 100,
      },
    }
    render(<VisualPlan plan={plan} />)
    expect(screen.getByText('Seq Scan')).toBeInTheDocument()
    expect(screen.getByText(/users/)).toBeInTheDocument()
    expect(screen.getByText(/Total Cost/)).toBeInTheDocument()
  })

  it('renders nested plan nodes', () => {
    const plan = {
      Plan: {
        'Node Type': 'Nested Loop',
        'Total Cost': 20,
        'Plan Rows': 50,
        Plans: [
          {
            'Node Type': 'Seq Scan',
            'Relation Name': 'orders',
            'Total Cost': 5,
            'Plan Rows': 10,
          },
        ],
      },
    }
    render(<VisualPlan plan={plan} />)
    expect(screen.getByText('Nested Loop')).toBeInTheDocument()
    expect(screen.getByText('Seq Scan')).toBeInTheDocument()
    expect(screen.getByText(/orders/)).toBeInTheDocument()
  })

  it('renders raw JSON fallback for unsupported plan format', () => {
    const plan = { some: 'unknown', format: true }
    render(<VisualPlan plan={plan} />)
    expect(screen.getByText('Raw Execution Plan')).toBeInTheDocument()
    expect(screen.getByText(/"some"/)).toBeInTheDocument()
  })
})
