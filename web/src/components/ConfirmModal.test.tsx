import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { ConfirmModal } from './ConfirmModal'

describe('ConfirmModal', () => {
  it('calls cancel on Escape and confirm on primary action', () => {
    const onCancel = vi.fn()
    const onConfirm = vi.fn()
    render(
      <ConfirmModal
        open
        title="Test"
        confirmLabel="OK"
        cancelLabel="No"
        onConfirm={onConfirm}
        onCancel={onCancel}
      >
        <p>Body</p>
      </ConfirmModal>,
    )
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(onCancel).toHaveBeenCalledTimes(1)
    fireEvent.click(screen.getByTestId('confirm-modal-confirm'))
    expect(onConfirm).toHaveBeenCalledTimes(1)
  })
})
