import { useNavigate } from 'react-router-dom'
import Button from '../../components/button/button'
import styles from './home-page.module.css'

export default function HomePage() {
  const navigate = useNavigate()

  function handleLogout() {
    localStorage.removeItem('token')
    navigate('/login', { replace: true })
  }

  return (
    <div className={styles.layout}>
      <aside className={styles.sidebar}>
        <div>
          <p className={styles.badge}>free5GC</p>
          <h1 className={styles.brand}>IT System</h1>

          <nav className={styles.nav}>
            <a className={styles.navItem} href="#">Dashboard</a>
          </nav>
        </div>

        <div className={styles.logoutWrap}>
          <Button variant="secondary" onClick={handleLogout}>Logout</Button>
        </div>
      </aside>

      <main className={styles.content}>
        <header className={styles.header}>
          <h2>Dashboard</h2>
        </header>
      </main>
    </div>
  )
}
