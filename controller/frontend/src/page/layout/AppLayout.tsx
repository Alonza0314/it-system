import { NavLink, Outlet, useNavigate } from 'react-router-dom'
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
      <header className={styles.globalNav}>
        <div className={styles.globalNavInner}>
          <div className={styles.brandWrap}>
            <p className={styles.badge}>free5GC</p>
            <h1 className={styles.brand}>IT System</h1>
          </div>

          <button type="button" className={styles.logoutButton} onClick={handleLogout}>
            Logout
          </button>
        </div>
      </header>

      <div className={styles.subNav}>
        <div className={styles.subNavInner}>
          <nav className={styles.nav}>
            <NavLink
              end
              to="/"
              className={({ isActive }) => `${styles.navItem} ${isActive ? styles.navItemActive : ''}`}
            >
              Dashboard
            </NavLink>
            <NavLink
              to="/runner"
              className={({ isActive }) => `${styles.navItem} ${isActive ? styles.navItemActive : ''}`}
            >
              Runner
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
              to="/history"
              className={({ isActive }) => `${styles.navItem} ${isActive ? styles.navItemActive : ''}`}
            >
              History
            </NavLink>
            <NavLink
              to="/tenant"
              className={({ isActive }) => `${styles.navItem} ${isActive ? styles.navItemActive : ''}`}
            >
              Tenant
            </NavLink>
          </nav>
        </div>
      </div>

      <main className={styles.content}>
        <div className={styles.contentInner}>
          <Outlet />
        </div>
      </main>
    </div>
  )
}
