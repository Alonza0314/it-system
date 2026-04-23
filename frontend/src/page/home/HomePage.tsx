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
          <p className={styles.badge}>Framework</p>
          <h1 className={styles.brand}>Starter Console</h1>

          <nav className={styles.nav}>
            <a className={styles.navItem} href="#">Home</a>
            <a className={styles.navItem} href="#">Module A</a>
            <a className={styles.navItem} href="#">Module B</a>
          </nav>
        </div>

        <div className={styles.logoutWrap}>
          <Button variant="secondary" onClick={handleLogout}>Logout</Button>
        </div>
      </aside>

      <main className={styles.content}>
        <header className={styles.header}>
          <h2>Home</h2>
          <p>A clean framework canvas ready for your features.</p>
        </header>

        <section className={styles.cardGrid}>
          <article className={styles.card}>
            <h3>Widget Area</h3>
            <p>Place dashboard cards, tables, or charts here.</p>
          </article>
          <article className={styles.card}>
            <h3>Feature Area</h3>
            <p>Use this section as a starting point for module pages.</p>
          </article>
        </section>
      </main>
    </div>
  )
}
