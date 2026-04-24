import { useMemo, useState } from 'react'
import { Configuration, DefaultApi } from '../../api'
import { getUserHeader } from '../../utils/auth'
import NotificationContainer from '../../components/notifications/NotificationContainer'
import { useNotifications } from '../../hooks/useNotifications'
import NfPrSelector, { type PrOption } from '../../components/test/NfPrSelector'
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
  { label: 'UPF', apiName: 'go-upf' },
]

const apiBasePath = import.meta.env.VITE_API_BASE_URL || `${window.location.protocol}//${window.location.hostname}:8888`

export default function TestPage() {
  const { errors, successes, addError, addSuccess, removeNotification } = useNotifications()

  const [isFormOpen, setIsFormOpen] = useState(false)
  const [isLoadingPrs, setIsLoadingPrs] = useState(false)
  const [prsByNf, setPrsByNf] = useState<Record<string, PrOption[]>>({})
  const [enabledNf, setEnabledNf] = useState<Record<string, boolean>>({})
  const [selectedPrByNf, setSelectedPrByNf] = useState<Record<string, string>>({})

  const api = useMemo(() => new DefaultApi(new Configuration({
    basePath: apiBasePath,
    accessToken: () => localStorage.getItem('token') || '',
  })), [])

  async function handleToggleNewTest() {
    const nextOpen = !isFormOpen
    setIsFormOpen(nextOpen)

    if (!nextOpen) {
      return
    }

    setIsLoadingPrs(true)
    try {
      const response = await api.getGithubPRs({
        headers: getUserHeader(),
      })

      const byNf: Record<string, PrOption[]> = {}
      for (const nf of response.data.nfs || []) {
        if (!nf.name) {
          continue
        }
        byNf[nf.name.toLowerCase()] = (nf.prs || []).map((item) => ({
          number: item.number,
          title: item.title,
        }))
      }

      setPrsByNf(byNf)
      addSuccess('PR list loaded')
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
      setIsLoadingPrs(false)
    }
  }

  function updateNfToggle(apiName: string, checked: boolean) {
    setEnabledNf((prev) => ({ ...prev, [apiName]: checked }))
    if (!checked) {
      setSelectedPrByNf((prev) => ({ ...prev, [apiName]: '' }))
    }
  }

  function updateSelectedPr(apiName: string, value: string) {
    setSelectedPrByNf((prev) => ({ ...prev, [apiName]: value }))
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
          disabled={isLoadingPrs}
        >
          {isLoadingPrs ? 'Loading PRs...' : isFormOpen ? 'Close New Test' : 'New Test'}
        </button>
      </header>

      <section className={`${styles.formPanel} ${isFormOpen ? styles.formPanelOpen : ''}`} aria-hidden={!isFormOpen}>
        <div className={styles.formInner}>
          {isLoadingPrs ? (
            <div className={styles.loaderWrap}>
              <div className={styles.loaderHead}>
                <span className={styles.spinner} aria-hidden="true" />
                <p>Loading PR list from GitHub...</p>
              </div>
              <div className={styles.loaderRows}>
                <div className={styles.loaderRow} />
                <div className={styles.loaderRow} />
                <div className={styles.loaderRow} />
                <div className={styles.loaderRow} />
              </div>
            </div>
          ) : (
            <div className={styles.formGrid}>
              {NF_ORDER.map((nf) => (
                <NfPrSelector
                  key={nf.apiName}
                  label={nf.label}
                  checked={Boolean(enabledNf[nf.apiName])}
                  options={prsByNf[nf.apiName] || []}
                  selectedPr={selectedPrByNf[nf.apiName] || ''}
                  disabled={isLoadingPrs}
                  onToggle={(checked) => updateNfToggle(nf.apiName, checked)}
                  onSelectPr={(value) => updateSelectedPr(nf.apiName, value)}
                />
              ))}
            </div>
          )}
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
