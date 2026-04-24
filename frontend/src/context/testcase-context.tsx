import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from 'react'
import { Configuration, DefaultApi, type Testcase } from '../api'
import { getUserHeader } from '../utils/auth'

interface AddTestcasePayload {
  name: string
  link?: string
}

interface TestcaseContextValue {
  testcases: Testcase[]
  isLoading: boolean
  hasLoaded: boolean
  refreshTestcases: () => Promise<void>
  addTestcase: (payload: AddTestcasePayload) => Promise<string>
  deleteTestcase: (name: string) => Promise<string>
}

const TestcaseContext = createContext<TestcaseContextValue | undefined>(undefined)

const apiBasePath = import.meta.env.VITE_API_BASE_URL || `${window.location.protocol}//${window.location.hostname}:8888`

function getErrorMessage(error: unknown, fallback: string) {
  if (
    typeof error === 'object' &&
    error !== null &&
    'response' in error &&
    typeof (error as { response?: { data?: { message?: string } } }).response?.data?.message === 'string'
  ) {
    return (error as { response?: { data?: { message?: string } } }).response?.data?.message || fallback
  }

  return fallback
}

export function TestcaseProvider({ children }: { children: ReactNode }) {
  const [testcases, setTestcases] = useState<Testcase[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [hasLoaded, setHasLoaded] = useState(false)

  const api = useMemo(() => {
    return new DefaultApi(
      new Configuration({
        basePath: apiBasePath,
        accessToken: () => localStorage.getItem('token') || '',
      }),
    )
  }, [])

  const refreshTestcases = useCallback(async () => {
    setIsLoading(true)

    try {
      const response = await api.getTestcases({
        headers: getUserHeader(),
      })
      setTestcases(response.data.testcases || [])
      setHasLoaded(true)
    } catch (error: unknown) {
      throw new Error(getErrorMessage(error, 'Failed to load testcases'))
    } finally {
      setIsLoading(false)
    }
  }, [api])

  const addTestcase = useCallback(async (payload: AddTestcasePayload) => {
    setIsLoading(true)

    try {
      const response = await api.addTestcases({
        testcases: [
          {
            name: payload.name,
            link: payload.link,
          },
        ],
      }, {
        headers: getUserHeader(),
      })

      const refreshResponse = await api.getTestcases({
        headers: getUserHeader(),
      })
      setTestcases(refreshResponse.data.testcases || [])
      setHasLoaded(true)

      return response.data.message || 'Testcase added successfully'
    } catch (error: unknown) {
      throw new Error(getErrorMessage(error, 'Failed to add testcase'))
    } finally {
      setIsLoading(false)
    }
  }, [api])

  const deleteTestcase = useCallback(async (name: string) => {
    setIsLoading(true)

    try {
      const response = await api.deleteTestcases({
        testcases: [
          {
            name,
          },
        ],
      }, {
        headers: getUserHeader(),
      })

      const refreshResponse = await api.getTestcases({
        headers: getUserHeader(),
      })
      setTestcases(refreshResponse.data.testcases || [])
      setHasLoaded(true)

      return response.data.message || 'Testcase deleted successfully'
    } catch (error: unknown) {
      throw new Error(getErrorMessage(error, 'Failed to delete testcase'))
    } finally {
      setIsLoading(false)
    }
  }, [api])

  return (
    <TestcaseContext.Provider value={{
      testcases,
      isLoading,
      hasLoaded,
      refreshTestcases,
      addTestcase,
      deleteTestcase,
    }}
    >
      {children}
    </TestcaseContext.Provider>
  )
}

export function useTestcaseContext() {
  const context = useContext(TestcaseContext)

  if (!context) {
    throw new Error('useTestcaseContext must be used within a TestcaseProvider')
  }

  return context
}
