import { useState, useCallback } from 'react'

interface NotificationItem {
  id: string
  message: string
  timestamp: number
  type: 'error' | 'success'
}

export function useNotifications() {
  const [notifications, setNotifications] = useState<NotificationItem[]>([])

  const addNotification = useCallback((message: string, type: 'error' | 'success') => {
    const id = Date.now().toString()
    setNotifications(prev => [...prev, {
      id,
      message,
      timestamp: Date.now(),
      type
    }])
    return id
  }, [])

  const removeNotification = useCallback((id: string) => {
    setNotifications(prev => prev.filter(notification => notification.id !== id))
  }, [])

  const addError = useCallback((message: string) => {
    return addNotification(message, 'error')
  }, [addNotification])

  const addSuccess = useCallback((message: string) => {
    return addNotification(message, 'success')
  }, [addNotification])

  const errors = notifications.filter(n => n.type === 'error')
  const successes = notifications.filter(n => n.type === 'success')

  return {
    errors,
    successes,
    addError,
    addSuccess,
    removeNotification
  }
}
