import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTestcaseContext } from '../../context/testcase-context'
import styles from './home-page.module.css'

export default function HomePage() {
  const navigate = useNavigate()
  const { testcases, hasLoaded, isLoading, refreshTestcases } = useTestcaseContext()

  const testcaseCountLabel = isLoading && !hasLoaded ? 'Loading...' : `${testcases.length}`

  async function handleRefresh() {
    await refreshTestcases()
  }

  useEffect(() => {
    if (!hasLoaded) {
      refreshTestcases().catch(() => undefined)
    }
  }, [hasLoaded, refreshTestcases])

  return (
    <section className={styles.dashboard}>
      <header className={styles.header}>
        <h2>free5GC Integratioon Test System</h2>
        <button type="button" className={styles.refreshButton} onClick={handleRefresh}>
          Refresh
        </button>
      </header>

      <section className={styles.cardGrid}>
        <article className={styles.card}>
          <h3>Total Testcases</h3>
          <p className={styles.metric}>{testcaseCountLabel}</p>
        </article>

        <article className={styles.card}>
          <h3>Manage Testcases</h3>
          <p>View, add, and delete testcases in one place.</p>
          <button
            type="button"
            className={styles.linkButton}
            onClick={() => navigate('/testcase')}
          >
            Go to Testcase Page
          </button>
        </article>
      </section>
    </section>
  )
}
