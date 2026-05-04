import { useEffect } from 'react'
import styles from './successBox.module.css'

interface SuccessMessage {
  id: string
  message: string
  timestamp: number
}

interface SuccessBoxProps {
  successes: SuccessMessage[]
  onClose: (id: string) => void
  duration?: number
}

function SingleSuccess({ 
  success, 
  onClose, 
  duration = 3000,
  index
}: { 
  success: SuccessMessage
  onClose: (id: string) => void
  duration: number
  index: number
}) {
  useEffect(() => {
    const timer = setTimeout(() => {
      onClose(success.id)
    }, duration)
    return () => clearTimeout(timer)
  }, [success.id, duration, onClose])

  return (
    <div 
      className={styles.successBox} 
      style={{ 
        top: `${(index * 4.5) + 1}rem`,
        animationDelay: `${index * 0.05}s`
      }}
    >
      <span className={styles.message}>{success.message}</span>
      <button 
        type="button" 
        className={styles.closeButton}
        onClick={() => onClose(success.id)}
        aria-label="Close success message"
      >
        x
      </button>
    </div>
  )
}

export default function SuccessBox({ successes, onClose, duration = 3000 }: SuccessBoxProps) {
  return (
    <div className={styles.successBoxContainer}>
      {successes.map((success, index) => (
        <SingleSuccess
          key={success.id}
          success={success}
          onClose={onClose}
          duration={duration}
          index={index}
        />
      ))}
    </div>
  )
}
