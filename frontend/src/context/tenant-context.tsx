import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from 'react'
import { Configuration, DefaultApi, type Tenant } from '../api'
import { getUserHeader } from '../utils/auth'

interface AddTenantPayload {
  username: string
  role: string
}

interface DeleteTenantPayload {
  username: string
  role: string
}

interface TenantContextValue {
  tenants: Tenant[]
  isLoading: boolean
  hasLoaded: boolean
  hasNoPermission: boolean
  refreshTenants: () => Promise<void>
  addTenant: (payload: AddTenantPayload) => Promise<string>
  deleteTenant: (payload: DeleteTenantPayload) => Promise<string>
}

const TenantContext = createContext<TenantContextValue | undefined>(undefined)

const apiBasePath = import.meta.env.VITE_API_BASE_URL || `${window.location.protocol}//${window.location.hostname}:8888`

function getErrorResponse(error: unknown) {
  if (
    typeof error === 'object' &&
    error !== null &&
    'response' in error
  ) {
    return (error as { response?: { status?: number; data?: { message?: string } } }).response
  }

  return undefined
}

function getErrorMessage(error: unknown, fallback: string) {
  const response = getErrorResponse(error)
  if (typeof response?.data?.message === 'string') {
    return response.data.message
  }

  return fallback
}

export function TenantProvider({ children }: { children: ReactNode }) {
  const [tenants, setTenants] = useState<Tenant[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [hasLoaded, setHasLoaded] = useState(false)
  const [hasNoPermission, setHasNoPermission] = useState(false)

  const api = useMemo(() => {
    return new DefaultApi(
      new Configuration({
        basePath: apiBasePath,
        accessToken: () => localStorage.getItem('token') || '',
      }),
    )
  }, [])

  const refreshTenants = useCallback(async () => {
    setIsLoading(true)

    try {
      const response = await api.getTenants({
        headers: getUserHeader(),
      })
      setTenants(response.data.tenants || [])
      setHasNoPermission(false)
      setHasLoaded(true)
    } catch (error: unknown) {
      const status = getErrorResponse(error)?.status
      if (status === 403) {
        setTenants([])
        setHasNoPermission(true)
        setHasLoaded(true)
        return
      }

      throw new Error(getErrorMessage(error, 'Failed to load tenants'))
    } finally {
      setIsLoading(false)
    }
  }, [api])

  const addTenant = useCallback(async (payload: AddTenantPayload) => {
    setIsLoading(true)

    try {
      const response = await api.addTenants({
        tenants: [
          {
            username: payload.username,
            role: payload.role,
          },
        ],
      }, {
        headers: getUserHeader(),
      })

      const refreshResponse = await api.getTenants({
        headers: getUserHeader(),
      })
      setTenants(refreshResponse.data.tenants || [])
      setHasNoPermission(false)
      setHasLoaded(true)

      return response.data.message || 'Tenants added successfully'
    } catch (error: unknown) {
      throw new Error(getErrorMessage(error, 'Failed to add tenant'))
    } finally {
      setIsLoading(false)
    }
  }, [api])

  const deleteTenant = useCallback(async (payload: DeleteTenantPayload) => {
    setIsLoading(true)

    try {
      const response = await api.deleteTenants({
        tenants: [
          {
            username: payload.username,
            role: payload.role,
          },
        ],
      }, {
        headers: getUserHeader(),
      })

      const refreshResponse = await api.getTenants({
        headers: getUserHeader(),
      })
      setTenants(refreshResponse.data.tenants || [])
      setHasNoPermission(false)
      setHasLoaded(true)

      return response.data.message || 'Tenants deleted successfully'
    } catch (error: unknown) {
      throw new Error(getErrorMessage(error, 'Failed to delete tenant'))
    } finally {
      setIsLoading(false)
    }
  }, [api])

  return (
    <TenantContext.Provider value={{
      tenants,
      isLoading,
      hasLoaded,
      hasNoPermission,
      refreshTenants,
      addTenant,
      deleteTenant,
    }}
    >
      {children}
    </TenantContext.Provider>
  )
}

export function useTenantContext() {
  const context = useContext(TenantContext)

  if (!context) {
    throw new Error('useTenantContext must be used within a TenantProvider')
  }

  return context
}
