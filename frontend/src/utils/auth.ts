function decodeJwtPayload(token: string): Record<string, unknown> | null {
  const parts = token.split('.')
  if (parts.length < 2) {
    return null
  }

  try {
    const base64Url = parts[1]
    const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/')
    const padded = base64 + '='.repeat((4 - (base64.length % 4)) % 4)
    const json = atob(padded)
    return JSON.parse(json) as Record<string, unknown>
  } catch {
    return null
  }
}

function readStringClaim(payload: Record<string, unknown> | null, key: string): string {
  const value = payload?.[key]
  return typeof value === 'string' ? value.trim() : ''
}

export function getCurrentUsername(): string {
  const savedUsername = localStorage.getItem('username')?.trim() || ''
  if (savedUsername) {
    return savedUsername
  }

  const token = localStorage.getItem('token') || ''
  const payload = decodeJwtPayload(token)

  return (
    readStringClaim(payload, 'user')
    || readStringClaim(payload, 'username')
    || readStringClaim(payload, 'sub')
  )
}

export function getUserHeader(): Record<string, string> {
  const username = getCurrentUsername()
  return username ? { user: username } : {}
}
