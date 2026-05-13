import { useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import {
  Configuration,
  DefaultApi,
  ResponseGetTaskStatusEnum,
  type ResponseGetTask,
} from '../../api'
import { getUserHeader } from '../../utils/auth'
import NotificationContainer from '../../components/notifications/NotificationContainer'
import { useNotifications } from '../../hooks/useNotifications'
import Modal from '../../components/modal/modal'
import Button from '../../components/button/button'
import styles from './task-detail-page.module.css'

const apiBasePath = import.meta.env.VITE_API_BASE_URL || `${window.location.protocol}//${window.location.hostname}:8888`

function formatCreateTime(unixTime?: number) {
  if (!unixTime) {
    return '-'
  }

  return new Date(unixTime * 1000).toLocaleString()
}

function normalizeStatus(status?: ResponseGetTaskStatusEnum) {
  if (status === ResponseGetTaskStatusEnum.Success) {
    return 'success'
  }
  if (status === ResponseGetTaskStatusEnum.Failed) {
    return 'failed'
  }
  if (status === ResponseGetTaskStatusEnum.Timeout) {
    return 'timeout'
  }
  if (status === ResponseGetTaskStatusEnum.Running) {
    return 'running'
  }
  if (status === ResponseGetTaskStatusEnum.Pending) {
    return 'pending'
  }
  if (status === ResponseGetTaskStatusEnum.Canceled) {
    return 'canceled'
  }
  return 'unknown'
}

interface TaskTestItem {
  name: string
  status?: string
}

function getTaskTestOrder(name: string) {
  const normalized = name.toLowerCase()

  if (normalized === 'prepare_free5gc' || normalized === 'preprare_free5gc') {
    return 0
  }
  if (normalized === 'fetch_prs') {
    return 1
  }
  if (normalized === 'make_nf') {
    return 2
  }
  if (normalized === 'cleanup') {
    return 4
  }

  return 3
}

function normalizeTestStatus(status?: string) {
  const normalized = (status || '').toLowerCase()
  if (normalized.includes('success')) {
    return 'success'
  }
  if (normalized.includes('fail')) {
    return 'failed'
  }
  if (normalized.includes('timeout')) {
    return 'timeout'
  }
  if (normalized.includes('running')) {
    return 'running'
  }
  if (normalized.includes('pending')) {
    return 'pending'
  }
  if (normalized.includes('cancel')) {
    return 'canceled'
  }
  return 'unknown'
}

function normalizeRawLog(log?: string) {
  if (!log) {
    return ''
  }

  return log
    .replace(/\r\n/g, '\n')
    .replace(/[\u0000-\u0008\u000B\u000C\u000E-\u001A\u001C-\u001F\u007F]/g, '')
}

interface AnsiSegment {
  text: string
  color?: string
  bold?: boolean
}

const ansiColorMap: Record<number, string> = {
  30: '#111827',
  31: '#dc2626',
  32: '#16a34a',
  33: '#ca8a04',
  34: '#2563eb',
  35: '#a21caf',
  36: '#0891b2',
  37: '#d1d5db',
  90: '#6b7280',
  91: '#f87171',
  92: '#4ade80',
  93: '#facc15',
  94: '#60a5fa',
  95: '#e879f9',
  96: '#22d3ee',
  97: '#f9fafb',
}

function parseAnsiLog(log: string): AnsiSegment[] {
  const segments: AnsiSegment[] = []
  const ansiRegex = /\u001b\[([0-9;]*)m/g

  let cursor = 0
  let currentColor: string | undefined
  let currentBold = false

  let match: RegExpExecArray | null
  while ((match = ansiRegex.exec(log)) !== null) {
    const fullMatch = match[0]
    const params = match[1]
    const matchIndex = match.index

    if (matchIndex > cursor) {
      segments.push({
        text: log.slice(cursor, matchIndex),
        color: currentColor,
        bold: currentBold,
      })
    }

    const codes = params.length === 0
      ? [0]
      : params.split(';').map((value) => Number(value) || 0)

    for (const code of codes) {
      if (code === 0) {
        currentColor = undefined
        currentBold = false
      } else if (code === 1) {
        currentBold = true
      } else if (code === 22) {
        currentBold = false
      } else if (code === 39) {
        currentColor = undefined
      } else if (ansiColorMap[code]) {
        currentColor = ansiColorMap[code]
      }
    }

    cursor = matchIndex + fullMatch.length
  }

  if (cursor < log.length) {
    segments.push({
      text: log.slice(cursor),
      color: currentColor,
      bold: currentBold,
    })
  }

  return segments
}

export default function TaskDetailPage() {
  const navigate = useNavigate()
  const { id } = useParams()
  const [searchParams] = useSearchParams()
  const { errors, successes, addError, addSuccess, removeNotification } = useNotifications()

  const [task, setTask] = useState<ResponseGetTask | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [isCancelling, setIsCancelling] = useState(false)
  const [isCancelModalOpen, setIsCancelModalOpen] = useState(false)
  const [isLogModalOpen, setIsLogModalOpen] = useState(false)
  const [isLogLoading, setIsLogLoading] = useState(false)
  const [logContent, setLogContent] = useState('')
  const [activeTestName, setActiveTestName] = useState('')

  const taskId = Number(id)
  const fromQueue = searchParams.get('from')
  const canCancelTask = fromQueue !== 'ongoing' && fromQueue !== 'history'
  const taskStatus = normalizeStatus(task?.status)
  const tests = useMemo<TaskTestItem[]>(() => {
    const rawTests = (task?.tests || []) as unknown[]

    return rawTests
      .map((item, index) => {
        if (typeof item === 'string') {
          return { name: item, index }
        }

        if (typeof item !== 'object' || item === null) {
          return null
        }

        const typed = item as { name?: string; testName?: string; status?: string }
        const name = typed.name || typed.testName || ''
        if (!name) {
          return null
        }

        return {
          name,
          status: typed.status,
          index,
        }
      })
      .filter((test): test is TaskTestItem & { index: number } => Boolean(test))
      .sort((a, b) => {
        const orderA = getTaskTestOrder(a.name)
        const orderB = getTaskTestOrder(b.name)

        if (orderA !== orderB) {
          return orderA - orderB
        }

        return a.index - b.index
      })
      .map(({ index: _index, ...test }) => test)
  }, [task])
  const renderedLogSegments = useMemo(() => parseAnsiLog(logContent), [logContent])

  const api = useMemo(() => new DefaultApi(new Configuration({
    basePath: apiBasePath,
    accessToken: () => localStorage.getItem('token') || '',
  })), [])

  async function fetchTaskDetail(showLoading: boolean) {
    if (showLoading) {
      setIsLoading(true)
    }

    try {
      const response = await api.getTask(taskId, {
        headers: getUserHeader(),
      })
      setTask(response.data)
    } catch (error: unknown) {
      const message =
        typeof error === 'object' &&
        error !== null &&
        'response' in error &&
        typeof (error as { response?: { data?: { message?: string } } }).response?.data?.message === 'string'
          ? (error as { response?: { data?: { message?: string } } }).response?.data?.message || 'Failed to load task detail'
          : 'Failed to load task detail'
      addError(message)
    } finally {
      if (showLoading) {
        setIsLoading(false)
      }
    }
  }

  useEffect(() => {
    if (!Number.isFinite(taskId) || taskId <= 0) {
      addError('Invalid task id')
      navigate('/test', { replace: true })
      return
    }

    void fetchTaskDetail(true)

    const interval = window.setInterval(() => {
      void fetchTaskDetail(false)
    }, 5000)

    return () => {
      window.clearInterval(interval)
    }
  }, [api, taskId, navigate, addError])

  async function handleCancelTask() {
    if (!Number.isFinite(taskId) || taskId <= 0) {
      addError('Invalid task id')
      return
    }

    setIsCancelling(true)
    try {
      const response = await api.cancelTask(taskId, {
        headers: getUserHeader(),
      })
      addSuccess(response.data.message || 'Task cancelled successfully')
      navigate('/test')
    } catch (error: unknown) {
      const message =
        typeof error === 'object' &&
        error !== null &&
        'response' in error &&
        typeof (error as { response?: { data?: { message?: string } } }).response?.data?.message === 'string'
          ? (error as { response?: { data?: { message?: string } } }).response?.data?.message || 'Failed to cancel task'
          : 'Failed to cancel task'
      addError(message)
    } finally {
      setIsCancelling(false)
    }
  }

  function openCancelModal() {
    setIsCancelModalOpen(true)
  }

  function closeCancelModal() {
    setIsCancelModalOpen(false)
  }

  function closeLogModal() {
    setIsLogModalOpen(false)
  }

  async function handleOpenTestLog(testName: string) {
    if (!Number.isFinite(taskId) || taskId <= 0) {
      addError('Invalid task id')
      return
    }

    setActiveTestName(testName)
    setLogContent('')
    setIsLogModalOpen(true)
    setIsLogLoading(true)

    try {
      const response = await api.getTestLog(
        taskId,
        testName,
        {
          headers: getUserHeader(),
        },
      )

      setLogContent(normalizeRawLog(response.data.log || ''))
    } catch (error: unknown) {
      const message =
        typeof error === 'object' &&
        error !== null &&
        'response' in error &&
        typeof (error as { response?: { data?: { message?: string } } }).response?.data?.message === 'string'
          ? (error as { response?: { data?: { message?: string } } }).response?.data?.message || 'Failed to load test log'
          : 'Failed to load test log'
      addError(message)
      setLogContent('')
    } finally {
      setIsLogLoading(false)
    }
  }

  return (
    <section className={styles.page}>
      <NotificationContainer
        errors={errors}
        successes={successes}
        onClose={removeNotification}
      />

      <header className={styles.header}>
        <div>
          <h2>Task Detail</h2>
          <p>
            Task #{Number.isFinite(taskId) ? taskId : '-'}
            {taskStatus !== 'unknown' && (
              <span className={`${styles.statusBadge} ${styles[`status${taskStatus[0].toUpperCase()}${taskStatus.slice(1)}`]}`}>
                {taskStatus}
              </span>
            )}
          </p>
        </div>
        <div className={styles.actions}>
          <Button variant="secondary" onClick={() => navigate('/test')}>Back</Button>
          {canCancelTask && (
            <Button onClick={openCancelModal} disabled={isCancelling || isLoading}>
              {isCancelling ? 'Cancelling...' : 'Cancel Task'}
            </Button>
          )}
        </div>
      </header>

      <article className={styles.card}>
        {isLoading ? (
          <p className={styles.loading}>Loading task detail...</p>
        ) : (
          <>
            <div className={styles.metaGrid}>
              <p><strong>ID:</strong> {task?.id ?? '-'}</p>
              <p><strong>Username:</strong> {task?.username || '-'}</p>
              <p><strong>Create Time:</strong> {formatCreateTime(task?.createTime)}</p>
            </div>

            <section className={styles.section}>
              <h3>NF PR List</h3>
              {task?.nfPrList && task.nfPrList.length > 0 ? (
                <ul className={styles.tagList}>
                  {task.nfPrList.map((item) => (
                    <li key={`${item.nfName}-${item.pr}`} className={styles.tag}>
                      {item.nfName.toUpperCase()} / PR #{item.pr}
                    </li>
                  ))}
                </ul>
              ) : (
                <p className={styles.empty}>No NF PR data</p>
              )}
            </section>

            <section className={styles.section}>
              <h3>Tests</h3>
              {tests.length > 0 ? (
                <div className={styles.tableWrap}>
                  <table className={styles.table}>
                    <thead>
                      <tr>
                        <th>Test</th>
                        <th>Status</th>
                        <th>Action</th>
                      </tr>
                    </thead>
                    <tbody>
                      {tests.map((test) => {
                        const status = normalizeTestStatus(test.status)
                        const canViewLog = status === 'success' || status === 'failed' || status === 'timeout'

                        return (
                        <tr key={test.name}>
                          <td>{test.name}</td>
                          <td>
                            <span className={`${styles.testStatus} ${styles[`status${status[0].toUpperCase()}${status.slice(1)}`]}`}>
                              {status}
                            </span>
                          </td>
                          <td>
                            <button
                              type="button"
                              className={styles.logButton}
                              onClick={() => {
                                if (!canViewLog) {
                                  return
                                }
                                handleOpenTestLog(test.name)
                              }}
                              disabled={!canViewLog}
                            >
                              View Log
                            </button>
                          </td>
                        </tr>
                        )
                      })}
                    </tbody>
                  </table>
                </div>
              ) : (
                <p className={styles.empty}>No tests</p>
              )}
            </section>
          </>
        )}
      </article>

      {canCancelTask && (
        <Modal
          isOpen={isCancelModalOpen}
          onClose={closeCancelModal}
          title="Confirm Cancel Task"
          onSubmit={handleCancelTask}
          submitText={isCancelling ? 'Cancelling...' : 'Confirm Cancel'}
          submitDisabled={isCancelling || isLoading}
        >
          <p className={styles.confirmMessage}>
            Are you sure you want to cancel task "#{taskId}"?
          </p>
        </Modal>
      )}

      {isLogModalOpen && (
        <div className={styles.logOverlay} onClick={closeLogModal}>
          <div className={styles.logModal} onClick={(event) => event.stopPropagation()}>
            <div className={styles.logModalHeader}>
              <h3>
                Test Log: {activeTestName || '-'}
              </h3>
              <Button variant="secondary" onClick={closeLogModal}>Close</Button>
            </div>

            <div className={styles.logContentWrap}>
              {isLogLoading ? (
                <p className={styles.loading}>Loading test log...</p>
              ) : (
                <pre className={styles.logContent}>
                  {logContent
                    ? renderedLogSegments.map((segment, index) => (
                        <span
                          key={`${index}-${segment.text.length}`}
                          style={{
                            color: segment.color,
                            fontWeight: segment.bold ? 700 : 400,
                          }}
                        >
                          {segment.text}
                        </span>
                      ))
                    : 'No log content'}
                </pre>
              )}
            </div>
          </div>
        </div>
      )}
    </section>
  )
}
