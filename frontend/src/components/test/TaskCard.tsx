import { useNavigate } from 'react-router-dom'
import styles from './task-card.module.css'

interface TaskCardProps {
  id: number
  username: string
  createTime: number
  status: 'pending' | 'ongoing'
}

function formatCreateTime(unixTime: number) {
  if (!unixTime) {
    return '-'
  }

  return new Date(unixTime * 1000).toLocaleString()
}

export default function TaskCard({ id, username, createTime, status }: TaskCardProps) {
  const navigate = useNavigate()

  return (
    <button
      type="button"
      className={styles.card}
      onClick={() => navigate(`/test/task/${id}?from=${status}`)}
    >
      <div className={styles.header}>
        <p className={styles.id}>Task #{id}</p>
        <div className={styles.statusWrap}>
          <span
            className={`${styles.spinner} ${status === 'pending' ? styles.pendingSpinner : styles.ongoingSpinner}`}
            aria-hidden="true"
          />
          <span className={styles.statusText}>{status === 'pending' ? 'Pending' : 'Ongoing'}</span>
        </div>
      </div>

      <div className={styles.body}>
        <p><strong>User:</strong> {username || '-'}</p>
        <p><strong>Create Time:</strong> {formatCreateTime(createTime)}</p>
      </div>
    </button>
  )
}
