import { useEffect, useMemo, useState } from 'react'
import Button from '../../components/button/button'
import Modal from '../../components/modal/modal'
import NotificationContainer from '../../components/notifications/NotificationContainer'
import { useNotifications } from '../../hooks/useNotifications'
import { useTestcaseContext } from '../../context/testcase-context'
import styles from './testcase-page.module.css'

interface FormState {
  name: string
  script: string
  link: string
  label: string
}

const emptyFormState: FormState = { name: '', script: '', link: '', label: '' }

export default function TestcasePage() {
  const { testcases, isLoading, hasLoaded, refreshTestcases, addTestcase, deleteTestcase } = useTestcaseContext()
  const { errors, successes, addError, addSuccess, removeNotification } = useNotifications()

  const [isAddModalOpen, setIsAddModalOpen] = useState(false)
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false)
  const [formState, setFormState] = useState<FormState>(emptyFormState)
  const [targetDeleteName, setTargetDeleteName] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  useEffect(() => {
    refreshTestcases().catch((error: unknown) => {
      const message = error instanceof Error ? error.message : 'Failed to load testcases'
      addError(message)
    })
  }, [refreshTestcases, addError])

  const nameValue = formState.name.trim()
  const scriptValue = formState.script.trim()
  const linkValue = formState.link.trim()
  const labelValue = formState.label.trim()
  const canAdd = nameValue.length > 0

  const sortedTestcases = useMemo(() => {
    return [...testcases].sort((a, b) => {
      const labelCompare = (a.label || '').localeCompare(b.label || '')
      if (labelCompare !== 0) {
        return labelCompare
      }

      return (a.id ?? 0) - (b.id ?? 0)
    })
  }, [testcases])

  const rowCountLabel = useMemo(() => {
    if (isLoading && !hasLoaded) {
      return 'Loading...'
    }

    return `${testcases.length} testcases`
  }, [isLoading, hasLoaded, testcases.length])

  function openAddModal() {
    setFormState(emptyFormState)
    setIsAddModalOpen(true)
  }

  function closeAddModal() {
    setIsAddModalOpen(false)
    setFormState(emptyFormState)
  }

  function openDeleteModal(name: string) {
    setTargetDeleteName(name)
    setIsDeleteModalOpen(true)
  }

  function closeDeleteModal() {
    setTargetDeleteName('')
    setIsDeleteModalOpen(false)
  }

  async function handleAddTestcase() {
    if (!canAdd) {
      addError('Testcase name is required')
      return
    }

    setIsSubmitting(true)
    try {
      const message = await addTestcase({
        name: nameValue,
        script: scriptValue || undefined,
        link: linkValue || undefined,
        label: labelValue || undefined,
      })
      addSuccess(message)
      closeAddModal()
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : 'Failed to add testcase'
      addError(message)
    } finally {
      setIsSubmitting(false)
    }
  }

  async function handleDeleteTestcase() {
    if (!targetDeleteName) {
      return
    }

    setIsSubmitting(true)
    try {
      const message = await deleteTestcase(targetDeleteName)
      addSuccess(message)
      closeDeleteModal()
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : 'Failed to delete testcase'
      addError(message)
    } finally {
      setIsSubmitting(false)
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
          <h2>Testcase</h2>
          <p>{rowCountLabel}</p>
        </div>
        <Button onClick={openAddModal}>Add Testcase</Button>
      </header>

      <div className={styles.tableWrap}>
        <table className={styles.table}>
          <thead>
            <tr>
              <th>ID</th>
              <th>Name</th>
              <th>Script</th>
              <th>Link</th>
              <th>Action</th>
            </tr>
          </thead>
          <tbody>
            {sortedTestcases.length === 0 ? (
              <tr>
                <td colSpan={5} className={styles.empty}>
                  {isLoading ? 'Loading testcases...' : 'No testcases yet'}
                </td>
              </tr>
            ) : (
              sortedTestcases.map((testcase) => (
                <tr key={testcase.name}>
                  <td>{testcase.id ?? <span className={styles.muted}>-</span>}</td>
                  <td>{testcase.name}</td>
                  <td>{testcase.script || <span className={styles.muted}>-</span>}</td>
                  <td>
                    {testcase.link ? (
                      <a href={testcase.link} target="_blank" rel="noreferrer" className={styles.link}>
                        {testcase.link}
                      </a>
                    ) : (
                      <span className={styles.muted}>-</span>
                    )}
                  </td>
                  <td>
                    <Button variant="secondary" onClick={() => openDeleteModal(testcase.name)}>
                      Delete
                    </Button>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      <Modal
        isOpen={isAddModalOpen}
        onClose={closeAddModal}
        title="Confirm Add Testcase"
        onSubmit={handleAddTestcase}
        submitText={isSubmitting ? 'Adding...' : 'Confirm Add'}
        submitDisabled={isSubmitting || !canAdd}
      >
        <p className={styles.modalQuestion}>Are you sure you want to add this testcase?</p>
        <div className={styles.formGroup}>
          <label className={styles.label} htmlFor="testcase-name">Testcase Name (required)</label>
          <input
            id="testcase-name"
            className={styles.input}
            value={formState.name}
            onChange={(event) => setFormState((prev) => ({ ...prev, name: event.target.value }))}
            placeholder="e.g. test-1"
            required
          />
        </div>
        <div className={styles.formGroup}>
          <label className={styles.label} htmlFor="testcase-script">Testcase Script (optional)</label>
          <input
            id="testcase-script"
            className={styles.input}
            value={formState.script}
            onChange={(event) => setFormState((prev) => ({ ...prev, script: event.target.value }))}
            placeholder="e.g. test.sh"
          />
        </div>
        <div className={styles.formGroup}>
          <label className={styles.label} htmlFor="testcase-link">Testcase Link (optional)</label>
          <input
            id="testcase-link"
            className={styles.input}
            value={formState.link}
            onChange={(event) => setFormState((prev) => ({ ...prev, link: event.target.value }))}
            placeholder="https://example.com"
          />
        </div>
        <div className={styles.formGroup}>
          <label className={styles.label} htmlFor="testcase-label">Testcase Label (optional)</label>
          <input
            id="testcase-label"
            className={styles.input}
            value={formState.label}
            onChange={(event) => setFormState((prev) => ({ ...prev, label: event.target.value }))}
            placeholder="e.g. it"
          />
        </div>
      </Modal>

      <Modal
        isOpen={isDeleteModalOpen}
        onClose={closeDeleteModal}
        title="Confirm Delete Testcase"
        onSubmit={handleDeleteTestcase}
        submitText={isSubmitting ? 'Deleting...' : 'Confirm Delete'}
        submitDisabled={isSubmitting}
      >
        <p className={styles.modalQuestion}>
          Are you sure you want to delete testcase "{targetDeleteName}"?
        </p>
      </Modal>
    </section>
  )
}
