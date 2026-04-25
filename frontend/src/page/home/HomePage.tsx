import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTestcaseContext } from '../../context/testcase-context'
import { useTenantContext } from '../../context/tenant-context'
import { Configuration, DefaultApi } from '../../api'
import { getUserHeader } from '../../utils/auth'
import styles from './home-page.module.css'

const apiBasePath = import.meta.env.VITE_API_BASE_URL || `${window.location.protocol}//${window.location.hostname}:8888`

export default function HomePage() {
  const navigate = useNavigate()
  const { testcases, isLoading, refreshTestcases } = useTestcaseContext()
  const {
    tenants,
    isLoading: isTenantLoading,
    hasNoPermission,
    refreshTenants,
  } = useTenantContext()
  const [isTaskLoading, setIsTaskLoading] = useState(false)
  const [pendingTaskCount, setPendingTaskCount] = useState(0)
  const [ongoingTaskCount, setOngoingTaskCount] = useState(0)

  const api = useMemo(() => new DefaultApi(new Configuration({
    basePath: apiBasePath,
    accessToken: () => localStorage.getItem('token') || '',
  })), [])

  const testcaseCountLabel = isLoading
    ? 'Loading...'
    : `${testcases.length}`
  const tenantCountLabel = isTenantLoading
    ? 'Loading...'
    : hasNoPermission
      ? 'No Access'
      : `${tenants.length}`
  const taskCountLabel = isTaskLoading
    ? 'Loading...'
    : `${pendingTaskCount + ongoingTaskCount}`

  async function refreshTasks() {
    setIsTaskLoading(true)

    try {
      const response = await api.getTasks({
        headers: getUserHeader(),
      })
      setPendingTaskCount(response.data.pendingTask?.length || 0)
      setOngoingTaskCount(response.data.ongoingTask?.length || 0)
    } catch {
      setPendingTaskCount(0)
      setOngoingTaskCount(0)
    } finally {
      setIsTaskLoading(false)
    }
  }

  async function handleRefresh() {
    await Promise.all([
      refreshTestcases(),
      refreshTenants(),
      refreshTasks(),
    ])
  }

  useEffect(() => {
    refreshTestcases().catch(() => undefined)
    refreshTenants().catch(() => undefined)
    refreshTasks().catch(() => undefined)
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

        <button
          type="button"
          className={`${styles.cardButton} ${styles.taskCardButton}`}
          onClick={() => navigate('/test')}
        >
          <article className={`${styles.card} ${styles.taskCard}`}>
            <div className={styles.taskHeader}>
              <h3>Tasks</h3>
              <p className={styles.metric}>{taskCountLabel}</p>
            </div>

            <div className={styles.queueSummary}>
              <section className={styles.queueBlock}>
                <p className={styles.queueLabel}>Pending Queue</p>
                <p className={styles.queueValue}>{isTaskLoading ? 'Loading...' : pendingTaskCount}</p>
              </section>

              <section className={styles.queueBlock}>
                <p className={styles.queueLabel}>Ongoing Queue</p>
                <p className={styles.queueValue}>{isTaskLoading ? 'Loading...' : ongoingTaskCount}</p>
              </section>
            </div>

            <p className={styles.subMetric}>Click to open Test page and view task cards.</p>
          </article>
        </button>
      </section>
    </section>
  )
}
