import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import Button from '../../components/button/button'
import styles from './app-layout.module.css'

export default function AppLayout() {
  const navigate = useNavigate()

  function handleLogout() {
    localStorage.removeItem('token')
    localStorage.removeItem('username')
    navigate('/login', { replace: true })
  }

  return (
    <div className={styles.layout}>
      <aside className={styles.sidebar}>
        <div>
          <p className={styles.badge}>free5GC</p>
          <h1 className={styles.brand}>IT System</h1>

          <nav className={styles.nav}>
            <NavLink
              end
              to="/"
              className={({ isActive }) => `${styles.navItem} ${isActive ? styles.navItemActive : ''}`}
            >
              Dashboard
            </NavLink>
            <NavLink
              to="/testcase"
              className={({ isActive }) => `${styles.navItem} ${isActive ? styles.navItemActive : ''}`}
            >
              Testcase
            </NavLink>
            <NavLink
              to="/test"
              className={({ isActive }) => `${styles.navItem} ${isActive ? styles.navItemActive : ''}`}
            >
              Test
            </NavLink>
            <NavLink
              to="/tenant"
              className={({ isActive }) => `${styles.navItem} ${isActive ? styles.navItemActive : ''}`}
            >
              Tenant
            </NavLink>
          </nav>
        </div>

        <div className={styles.logoutWrap}>
          <Button variant="secondary" onClick={handleLogout}>Logout</Button>
        </div>
      </aside>

      <main className={styles.content}>
        <Outlet />
      </main>
    </div>
  )
}
