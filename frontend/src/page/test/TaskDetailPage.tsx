import { useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import {
  Configuration,
  DefaultApi,
  ResponseGetTaskStatusEnum,
  TaskTestResultStatusEnum,
  type ResponseGetTask,
  type TaskTestResult,
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

function normalizeTestStatus(status?: TaskTestResultStatusEnum) {
  if (status === TaskTestResultStatusEnum.Success) {
    return 'success'
  }
  if (status === TaskTestResultStatusEnum.Failed) {
    return 'failed'
  }
  if (status === TaskTestResultStatusEnum.Running) {
    return 'running'
  }
  if (status === TaskTestResultStatusEnum.Pending) {
    return 'pending'
  }
  if (status === TaskTestResultStatusEnum.Canceled) {
    return 'canceled'
  }
  return 'unknown'
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

  const taskId = Number(id)
  const fromQueue = searchParams.get('from')
  const canCancelTask = fromQueue !== 'ongoing' && fromQueue !== 'history'
  const taskStatus = normalizeStatus(task?.status)
  const tests = useMemo<TaskTestResult[]>(() => task?.tests || [], [task])

  const api = useMemo(() => new DefaultApi(new Configuration({
    basePath: apiBasePath,
    accessToken: () => localStorage.getItem('token') || '',
  })), [])

  useEffect(() => {
    if (!Number.isFinite(taskId) || taskId <= 0) {
      addError('Invalid task id')
      navigate('/test', { replace: true })
      return
    }

    setIsLoading(true)
    api.getTask(taskId, {
      headers: getUserHeader(),
    })
      .then((response) => {
        setTask(response.data)
      })
      .catch((error: unknown) => {
        const message =
          typeof error === 'object' &&
          error !== null &&
          'response' in error &&
          typeof (error as { response?: { data?: { message?: string } } }).response?.data?.message === 'string'
            ? (error as { response?: { data?: { message?: string } } }).response?.data?.message || 'Failed to load task detail'
            : 'Failed to load task detail'
        addError(message)
      })
      .finally(() => {
        setIsLoading(false)
      })
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

  function handleOpenTestLog(testName: string) {
    addError(`Test log API is not implemented yet: ${testName}`)
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
                              onClick={() => handleOpenTestLog(test.name)}
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
    </section>
  )
}
