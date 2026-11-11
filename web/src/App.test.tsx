import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import App from './App'

describe('App', () => {
  it('renders home shell', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn((url: string) => {
        if (String(url).includes('/api/me')) {
          return Promise.resolve({ status: 401 } as Response)
        }
        return Promise.resolve({ json: () => Promise.resolve({}) } as Response)
      }),
    )
    render(<App />)
    expect(screen.getByTestId('kadr-app')).toBeTruthy()
    expect(await screen.findByRole('heading', { name: 'Моя лента' })).toBeTruthy()
  })
})
