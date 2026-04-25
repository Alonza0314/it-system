import { useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Configuration, DefaultApi, type ResponseGetTask } from '../../api'
import { getUserHeader } from '../../utils/auth'
import NotificationContainer from '../../components/notifications/NotificationContainer'
import { useNotifications } from '../../hooks/useNotifications'
import Button from '../../components/button/button'
import styles from './task-detail-page.module.css'

const apiBasePath = import.meta.env.VITE_API_BASE_URL || `${window.location.protocol}//${window.location.hostname}:8888`

function formatCreateTime(unixTime?: number) {
  if (!unixTime) {
    return '-'
  }

  return new Date(unixTime * 1000).toLocaleString()
}

export default function TaskDetailPage() {
  const navigate = useNavigate()
  const { id } = useParams()
  const { errors, successes, addError, addSuccess, removeNotification } = useNotifications()

  const [task, setTask] = useState<ResponseGetTask | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [isCancelling, setIsCancelling] = useState(false)

  const taskId = Number(id)

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
          <p>Task #{Number.isFinite(taskId) ? taskId : '-'}</p>
        </div>
        <div className={styles.actions}>
          <Button variant="secondary" onClick={() => navigate('/test')}>Back</Button>
          <Button onClick={handleCancelTask} disabled={isCancelling || isLoading}>
            {isCancelling ? 'Cancelling...' : 'Cancel Task'}
          </Button>
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
              <h3>Tests</h3>
              {task?.tests && task.tests.length > 0 ? (
                <ul className={styles.tagList}>
                  {task.tests.map((testName) => (
                    <li key={testName} className={styles.tag}>{testName}</li>
                  ))}
                </ul>
              ) : (
                <p className={styles.empty}>No tests</p>
              )}
            </section>

            <section className={styles.section}>
              <h3>NF PR List</h3>
              {task?.nfPrList && task.nfPrList.length > 0 ? (
                <div className={styles.tableWrap}>
                  <table className={styles.table}>
                    <thead>
                      <tr>
                        <th>NF</th>
                        <th>PR</th>
                      </tr>
                    </thead>
                    <tbody>
                      {task.nfPrList.map((item) => (
                        <tr key={`${item.nfName}-${item.pr}`}>
                          <td>{item.nfName}</td>
                          <td>{item.pr}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : (
                <p className={styles.empty}>No NF PR data</p>
              )}
            </section>
          </>
        )}
      </article>
    </section>
  )
}
