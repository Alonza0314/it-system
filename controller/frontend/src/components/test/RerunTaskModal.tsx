import { useMemo, useState } from 'react'
import {
  Configuration,
  DefaultApi,
  type NfPr,
  type TestDetail,
} from '../../api'
import { getUserHeader } from '../../utils/auth'
import { isPipelineStep } from '../../utils/task'
import Modal from '../modal/modal'
import styles from './rerun-task-modal.module.css'

const apiBasePath = import.meta.env.VITE_API_BASE_URL || `${window.location.protocol}//${window.location.hostname}:8888`

interface RerunTaskModalProps {
  // The task to run again; the modal is closed when it is null
  task: { id?: number; tests?: TestDetail[]; nfPrList?: NfPr[] } | null
  onClose: () => void
  onSuccess: (message: string) => void
  onError: (message: string) => void
}

export default function RerunTaskModal({ task, onClose, onSuccess, onError }: RerunTaskModalProps) {
  const [isSubmitting, setIsSubmitting] = useState(false)

  const api = useMemo(() => new DefaultApi(new Configuration({
    basePath: apiBasePath,
    accessToken: () => localStorage.getItem('token') || '',
  })), [])

  const testcases = (task?.tests || []).filter((test) => !isPipelineStep(test.name))

  async function handleRerun() {
    if (!task?.id) {
      return
    }

    setIsSubmitting(true)
    try {
      const response = await api.rerunTask(task.id, {
        headers: getUserHeader(),
      })
      onSuccess(response.data.message || `Task ${task.id} submitted again`)
      onClose()
    } catch (error: unknown) {
      const message =
        typeof error === 'object' &&
        error !== null &&
        'response' in error &&
        typeof (error as { response?: { data?: { message?: string } } }).response?.data?.message === 'string'
          ? (error as { response?: { data?: { message?: string } } }).response?.data?.message || 'Failed to re-run task'
          : 'Failed to re-run task'
      onError(message)
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <Modal
      isOpen={task !== null}
      onClose={onClose}
      title={`Re-run Task #${task?.id ?? '-'}`}
      onSubmit={handleRerun}
      submitText={isSubmitting ? 'Submitting...' : 'Confirm Re-run'}
      submitDisabled={isSubmitting || !task?.id}
    >
      <div className={styles.body}>
        <p className={styles.title}>
          A new task with the same testcases and NF PRs will be submitted under your account.
          Each PR is fetched again, so its latest commit is tested.
        </p>

        <section className={styles.section}>
          <p className={styles.label}>Testcases</p>
          <div className={styles.chips}>
            {testcases.map((test) => (
              <span key={test.name} className={styles.chip}>{test.name}</span>
            ))}
          </div>
        </section>

        <section className={styles.section}>
          <p className={styles.label}>NF / PR</p>
          {task?.nfPrList && task.nfPrList.length > 0 ? (
            <ul className={styles.list}>
              {task.nfPrList.map((item) => (
                <li key={`${item.nfName}-${item.pr}`}>
                  {item.nfName.toUpperCase()} / PR #{item.pr}
                </li>
              ))}
            </ul>
          ) : (
            <p className={styles.empty}>No NF PR</p>
          )}
        </section>
      </div>
    </Modal>
  )
}
