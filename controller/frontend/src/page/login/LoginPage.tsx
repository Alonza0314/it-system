import { useState, type FormEvent } from 'react'
import Button from '../../components/button/button'
import NotificationContainer from '../../components/notifications/NotificationContainer'
import { useNotifications } from '../../hooks/useNotifications'
import { Configuration, DefaultApi } from '../../api'
import { useNavigate } from 'react-router-dom'
import styles from './login-page.module.css'

const apiBasePath = import.meta.env.VITE_API_BASE_URL || `${window.location.protocol}//${window.location.hostname}:8888`
const api = new DefaultApi(new Configuration({ basePath: apiBasePath }))

export default function LoginPage() {
  const navigate = useNavigate()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [isLoading, setIsLoading] = useState(false)

  const { errors, successes, addError, addSuccess, removeNotification } = useNotifications()

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setIsLoading(true)

    try {
      const response = await api.login({ username, password })
      const token = response.data.token || ''
      localStorage.setItem('token', token)
      localStorage.setItem('username', username.trim())
      addSuccess(response.data.message || 'Login successful')
      navigate('/', { replace: true })
    } catch (error: unknown) {
      const message =
        typeof error === 'object' &&
        error !== null &&
        'response' in error &&
        typeof (error as { response?: { data?: { message?: string } } }).response?.data?.message === 'string'
          ? (error as { response?: { data?: { message?: string } } }).response?.data?.message || 'Login failed'
          : 'Login failed'

      addError(message)
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className={styles.page}>
      <NotificationContainer
        errors={errors}
        successes={successes}
        onClose={removeNotification}
      />
      <main className={styles.card}>
        <div className={styles.headerBlock}>
          <p className={styles.kicker}>free5GC</p>
          <h1 className={styles.title}>IT System</h1>
          <p className={styles.subtitle}>Sign in to continue to your control panel.</p>
        </div>

        <form className={styles.form} onSubmit={handleSubmit}>
          <label className={styles.label} htmlFor="username">Username</label>
          <input
            id="username"
            className={styles.input}
            value={username}
            onChange={(event) => setUsername(event.target.value)}
            autoComplete="username"
            required
          />

          <label className={styles.label} htmlFor="password">Password</label>
          <input
            id="password"
            type="password"
            className={styles.input}
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            autoComplete="current-password"
            required
          />

          <div className={styles.actionRow}>
            <Button type="submit" disabled={isLoading}>
              {isLoading ? 'Signing in...' : 'Sign In'}
            </Button>
          </div>
        </form>
      </main>
    </div>
  )
}
