import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Configuration,
  DefaultApi,
  type TaskDetail,
} from '../../api'
import { getCurrentUsername, getUserHeader } from '../../utils/auth'
import NotificationContainer from '../../components/notifications/NotificationContainer'
import { useNotifications } from '../../hooks/useNotifications'
import styles from './history-page.module.css'

const apiBasePath = import.meta.env.VITE_API_BASE_URL || `${window.location.protocol}//${window.location.hostname}:8888`

// Pipeline steps the runner adds around the selected testcases
const PIPELINE_STEPS = ['prepare_free5gc', 'fetch_prs', 'make_nf', 'cleanup']

const STATUS_OPTIONS = ['running', 'pending', 'success', 'failed', 'timeout']

function formatCreateTime(unixTime: number) {
  if (!unixTime) {
    return '-'
  }

  return new Date(unixTime * 1000).toLocaleString()
}

function getStatusClassName(status: string) {
  return styles[`status${status[0]?.toUpperCase()}${status.slice(1)}`] || styles.statusUnknown
}

function summarizeTests(task: TaskDetail) {
  const tests = task.tests || []
  const testcases = tests.filter((test) => !PIPELINE_STEPS.includes(test.name))

  return {
    passed: testcases.filter((test) => test.status === 'success').length,
    total: testcases.length,
    failed: tests
      .filter((test) => test.status === 'failed' || test.status === 'timeout')
      .map((test) => test.name),
  }
}

// Steps in the order the runner runs them
function orderedSteps(task: TaskDetail) {
  const tests = task.tests || []
  const byName = (name: string) => tests.filter((test) => test.name === name)

  return [
    ...byName('prepare_free5gc'),
    ...byName('fetch_prs'),
    ...byName('make_nf'),
    ...tests.filter((test) => !PIPELINE_STEPS.includes(test.name)),
    ...byName('cleanup'),
  ]
}

// A step gets its status when the runner reports it, so the running step is
// the first one that is still pending
function describeProgress(task: TaskDetail) {
  if (task.status === 'pending') {
    return 'Waiting for a runner'
  }

  const steps = orderedSteps(task)
  const current = steps.findIndex((step) => step.status === 'pending')
  return current === -1 ? 'Finishing' : `Running ${steps[current].name} (${current + 1}/${steps.length})`
}

function getQueueName(task: TaskDetail) {
  if (task.status === 'pending') {
    return 'pending'
  }
  return task.status === 'running' ? 'ongoing' : 'history'
}

function matchesSearch(task: TaskDetail, query: string) {
  if (!query) {
    return true
  }

  const fields = [
    `#${task.id}`,
    ...(task.nfPrList || []).map((item) => `${item.nfName} #${item.pr}`),
    ...(task.tests || []).map((test) => test.name),
  ]

  return fields.some((field) => field.toLowerCase().includes(query))
}

export default function HistoryPage() {
  const navigate = useNavigate()
  const { errors, successes, addError, removeNotification } = useNotifications()

  const [tasks, setTasks] = useState<TaskDetail[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [userFilter, setUserFilter] = useState('')
  const [statusFilter, setStatusFilter] = useState('')
  const [search, setSearch] = useState('')

  const currentUsername = getCurrentUsername()

  const api = useMemo(() => new DefaultApi(new Configuration({
    basePath: apiBasePath,
    accessToken: () => localStorage.getItem('token') || '',
  })), [])

  const usernames = useMemo(
    () => Array.from(new Set(tasks.map((task) => task.username))).sort((a, b) => a.localeCompare(b)),
    [tasks],
  )

  const filteredTasks = useMemo(() => {
    const query = search.trim().toLowerCase()

    return tasks.filter((task) => (
      (!userFilter || task.username === userFilter)
      && (!statusFilter || task.status === statusFilter)
      && matchesSearch(task, query)
    ))
  }, [tasks, userFilter, statusFilter, search])

  function extractErrorMessage(error: unknown, fallback: string) {
    return (
      typeof error === 'object'
      && error !== null
      && 'response' in error
      && typeof (error as { response?: { data?: { message?: string } } }).response?.data?.message === 'string'
    )
      ? (error as { response?: { data?: { message?: string } } }).response?.data?.message || fallback
      : fallback
  }

  async function refreshTasks(options?: { showLoading?: boolean; notifyError?: boolean }) {
    const showLoading = options?.showLoading ?? true
    const notifyError = options?.notifyError ?? true

    if (showLoading) {
      setIsLoading(true)
    }

    try {
      const response = await api.getTasksDetail({
        headers: getUserHeader(),
      })
      const { pendingTask = [], ongoingTask = [], historyTask = [] } = response.data
      setTasks([...pendingTask, ...ongoingTask, ...historyTask].sort((a, b) => b.id - a.id))
    } catch (error: unknown) {
      if (notifyError) {
        addError(extractErrorMessage(error, 'Failed to load tasks'))
      }
    } finally {
      if (showLoading) {
        setIsLoading(false)
      }
    }
  }

  useEffect(() => {
    void refreshTasks({ showLoading: true, notifyError: true })

    const timer = window.setInterval(() => {
      // Auto refresh errors are intentionally silent to avoid notification spam.
      void refreshTasks({ showLoading: false, notifyError: false })
    }, 5000)

    return () => {
      window.clearInterval(timer)
    }
  }, [api, addError])

  return (
    <section className={styles.page}>
      <NotificationContainer
        errors={errors}
        successes={successes}
        onClose={removeNotification}
      />

      <header className={styles.header}>
        <div>
          <h2>History</h2>
          <p>Pending, ongoing and finished tasks with their NF PRs and test results</p>
        </div>
        <div className={styles.headerActions}>
          <button
            type="button"
            className={styles.refreshButton}
            onClick={() => refreshTasks()}
            disabled={isLoading}
          >
            {isLoading ? 'Refreshing...' : 'Refresh'}
          </button>
        </div>
      </header>

      <article className={styles.card}>
        <div className={styles.filters}>
          <label className={styles.filter}>
            <span>User</span>
            <select value={userFilter} onChange={(event) => setUserFilter(event.target.value)}>
              <option value="">All users</option>
              {usernames.map((username) => (
                <option key={username} value={username}>
                  {username === currentUsername ? `${username} (you)` : username}
                </option>
              ))}
            </select>
          </label>

          <label className={styles.filter}>
            <span>Status</span>
            <select value={statusFilter} onChange={(event) => setStatusFilter(event.target.value)}>
              <option value="">All statuses</option>
              {STATUS_OPTIONS.map((status) => (
                <option key={status} value={status}>{status}</option>
              ))}
            </select>
          </label>

          <label className={`${styles.filter} ${styles.searchFilter}`}>
            <span>Search</span>
            <input
              type="search"
              placeholder="Task #, NF, PR # or testcase"
              value={search}
              onChange={(event) => setSearch(event.target.value)}
            />
          </label>
        </div>

        {isLoading && tasks.length === 0 ? (
          <p className={styles.hint}>Loading tasks...</p>
        ) : filteredTasks.length === 0 ? (
          <p className={styles.hint}>{tasks.length === 0 ? 'No tasks' : 'No tasks match the filters'}</p>
        ) : (
          <div className={styles.tableWrap}>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th>Task</th>
                  <th>User</th>
                  <th>Create Time</th>
                  <th>Status</th>
                  <th>NF / PR</th>
                  <th>Tests</th>
                  <th>Action</th>
                </tr>
              </thead>
              <tbody>
                {filteredTasks.map((task) => {
                  const summary = summarizeTests(task)
                  const isActive = task.status === 'pending' || task.status === 'running'

                  return (
                    <tr key={task.id}>
                      <td>#{task.id}</td>
                      <td>{task.username || '-'}</td>
                      <td>{formatCreateTime(task.createTime)}</td>
                      <td>
                        <span className={`${styles.statusBadge} ${getStatusClassName(task.status)}`}>
                          {task.status}
                        </span>
                      </td>
                      <td>
                        {task.nfPrList && task.nfPrList.length > 0 ? (
                          <ul className={styles.tagList}>
                            {task.nfPrList.map((item) => (
                              <li key={`${item.nfName}-${item.pr}`} className={styles.tag}>
                                {item.nfName.toUpperCase()} #{item.pr}
                              </li>
                            ))}
                          </ul>
                        ) : '-'}
                      </td>
                      <td>
                        <p className={styles.testSummary}>
                          {isActive ? describeProgress(task) : `${summary.passed} / ${summary.total} passed`}
                        </p>
                        {summary.failed.length > 0 && (
                          <p className={styles.failedTests}>Failed: {summary.failed.join(', ')}</p>
                        )}
                      </td>
                      <td>
                        <div className={styles.rowActions}>
                          <button
                            type="button"
                            className={styles.actionButton}
                            onClick={() => navigate(`/test/task/${task.id}?from=${getQueueName(task)}`)}
                          >
                            View
                          </button>
                        </div>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
      </article>
    </section>
  )
}
