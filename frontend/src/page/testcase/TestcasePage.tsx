import { useEffect, useMemo, useState } from 'react'
import Button from '../../components/button/button'
import Modal from '../../components/modal/modal'
import NotificationContainer from '../../components/notifications/NotificationContainer'
import { useNotifications } from '../../hooks/useNotifications'
import { useTestcaseContext } from '../../context/testcase-context'
import styles from './testcase-page.module.css'

interface FormState {
  name: string
  link: string
}

export default function TestcasePage() {
  const { testcases, isLoading, hasLoaded, refreshTestcases, addTestcase, deleteTestcase } = useTestcaseContext()
  const { errors, successes, addError, addSuccess, removeNotification } = useNotifications()

  const [isAddModalOpen, setIsAddModalOpen] = useState(false)
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false)
  const [formState, setFormState] = useState<FormState>({ name: '', link: '' })
  const [targetDeleteName, setTargetDeleteName] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  useEffect(() => {
    refreshTestcases().catch((error: unknown) => {
      const message = error instanceof Error ? error.message : 'Failed to load testcases'
      addError(message)
    })
  }, [refreshTestcases, addError])

  const nameValue = formState.name.trim()
  const linkValue = formState.link.trim()
  const canAdd = nameValue.length > 0

  const rowCountLabel = useMemo(() => {
    if (isLoading && !hasLoaded) {
      return 'Loading...'
    }

    return `${testcases.length} testcases`
  }, [isLoading, hasLoaded, testcases.length])

  function openAddModal() {
    setFormState({ name: '', link: '' })
    setIsAddModalOpen(true)
  }

  function closeAddModal() {
    setIsAddModalOpen(false)
    setFormState({ name: '', link: '' })
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
        link: linkValue || undefined,
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
              <th>Name</th>
              <th>Link</th>
              <th>Action</th>
            </tr>
          </thead>
          <tbody>
            {testcases.length === 0 ? (
              <tr>
                <td colSpan={3} className={styles.empty}>
                  {isLoading ? 'Loading testcases...' : 'No testcases yet'}
                </td>
              </tr>
            ) : (
              testcases.map((testcase) => (
                <tr key={testcase.name}>
                  <td>{testcase.name}</td>
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
          <label className={styles.label} htmlFor="testcase-link">Testcase Link (optional)</label>
          <input
            id="testcase-link"
            className={styles.input}
            value={formState.link}
            onChange={(event) => setFormState((prev) => ({ ...prev, link: event.target.value }))}
            placeholder="https://example.com"
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
