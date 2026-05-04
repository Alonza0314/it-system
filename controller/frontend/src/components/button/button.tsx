import styles from './button.module.css'

interface ButtonProps {
  children: React.ReactNode
  onClick?: () => void
  variant?: 'primary' | 'secondary'
  type?: 'button' | 'submit' | 'reset'
  disabled?: boolean
}

export default function Button({ 
  children, 
  onClick, 
  variant = 'primary',
  type = 'button',
  disabled = false,
}: ButtonProps) {
  return (
    <button
      type={type}
      className={`${styles.button} ${styles[variant]}`}
      onClick={onClick}
      disabled={disabled}
    >
      {children}
    </button>
  )
}
