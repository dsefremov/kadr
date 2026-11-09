import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import App from './App'

describe('App', () => {
  it('renders placeholder shell', () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() =>
        Promise.resolve({
          json: () => Promise.resolve({ status: 'ok', build: { version: 'stage0' } }),
        } as Response),
      ),
    )
    render(<App />)
    expect(screen.getByTestId('kadr-app')).toBeTruthy()
    expect(screen.getByRole('heading', { name: 'Кадр' })).toBeTruthy()
  })
})
