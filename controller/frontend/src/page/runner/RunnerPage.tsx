import { Suspense, lazy, useCallback, useEffect, useMemo, useState } from 'react'
import NotificationContainer from '../../components/notifications/NotificationContainer'
import Modal from '../../components/modal/modal'
import { useNotifications } from '../../hooks/useNotifications'
import { deleteRunner, getRunners, type Runner } from '../../api/runner'
import { isAdmin } from '../../utils/auth'
import styles from './runner-page.module.css'

// xterm.js and the terminal font are loaded only when a terminal is opened
const RunnerTerminalWindow = lazy(() => import('../../components/runnerTerminal/RunnerTerminalWindow'))

function getStatusClass(status: string) {
  const normalized = status.toLowerCase()
  if (normalized === 'running') {
    return styles.statusRunning
  }
  if (normalized === 'idle' || normalized === 'idel') {
    return styles.statusIdle
  }
  return styles.statusOffline
}

export default function RunnerPage() {
  const { errors, successes, addError, addSuccess, removeNotification } = useNotifications()
  const [runners, setRunners] = useState<Runner[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false)
  const [targetDeleteName, setTargetDeleteName] = useState('')
  const [deletingName, setDeletingName] = useState('')
  const [canOpenTerminal] = useState(isAdmin)
  // open terminal windows, most recently focused last
  const [terminals, setTerminals] = useState<string[]>([])
  const [focusOrder, setFocusOrder] = useState<string[]>([])

  const refreshRunners = useCallback(async () => {
    setIsLoading(true)
    try {
      const nextRunners = await getRunners()
      setRunners(nextRunners)
    } catch (error: unknown) {
      const message =
        typeof error === 'object' &&
        error !== null &&
        'response' in error &&
        typeof (error as { response?: { data?: { message?: string } } }).response?.data?.message === 'string'
          ? (error as { response?: { data?: { message?: string } } }).response?.data?.message || 'Failed to load runners'
          : 'Failed to load runners'
      addError(message)
    } finally {
      setIsLoading(false)
    }
  }, [addError])

  useEffect(() => {
    refreshRunners().catch(() => {
      addError('Failed to load runners')
    })

    const timer = window.setInterval(() => {
      refreshRunners().catch(() => {
        addError('Failed to load runners')
      })
    }, 30_000)

    return () => {
      window.clearInterval(timer)
    }
  }, [refreshRunners, addError])

  const counts = useMemo(() => {
    return runners.reduce((acc, runner) => {
      const normalized = runner.status.toLowerCase()
      if (normalized === 'running') {
        acc.running += 1
      } else if (normalized === 'idle' || normalized === 'idel') {
        acc.idle += 1
      } else {
        acc.offline += 1
      }
      return acc
    }, { offline: 0, idle: 0, running: 0 })
  }, [runners])

  function focusTerminal(name: string) {
    setFocusOrder((order) => (order[order.length - 1] === name ? order : [...order.filter((n) => n !== name), name]))
  }

  function openTerminal(name: string) {
    setTerminals((open) => (open.includes(name) ? open : [...open, name]))
    focusTerminal(name)
  }

  function closeTerminal(name: string) {
    setTerminals((open) => open.filter((n) => n !== name))
    setFocusOrder((order) => order.filter((n) => n !== name))
  }

  function openDeleteModal(name: string) {
    setTargetDeleteName(name)
    setIsDeleteModalOpen(true)
  }

  function closeDeleteModal() {
    setTargetDeleteName('')
    setIsDeleteModalOpen(false)
  }

  async function handleDeleteRunner() {
    if (!targetDeleteName) {
      return
    }

    setDeletingName(targetDeleteName)
    try {
      const message = await deleteRunner(targetDeleteName)
      addSuccess(message)
      closeDeleteModal()
      await refreshRunners()
    } catch (error: unknown) {
      const message =
        typeof error === 'object' &&
        error !== null &&
        'response' in error &&
        typeof (error as { response?: { data?: { message?: string } } }).response?.data?.message === 'string'
          ? (error as { response?: { data?: { message?: string } } }).response?.data?.message || 'Failed to delete runner'
          : 'Failed to delete runner'
      addError(message)
    } finally {
      setDeletingName('')
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
        <h2>Runner</h2>
        <button type="button" className={styles.refreshButton} onClick={() => refreshRunners()}>
          Refresh
        </button>
      </header>

      <section className={styles.summaryCard}>
        <h3>Runner Overview</h3>
        <div className={styles.summaryGrid}>
          <div className={styles.summaryItem}>
            <p className={styles.summaryLabel}>Offline</p>
            <p className={styles.summaryValue}>{counts.offline}</p>
          </div>
          <div className={styles.summaryItem}>
            <p className={styles.summaryLabel}>Idle</p>
            <p className={styles.summaryValue}>{counts.idle}</p>
          </div>
          <div className={styles.summaryItem}>
            <p className={styles.summaryLabel}>Running</p>
            <p className={styles.summaryValue}>{counts.running}</p>
          </div>
        </div>
      </section>

      {isLoading && runners.length === 0 ? (
        <p className={styles.hint}>Loading runners...</p>
      ) : (
        <section className={styles.cardGrid}>
          {runners.map((runner) => (
            <article key={`${runner.name}-${runner.ip}`} className={`${styles.runnerCard} ${getStatusClass(runner.status)}`}>
              <div className={styles.cardHeader}>
                <h3>{runner.name}</h3>
                <div className={styles.cardActions}>
                  {canOpenTerminal && (
                    <button
                      type="button"
                      className={styles.terminalButton}
                      onClick={() => openTerminal(runner.name)}
                      aria-label={`Open terminal on runner ${runner.name}`}
                      title="Open terminal"
                    >
                      <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false" className={styles.terminalIcon}>
                        <path d="M3 5h18v14H3V5zm2 2v10h14V7H5zm1.6 1.6L10 12l-3.4 3.4-1.1-1.1L7.8 12 5.5 9.7l1.1-1.1zM11 14.5h6V16h-6v-1.5z" fill="currentColor" />
                      </svg>
                    </button>
                  )}
                  <button
                    type="button"
                    className={styles.deleteButton}
                    onClick={() => openDeleteModal(runner.name)}
                    disabled={deletingName === runner.name}
                    aria-label={`Delete runner ${runner.name}`}
                  >
                    <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false" className={styles.trashIcon}>
                      <path d="M9 3h6l1 2h4v2H4V5h4l1-2zm1 6h2v8h-2V9zm4 0h2v8h-2V9zM7 9h2v8H7V9z" fill="currentColor" />
                    </svg>
                  </button>
                </div>
              </div>

              <div className={styles.cardBody}>
                <p><strong>IP:</strong> {runner.ip}</p>
                <p><strong>Status:</strong> {runner.status}</p>
                <p><strong>Ongoing Task:</strong> {runner.onGoingTask === 0 ? 'No task' : runner.onGoingTask}</p>
              </div>
            </article>
          ))}

          {runners.length === 0 && (
            <p className={styles.hint}>No runners available.</p>
          )}
        </section>
      )}

      <Suspense fallback={null}>
        {terminals.map((name, index) => (
          <RunnerTerminalWindow
            key={name}
            runner={name}
            onGoingTask={runners.find((r) => r.name === name)?.onGoingTask ?? 0}
            offset={index}
            zIndex={1100 + Math.max(0, focusOrder.indexOf(name))}
            onFocus={() => focusTerminal(name)}
            onClose={() => closeTerminal(name)}
          />
        ))}
      </Suspense>

      <Modal
        isOpen={isDeleteModalOpen}
        onClose={closeDeleteModal}
        title="Confirm Delete Runner"
        onSubmit={handleDeleteRunner}
        submitText={deletingName ? 'Deleting...' : 'Confirm Delete'}
        submitDisabled={Boolean(deletingName)}
      >
        <p className={styles.hint}>
          Are you sure you want to delete runner "{targetDeleteName}"?
        </p>
      </Modal>
    </section>
  )
}
