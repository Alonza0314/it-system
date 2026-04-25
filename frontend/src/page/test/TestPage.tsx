import { useEffect, useMemo, useState } from 'react'
import { Configuration, DefaultApi, type GetGithubPRsNfEnum } from '../../api'
import { getUserHeader } from '../../utils/auth'
import NotificationContainer from '../../components/notifications/NotificationContainer'
import { useNotifications } from '../../hooks/useNotifications'
import NfPrSelector, { type PrOption } from '../../components/test/NfPrSelector'
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

  const [isFormOpen, setIsFormOpen] = useState(false)
  const [prsByNf, setPrsByNf] = useState<Record<string, PrOption[]>>({})
  const [loadingByNf, setLoadingByNf] = useState<Record<string, boolean>>({})
  const [hasFetchedByNf, setHasFetchedByNf] = useState<Record<string, boolean>>({})
  const [enabledNf, setEnabledNf] = useState<Record<string, boolean>>({})
  const [selectedPrByNf, setSelectedPrByNf] = useState<Record<string, string>>({})
  const [selectedTestcases, setSelectedTestcases] = useState<string[]>([])

  const api = useMemo(() => new DefaultApi(new Configuration({
    basePath: apiBasePath,
    accessToken: () => localStorage.getItem('token') || '',
  })), [])

  const allSelected = testcases.length > 0 && selectedTestcases.length === testcases.length

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
      setPrsByNf({})
      setLoadingByNf({})
      setHasFetchedByNf({})
      setEnabledNf({})
      setSelectedPrByNf({})
      setSelectedTestcases([])
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
          </div>
        </div>
      </section>

      <section className={styles.columns}>
        <article className={styles.columnCard}>
          <h3>Column 1</h3>
        </article>
        <article className={styles.columnCard}>
          <h3>Column 2</h3>
        </article>
        <article className={styles.columnCard}>
          <h3>Column 3</h3>
        </article>
      </section>
    </section>
  )
}
