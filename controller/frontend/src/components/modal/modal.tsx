import type { ReactNode } from 'react'
import styles from './modal.module.css'
import Button from '../button/button'

interface ModalProps {
  isOpen: boolean
  onClose: () => void
  title: string
  children: ReactNode
  onSubmit?: () => void
  submitText?: string
  cancelText?: string
  submitDisabled?: boolean
}

export default function Modal({ 
  isOpen, 
  onClose, 
  title, 
  children,
  onSubmit,
  submitText = 'Submit',
  cancelText = 'Cancel',
  submitDisabled = false,
}: ModalProps) {
  if (!isOpen) return null

  return (
    <div className={styles.overlay}>
      <div className={styles.modal}>
        <div className={styles.header}>
          <h2 className={styles.title}>{title}</h2>
        </div>
        <div className={styles.body}>
          {children}
        </div>
        <div className={styles.footer}>
          <Button variant="secondary" onClick={onClose}>
            {cancelText}
          </Button>
          {onSubmit && (
            <Button onClick={onSubmit} disabled={submitDisabled}>
              {submitText}
            </Button>
          )}
        </div>
      </div>
    </div>
  )
}
