import axios from 'axios'
import { getUserHeader } from '../utils/auth'

export interface Runner {
  name: string
  ip: string
  onGoingTask: number
  status: string
}

interface GetRunnersResponse {
  message?: string
  runners?: Runner[]
}

interface MessageResponse {
  message?: string
}

const apiBasePath = import.meta.env.VITE_API_BASE_URL || `${window.location.protocol}//${window.location.hostname}:8888`

function getAuthHeaders() {
  const token = localStorage.getItem('token') || ''

  return {
    Authorization: token ? `Bearer ${token}` : '',
    ...getUserHeader(),
  }
}

export async function getRunners() {
  const response = await axios.get<GetRunnersResponse>(`${apiBasePath}/api/runner`, {
    headers: getAuthHeaders(),
  })

  return response.data.runners || []
}

export async function deleteRunner(name: string) {
  const response = await axios.delete<MessageResponse>(`${apiBasePath}/api/admin/runner`, {
    params: { name },
    headers: getAuthHeaders(),
  })

  return response.data.message || 'Runner deleted successfully'
}
