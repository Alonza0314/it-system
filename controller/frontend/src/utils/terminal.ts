import { Configuration, DefaultApi } from '../api'
import { apiBasePath } from './apiBase'

// Protocol of the runner terminal WebSocket; see design/runner-terminal.md.
export const TERMINAL_SUBPROTOCOL = 'it-terminal.v1'
export const TERMINAL_TICKET_PREFIX = 'ticket.'

export const terminalApi = new DefaultApi(new Configuration({
  basePath: apiBasePath,
  accessToken: () => localStorage.getItem('token') || '',
}))

export function terminalWebSocketUrl(): string {
  const url = new URL('/api/terminal/ws', apiBasePath)
  url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
  return url.toString()
}

export type TerminalServerMessage =
  | { type: 'ready' }
  | { type: 'exit'; code?: number }
  | { type: 'error'; message?: string }

export function parseTerminalMessage(data: string): TerminalServerMessage | null {
  try {
    const msg = JSON.parse(data) as { type?: unknown }
    if (msg.type === 'ready' || msg.type === 'exit' || msg.type === 'error') {
      return msg as TerminalServerMessage
    }
  } catch {
    // not ours
  }
  return null
}

// Text for a close that came without an error message from the controller.
export function describeTerminalClose(code: number, reason: string): string {
  switch (code) {
    case 1000:
      return reason || 'Session closed'
    case 1001:
      return 'Controller is shutting down'
    case 4403:
      return reason || 'Not allowed'
    case 4408:
      return reason === 'idle timeout' ? 'Closed after being idle' : 'Session time limit reached'
    case 4409:
      return 'Runner disconnected'
    case 4429:
      return 'Session limit reached'
    case 4504:
      return 'Runner did not respond'
    default:
      return reason || `Connection closed (${code})`
  }
}

export function apiErrorMessage(error: unknown, fallback: string): string {
  const message = (error as { response?: { data?: { message?: unknown } } })?.response?.data?.message
  return typeof message === 'string' && message ? message : fallback
}
