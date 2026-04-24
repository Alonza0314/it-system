import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTestcaseContext } from '../../context/testcase-context'
import { useTenantContext } from '../../context/tenant-context'
import styles from './home-page.module.css'

export default function HomePage() {
  const navigate = useNavigate()
  const { testcases, isLoading, refreshTestcases } = useTestcaseContext()
  const {
    tenants,
    isLoading: isTenantLoading,
    hasNoPermission,
    refreshTenants,
  } = useTenantContext()

  const testcaseCountLabel = isLoading
    ? 'Loading...'
    : `${testcases.length}`
  const tenantCountLabel = isTenantLoading
    ? 'Loading...'
    : hasNoPermission
      ? 'No Access'
      : `${tenants.length}`

  async function handleRefresh() {
    await Promise.all([
      refreshTestcases(),
      refreshTenants(),
    ])
  }

  useEffect(() => {
    refreshTestcases().catch(() => undefined)
    refreshTenants().catch(() => undefined)
  }, [refreshTestcases, refreshTenants])

  return (
    <section className={styles.dashboard}>
      <header className={styles.header}>
        <h2>free5GC Integratioon Test System</h2>
        <button type="button" className={styles.refreshButton} onClick={handleRefresh}>
          Refresh
        </button>
      </header>

      <section className={styles.cardGrid}>
        <button
          type="button"
          className={styles.cardButton}
          onClick={() => navigate('/testcase')}
        >
          <article className={styles.card}>
            <h3>Testcases</h3>
            <p className={styles.metric}>{testcaseCountLabel}</p>
          </article>
        </button>

        <button
          type="button"
          className={styles.cardButton}
          onClick={() => navigate('/tenant')}
        >
          <article className={styles.card}>
            <h3>Tenants</h3>
            <p className={styles.metric}>{tenantCountLabel}</p>
          </article>
        </button>
      </section>
    </section>
  )
}
