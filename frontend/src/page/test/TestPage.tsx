import { useEffect, useMemo, useState } from 'react'
import { Configuration, DefaultApi, type GetGithubPRsNfEnum, type RequestSubmitTask, type TaskSimple } from '../../api'
import { getUserHeader } from '../../utils/auth'
import NotificationContainer from '../../components/notifications/NotificationContainer'
import { useNotifications } from '../../hooks/useNotifications'
import NfPrSelector, { type PrOption } from '../../components/test/NfPrSelector'
import TaskCard from '../../components/test/TaskCard'
import { useTestcaseContext } from '../../context/testcase-context'
import styles from './test-page.module.css'

interface NfDef {
  label: string
  apiName: string
}

const NF_ORDER: NfDef[] = [
  { label: 'AMF', apiName: 'amf' },
  { label: 'AUSF', apiName: 'ausf' },
  { label: 'BSF', apiName: 'bsf' },
  { label: 'CHF', apiName: 'chf' },
  { label: 'N3IWF', apiName: 'n3iwf' },
  { label: 'NEF', apiName: 'nef' },
  { label: 'NRF', apiName: 'nrf' },
  { label: 'NSSF', apiName: 'nssf' },
  { label: 'PCF', apiName: 'pcf' },
  { label: 'SMF', apiName: 'smf' },
  { label: 'TNGF', apiName: 'tngf' },
  { label: 'UDM', apiName: 'udm' },
  { label: 'UDR', apiName: 'udr' },
  { label: 'UPF', apiName: 'upf' },
]

const apiBasePath = import.meta.env.VITE_API_BASE_URL || `${window.location.protocol}//${window.location.hostname}:8888`

export default function TestPage() {
  const { errors, successes, addError, addSuccess, removeNotification } = useNotifications()
  const { testcases, hasLoaded: hasTestcasesLoaded, refreshTestcases } = useTestcaseContext()

  const [isLoadingTasks, setIsLoadingTasks] = useState(false)
  const [isFormOpen, setIsFormOpen] = useState(false)
  const [isSubmittingTask, setIsSubmittingTask] = useState(false)
  const [prsByNf, setPrsByNf] = useState<Record<string, PrOption[]>>({})
  const [loadingByNf, setLoadingByNf] = useState<Record<string, boolean>>({})
  const [hasFetchedByNf, setHasFetchedByNf] = useState<Record<string, boolean>>({})
  const [enabledNf, setEnabledNf] = useState<Record<string, boolean>>({})
  const [selectedPrByNf, setSelectedPrByNf] = useState<Record<string, string>>({})
  const [selectedTestcases, setSelectedTestcases] = useState<string[]>([])
  const [pendingTasks, setPendingTasks] = useState<TaskSimple[]>([])
  const [ongoingTasks, setOngoingTasks] = useState<TaskSimple[]>([])

  const api = useMemo(() => new DefaultApi(new Configuration({
    basePath: apiBasePath,
    accessToken: () => localStorage.getItem('token') || '',
  })), [])

  const allSelected = testcases.length > 0 && selectedTestcases.length === testcases.length

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

  function resetFormState() {
    setPrsByNf({})
    setLoadingByNf({})
    setHasFetchedByNf({})
    setEnabledNf({})
    setSelectedPrByNf({})
    setSelectedTestcases([])
  }

  async function refreshTaskQueues() {
    setIsLoadingTasks(true)
    try {
      const response = await api.getTasks({
        headers: getUserHeader(),
      })
      setPendingTasks(response.data.pendingTask || [])
      setOngoingTasks(response.data.ongoingTask || [])
    } catch (error: unknown) {
      addError(extractErrorMessage(error, 'Failed to load tasks'))
    } finally {
      setIsLoadingTasks(false)
    }
  }

  useEffect(() => {
    refreshTaskQueues().catch(() => {
      addError('Failed to load tasks')
    })
  }, [])

  useEffect(() => {
    if (!isFormOpen || hasTestcasesLoaded) {
      return
    }

    refreshTestcases().catch((error: unknown) => {
      const message =
        error instanceof Error
          ? error.message
          : 'Failed to load testcases'
      addError(message)
    })
  }, [isFormOpen, hasTestcasesLoaded, refreshTestcases, addError])

  async function handleToggleNewTest() {
    const nextOpen = !isFormOpen
    setIsFormOpen(nextOpen)

    if (!nextOpen) {
      resetFormState()
      return
    }
  }

  async function loadPrsForNf(apiName: string) {
    if (!isFormOpen || hasFetchedByNf[apiName] || loadingByNf[apiName]) {
      return
    }

    setLoadingByNf((prev) => ({ ...prev, [apiName]: true }))
    try {
      const response = await api.getGithubPRs(apiName as GetGithubPRsNfEnum, {
        headers: getUserHeader(),
      })

      setPrsByNf((prev) => ({
        ...prev,
        [apiName]: (response.data.prs || []).map((item) => ({
          number: item.number,
          title: item.title,
        })),
      }))
      setHasFetchedByNf((prev) => ({ ...prev, [apiName]: true }))
      addSuccess(`${apiName.toUpperCase()} PR list loaded`)
    } catch (error: unknown) {
      const message =
        typeof error === 'object' &&
        error !== null &&
        'response' in error &&
        typeof (error as { response?: { data?: { message?: string } } }).response?.data?.message === 'string'
          ? (error as { response?: { data?: { message?: string } } }).response?.data?.message || 'Failed to load PR list'
          : 'Failed to load PR list'
      addError(message)
    } finally {
      setLoadingByNf((prev) => ({ ...prev, [apiName]: false }))
    }
  }

  async function handleSubmitTask() {
    if (selectedTestcases.length === 0) {
      addError('Please select at least one testcase')
      return
    }

    const enabledNfNames = NF_ORDER
      .map((nf) => nf.apiName)
      .filter((apiName) => Boolean(enabledNf[apiName]))

    if (enabledNfNames.length === 0) {
      addError('Please enable at least one NF')
      return
    }

    const missingPrNf = enabledNfNames.filter((apiName) => !selectedPrByNf[apiName])
    if (missingPrNf.length > 0) {
      addError(`Please select PR for: ${missingPrNf.join(', ')}`)
      return
    }

    const payload: RequestSubmitTask = {
      tests: selectedTestcases,
      nfPrList: enabledNfNames.map((apiName) => ({
        nfName: apiName,
        pr: Number(selectedPrByNf[apiName]),
      })),
    }

    setIsSubmittingTask(true)
    try {
      const response = await api.submitTask(payload, {
        headers: getUserHeader(),
      })
      addSuccess(response.data.message || 'Task submitted successfully')
      setIsFormOpen(false)
      resetFormState()
      await refreshTaskQueues()
    } catch (error: unknown) {
      addError(extractErrorMessage(error, 'Failed to submit task'))
    } finally {
      setIsSubmittingTask(false)
    }
  }

  function updateNfToggle(apiName: string, checked: boolean) {
    setEnabledNf((prev) => ({ ...prev, [apiName]: checked }))

    if (checked) {
      loadPrsForNf(apiName).catch(() => {
        // Error notification is handled in loadPrsForNf
      })
    }

    if (!checked) {
      setSelectedPrByNf((prev) => ({ ...prev, [apiName]: '' }))
    }
  }

  function updateSelectedPr(apiName: string, value: string) {
    setSelectedPrByNf((prev) => ({ ...prev, [apiName]: value }))
  }

  function toggleAllTestcases(checked: boolean) {
    if (checked) {
      setSelectedTestcases(testcases.map((item) => item.name))
      return
    }

    setSelectedTestcases([])
  }

  function toggleSingleTestcase(name: string, checked: boolean) {
    setSelectedTestcases((prev) => {
      if (checked) {
        if (prev.includes(name)) {
          return prev
        }
        return [...prev, name]
      }

      return prev.filter((item) => item !== name)
    })
  }

  return (
    <section className={styles.page}>
      <NotificationContainer
        errors={errors}
        successes={successes}
        onClose={removeNotification}
      />

      <header className={styles.header}>
        <h2>Test</h2>
        <button
          type="button"
          className={styles.newTestButton}
          onClick={handleToggleNewTest}
        >
          {isFormOpen ? 'Close New Test' : 'New Test'}
        </button>
      </header>

      <section className={`${styles.formPanel} ${isFormOpen ? styles.formPanelOpen : ''}`} aria-hidden={!isFormOpen}>
        <div className={styles.formInner}>
          <div className={styles.formGrid}>
            {NF_ORDER.map((nf) => (
              <NfPrSelector
                key={nf.apiName}
                label={nf.label}
                checked={Boolean(enabledNf[nf.apiName])}
                options={prsByNf[nf.apiName] || []}
                selectedPr={selectedPrByNf[nf.apiName] || ''}
                disabled={Boolean(loadingByNf[nf.apiName])}
                onToggle={(checked) => updateNfToggle(nf.apiName, checked)}
                onSelectPr={(value) => updateSelectedPr(nf.apiName, value)}
              />
            ))}

            <section className={styles.testcasePicker}>
              <div className={styles.testcaseHeader}>
                <h3>Testcases</h3>
                <p>Multi-select with quick All option</p>
              </div>

              <div className={styles.testcaseOptions}>
                <label className={`${styles.testcaseOption} ${styles.allOption}`}>
                  <input
                    type="checkbox"
                    checked={allSelected}
                    onChange={(event) => toggleAllTestcases(event.target.checked)}
                    disabled={testcases.length === 0}
                  />
                  <span>All</span>
                </label>

                {testcases.map((item) => {
                  const checked = selectedTestcases.includes(item.name)
                  return (
                    <label key={item.name} className={styles.testcaseOption}>
                      <input
                        type="checkbox"
                        checked={checked}
                        onChange={(event) => toggleSingleTestcase(item.name, event.target.checked)}
                      />
                      <span>{item.name}</span>
                    </label>
                  )
                })}

                {testcases.length === 0 && (
                  <p className={styles.noTestcases}>No testcase options available.</p>
                )}
              </div>
            </section>

            <div className={styles.submitRow}>
              <button
                type="button"
                className={styles.submitButton}
                onClick={handleSubmitTask}
                disabled={isSubmittingTask}
              >
                {isSubmittingTask ? 'Submitting...' : 'Submit Task'}
              </button>
            </div>
          </div>
        </div>
      </section>

      <section className={styles.columns}>
        <article className={styles.columnCard}>
          <h3>Pending Queue</h3>
          <div className={styles.queueList}>
            {isLoadingTasks ? (
              <p className={styles.queueHint}>Loading pending tasks...</p>
            ) : pendingTasks.length === 0 ? (
              <p className={styles.queueHint}>No pending tasks</p>
            ) : (
              pendingTasks.map((task) => (
                <TaskCard
                  key={`pending-${task.id}`}
                  id={task.id}
                  username={task.username}
                  createTime={task.createTime}
                  status="pending"
                />
              ))
            )}
          </div>
        </article>
        <article className={styles.columnCard}>
          <h3>Ongoing Queue</h3>
          <div className={styles.queueList}>
            {isLoadingTasks ? (
              <p className={styles.queueHint}>Loading ongoing tasks...</p>
            ) : ongoingTasks.length === 0 ? (
              <p className={styles.queueHint}>No ongoing tasks</p>
            ) : (
              ongoingTasks.map((task) => (
                <TaskCard
                  key={`ongoing-${task.id}`}
                  id={task.id}
                  username={task.username}
                  createTime={task.createTime}
                  status="ongoing"
                />
              ))
            )}
          </div>
        </article>
        <article className={styles.columnCard}>
          <h3>History Record</h3>
          <p className={styles.queueHint}>Reserved for future history records.</p>
        </article>
      </section>
    </section>
  )
}
