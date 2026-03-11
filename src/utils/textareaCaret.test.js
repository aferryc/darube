import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { getTextareaCaretViewportPosition } from './textareaCaret'

describe('getTextareaCaretViewportPosition', () => {
  const origGetComputedStyle = window.getComputedStyle
  const origGetBounding = HTMLElement.prototype.getBoundingClientRect
  const origOffsetTop = Object.getOwnPropertyDescriptor(HTMLElement.prototype, 'offsetTop')
  const origOffsetLeft = Object.getOwnPropertyDescriptor(HTMLElement.prototype, 'offsetLeft')

  beforeEach(() => {
    window.getComputedStyle = vi.fn(() => ({
      boxSizing: 'border-box',
      width: '200px',
      paddingTop: '0px',
      paddingRight: '0px',
      paddingBottom: '0px',
      paddingLeft: '0px',
      borderTopWidth: '0px',
      borderRightWidth: '0px',
      borderBottomWidth: '0px',
      borderLeftWidth: '0px',
      fontFamily: 'monospace',
      fontSize: '12px',
      fontWeight: '400',
      fontStyle: 'normal',
      letterSpacing: '0px',
      textTransform: 'none',
      textIndent: '0px',
      lineHeight: '20px',
      tabSize: '2',
    }))

    HTMLElement.prototype.getBoundingClientRect = vi.fn(() => ({
      top: 100,
      left: 50,
      width: 300,
      height: 40,
      right: 0,
      bottom: 0,
      x: 0,
      y: 0,
      toJSON: () => {},
    }))

    Object.defineProperty(HTMLElement.prototype, 'offsetTop', {
      configurable: true,
      get() { return 12 },
    })
    Object.defineProperty(HTMLElement.prototype, 'offsetLeft', {
      configurable: true,
      get() { return 34 },
    })
  })

  afterEach(() => {
    window.getComputedStyle = origGetComputedStyle
    HTMLElement.prototype.getBoundingClientRect = origGetBounding
    if (origOffsetTop) Object.defineProperty(HTMLElement.prototype, 'offsetTop', origOffsetTop)
    else delete HTMLElement.prototype.offsetTop
    if (origOffsetLeft) Object.defineProperty(HTMLElement.prototype, 'offsetLeft', origOffsetLeft)
    else delete HTMLElement.prototype.offsetLeft
  })

  it('returns null when textarea is missing', () => {
    expect(getTextareaCaretViewportPosition(null, 0)).toBeNull()
  })

  it('uses caretPos when provided', () => {
    const ta = document.createElement('textarea')
    ta.value = 'hello world'
    ta.scrollTop = 2
    ta.scrollLeft = 3

    const pos = getTextareaCaretViewportPosition(ta, 5)
    expect(pos).toEqual({ top: 110, left: 81, height: 20 })
  })

  it('falls back to selectionEnd when caretPos is not a number', () => {
    const ta = document.createElement('textarea')
    ta.value = 'abc'
    ta.selectionEnd = 2
    const pos = getTextareaCaretViewportPosition(ta)
    expect(pos).toEqual({ top: 112, left: 84, height: 20 })
  })

  it('falls back to default line height when computed lineHeight is not numeric', () => {
    window.getComputedStyle = vi.fn(() => ({
      lineHeight: '',
    }))

    const ta = document.createElement('textarea')
    ta.value = ''
    ta.selectionEnd = 0
    const pos = getTextareaCaretViewportPosition(ta)
    expect(pos.height).toBe(18)
  })
})
