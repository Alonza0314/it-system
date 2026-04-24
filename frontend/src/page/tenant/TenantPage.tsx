import { useEffect, useMemo, useState } from 'react'
import Button from '../../components/button/button'
import Modal from '../../components/modal/modal'
import NotificationContainer from '../../components/notifications/NotificationContainer'
import { useNotifications } from '../../hooks/useNotifications'
import { useTenantContext } from '../../context/tenant-context'
import styles from './tenant-page.module.css'

interface FormState {
  username: string
  role: string
}

export default function TenantPage() {
  const {
    tenants,
    isLoading,
    hasLoaded,
    hasNoPermission,
    refreshTenants,
    addTenant,
    deleteTenant,
  } = useTenantContext()

  const { errors, successes, addError, addSuccess, removeNotification } = useNotifications()

  const [isAddModalOpen, setIsAddModalOpen] = useState(false)
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false)
  const [formState, setFormState] = useState<FormState>({ username: '', role: 'default' })
  const [targetDelete, setTargetDelete] = useState<FormState>({ username: '', role: '' })
  const [isSubmitting, setIsSubmitting] = useState(false)

  useEffect(() => {
    refreshTenants().catch((error: unknown) => {
      const message = error instanceof Error ? error.message : 'Failed to load tenants'
      addError(message)
    })
  }, [refreshTenants, addError])

  const usernameValue = formState.username.trim()
  const roleValue = formState.role.trim()
  const canAdd = usernameValue.length > 0 && roleValue.length > 0

  const countLabel = useMemo(() => {
    if (isLoading && !hasLoaded) {
      return 'Loading...'
    }

    if (hasNoPermission) {
      return 'No Access'
    }

    return `${tenants.length} tenants`
  }, [isLoading, hasLoaded, hasNoPermission, tenants.length])

  function openAddModal() {
    setFormState({ username: '', role: 'default' })
    setIsAddModalOpen(true)
  }

  function closeAddModal() {
    setIsAddModalOpen(false)
    setFormState({ username: '', role: 'default' })
  }

  function openDeleteModal(username: string, role: string) {
    setTargetDelete({ username, role })
    setIsDeleteModalOpen(true)
  }

  function closeDeleteModal() {
    setTargetDelete({ username: '', role: '' })
    setIsDeleteModalOpen(false)
  }

  async function handleAddTenant() {
    if (!canAdd) {
      addError('Username and role are required')
      return
    }

    setIsSubmitting(true)
    try {
      const message = await addTenant({
        username: usernameValue,
        role: roleValue,
      })
      addSuccess(message)
      closeAddModal()
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : 'Failed to add tenant'
      addError(message)
    } finally {
      setIsSubmitting(false)
    }
  }

  async function handleDeleteTenant() {
    if (!targetDelete.username || !targetDelete.role) {
      return
    }

    setIsSubmitting(true)
    try {
      const message = await deleteTenant(targetDelete)
      addSuccess(message)
      closeDeleteModal()
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : 'Failed to delete tenant'
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
          <h2>Tenant</h2>
          <p>{countLabel}</p>
        </div>
        {!hasNoPermission && (
          <Button onClick={openAddModal}>Add Tenant</Button>
        )}
      </header>

      {hasNoPermission ? (
        <section className={styles.permissionCard}>
          <h3>No Permission</h3>
          <p>You do not have permission to view or manage tenants.</p>
        </section>
      ) : (
        <div className={styles.tableWrap}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th>Username</th>
                <th>Role</th>
                <th>Action</th>
              </tr>
            </thead>
            <tbody>
              {tenants.length === 0 ? (
                <tr>
                  <td colSpan={3} className={styles.empty}>
                    {isLoading ? 'Loading tenants...' : 'No tenants yet'}
                  </td>
                </tr>
              ) : (
                tenants.map((tenant) => (
                  <tr key={tenant.username}>
                    <td>{tenant.username}</td>
                    <td>{tenant.role}</td>
                    <td>
                      <Button variant="secondary" onClick={() => openDeleteModal(tenant.username, tenant.role)}>
                        Delete
                      </Button>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      )}

      <Modal
        isOpen={isAddModalOpen}
        onClose={closeAddModal}
        title="Confirm Add Tenant"
        onSubmit={handleAddTenant}
        submitText={isSubmitting ? 'Adding...' : 'Confirm Add'}
        submitDisabled={isSubmitting || !canAdd}
      >
        <p className={styles.modalQuestion}>Are you sure you want to add this tenant?</p>
        <div className={styles.formGroup}>
          <label className={styles.label} htmlFor="tenant-username">Username (required)</label>
          <input
            id="tenant-username"
            className={styles.input}
            value={formState.username}
            onChange={(event) => setFormState((prev) => ({ ...prev, username: event.target.value }))}
            placeholder="e.g. alonza"
            required
          />
        </div>
        <div className={styles.formGroup}>
          <label className={styles.label} htmlFor="tenant-role">Role (required)</label>
          <select
            id="tenant-role"
            className={styles.input}
            value={formState.role}
            onChange={(event) => setFormState((prev) => ({ ...prev, role: event.target.value }))}
          >
            <option value="default">default</option>
            <option value="admin">admin</option>
          </select>
        </div>
      </Modal>

      <Modal
        isOpen={isDeleteModalOpen}
        onClose={closeDeleteModal}
        title="Confirm Delete Tenant"
        onSubmit={handleDeleteTenant}
        submitText={isSubmitting ? 'Deleting...' : 'Confirm Delete'}
        submitDisabled={isSubmitting}
      >
        <p className={styles.modalQuestion}>
          Are you sure you want to delete tenant "{targetDelete.username}"?
        </p>
      </Modal>
    </section>
  )
}
