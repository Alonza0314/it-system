import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTestcaseContext } from '../../context/testcase-context'
import { useTenantContext } from '../../context/tenant-context'
import { Configuration, DefaultApi } from '../../api'
import { getRunners } from '../../api/runner'
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
  const [isRunnerLoading, setIsRunnerLoading] = useState(false)
  const [pendingTaskCount, setPendingTaskCount] = useState(0)
  const [ongoingTaskCount, setOngoingTaskCount] = useState(0)
  const [historyTaskCount, setHistoryTaskCount] = useState(0)
  const [offlineRunnerCount, setOfflineRunnerCount] = useState(0)
  const [idleRunnerCount, setIdleRunnerCount] = useState(0)
  const [runningRunnerCount, setRunningRunnerCount] = useState(0)

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
    : `${pendingTaskCount + ongoingTaskCount + historyTaskCount}`
  const runnerCountLabel = isRunnerLoading
    ? 'Loading...'
    : `${offlineRunnerCount + idleRunnerCount + runningRunnerCount}`

  async function refreshTasks() {
    setIsTaskLoading(true)

    try {
      const response = await api.getTasks({
        headers: getUserHeader(),
      })
      setPendingTaskCount(response.data.pendingTask?.length || 0)
      setOngoingTaskCount(response.data.ongoingTask?.length || 0)
      setHistoryTaskCount((response.data as { historyTask?: unknown[] }).historyTask?.length || 0)
    } catch {
      setPendingTaskCount(0)
      setOngoingTaskCount(0)
      setHistoryTaskCount(0)
    } finally {
      setIsTaskLoading(false)
    }
  }

  async function refreshRunners() {
    setIsRunnerLoading(true)

    try {
      const runners = await getRunners()
      const counts = runners.reduce((acc, runner) => {
        const normalized = runner.status.toLowerCase()
        if (normalized === 'running') {
          acc.running += 1
        } else if (normalized === 'idle' || normalized === 'idel') {
          acc.idle += 1
        } else {
          acc.offline += 1
        }
        return acc
      }, {
        offline: 0,
        idle: 0,
        running: 0,
      })

      setOfflineRunnerCount(counts.offline)
      setIdleRunnerCount(counts.idle)
      setRunningRunnerCount(counts.running)
    } catch {
      setOfflineRunnerCount(0)
      setIdleRunnerCount(0)
      setRunningRunnerCount(0)
    } finally {
      setIsRunnerLoading(false)
    }
  }

  async function handleRefresh() {
    await Promise.all([
      refreshTestcases(),
      refreshTenants(),
      refreshTasks(),
      refreshRunners(),
    ])
  }

  useEffect(() => {
    refreshTestcases().catch(() => undefined)
    refreshTenants().catch(() => undefined)
    refreshTasks().catch(() => undefined)
    refreshRunners().catch(() => undefined)

    const timer = window.setInterval(() => {
      refreshTasks().catch(() => undefined)
      refreshRunners().catch(() => undefined)
    }, 30_000)

    return () => {
      window.clearInterval(timer)
    }
  }, [refreshTestcases, refreshTenants])

  return (
    <section className={styles.dashboard}>
      <header className={styles.topBar}>
        <div>
          <p className={styles.eyebrow}>System Overview</p>
          <h2>free5GC Integration Test System</h2>
          <p className={styles.subtitle}>Track testcase, tenant, task queue, and runner status in one place.</p>
        </div>
        <button type="button" className={styles.refreshButton} onClick={handleRefresh}>
          Refresh
        </button>
      </header>

      <section className={styles.kpiGrid}>
        <button
          type="button"
          className={styles.tileButton}
          onClick={() => navigate('/testcase')}
        >
          <article className={`${styles.kpiCard} ${styles.kpiTestcase}`}>
            <p className={styles.kpiLabel}>Testcases</p>
            <p className={styles.kpiValue}>{testcaseCountLabel}</p>
          </article>
        </button>

        <button
          type="button"
          className={styles.tileButton}
          onClick={() => navigate('/tenant')}
        >
          <article className={`${styles.kpiCard} ${styles.kpiTenant}`}>
            <p className={styles.kpiLabel}>Tenants</p>
            <p className={styles.kpiValue}>{tenantCountLabel}</p>
          </article>
        </button>

        <button
          type="button"
          className={styles.tileButton}
          onClick={() => navigate('/test')}
        >
          <article className={`${styles.kpiCard} ${styles.kpiTask}`}>
            <p className={styles.kpiLabel}>Tasks</p>
            <p className={styles.kpiValue}>{taskCountLabel}</p>
          </article>
        </button>

        <button
          type="button"
          className={styles.tileButton}
          onClick={() => navigate('/runner')}
        >
          <article className={`${styles.kpiCard} ${styles.kpiRunner}`}>
            <p className={styles.kpiLabel}>Runners</p>
            <p className={styles.kpiValue}>{runnerCountLabel}</p>
          </article>
        </button>
      </section>

      <section className={styles.detailGrid}>
        <button
          type="button"
          className={styles.tileButton}
          onClick={() => navigate('/test')}
        >
          <article className={`${styles.detailCard} ${styles.taskPanel}`}>
            <div className={styles.panelHeader}>
              <h3>Task Queue</h3>
              <p className={styles.panelMetric}>{taskCountLabel}</p>
            </div>

            <div className={styles.panelStats}>
              <section className={styles.statBlock}>
                <p className={styles.statLabel}>Pending</p>
                <p className={styles.statValue}>{isTaskLoading ? 'Loading...' : pendingTaskCount}</p>
              </section>

              <section className={styles.statBlock}>
                <p className={styles.statLabel}>Ongoing</p>
                <p className={styles.statValue}>{isTaskLoading ? 'Loading...' : ongoingTaskCount}</p>
              </section>

              <section className={styles.statBlock}>
                <p className={styles.statLabel}>History</p>
                <p className={styles.statValue}>{isTaskLoading ? 'Loading...' : historyTaskCount}</p>
              </section>
            </div>
          </article>
        </button>

        <button
          type="button"
          className={styles.tileButton}
          onClick={() => navigate('/runner')}
        >
          <article className={`${styles.detailCard} ${styles.runnerPanel}`}>
            <div className={styles.panelHeader}>
              <h3>Runner State</h3>
              <p className={styles.panelMetric}>{runnerCountLabel}</p>
            </div>

            <div className={styles.panelStats}>
              <section className={styles.statBlock}>
                <p className={styles.statLabel}>Offline</p>
                <p className={styles.statValue}>{isRunnerLoading ? 'Loading...' : offlineRunnerCount}</p>
              </section>

              <section className={styles.statBlock}>
                <p className={styles.statLabel}>Idle</p>
                <p className={styles.statValue}>{isRunnerLoading ? 'Loading...' : idleRunnerCount}</p>
              </section>

              <section className={styles.statBlock}>
                <p className={styles.statLabel}>Running</p>
                <p className={styles.statValue}>{isRunnerLoading ? 'Loading...' : runningRunnerCount}</p>
              </section>
            </div>
          </article>
        </button>
      </section>
    </section>
  )
}
