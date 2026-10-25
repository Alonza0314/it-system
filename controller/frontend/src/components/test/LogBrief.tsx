import type { LogBriefEntry } from '../../api'
import styles from './log-brief.module.css'

interface LogBriefProps {
  entries: LogBriefEntry[]
  // Show this line of the full log
  onJumpToLine: (line: number) => void
}

function getBadge(entry: LogBriefEntry) {
  if (entry.category === 'test') {
    if (entry.message.startsWith('--- PASS')) {
      return { label: 'PASS', className: styles.badgePass }
    }
    return { label: entry.message.startsWith('--- FAIL') ? 'FAIL' : 'ASSERT', className: styles.badgeFail }
  }
  if (entry.category === 'error') {
    return { label: 'ERROR', className: styles.badgeError }
  }
  return { label: 'EVENT', className: styles.badgeEvent }
}

// "2026-10-25T06:27:39.512046137Z" -> "06:27:39.512"
function formatTime(time?: string) {
  return time ? time.slice(11, 23) : ''
}

export function hasLogProblem(entries: LogBriefEntry[]) {
  return entries.some((entry) => entry.category === 'error' || (entry.category === 'test' && !entry.message.startsWith('--- PASS')))
}

export default function LogBrief({ entries, onJumpToLine }: LogBriefProps) {
  if (entries.length === 0) {
    return <p className={styles.empty}>No test result, error or key event found in this log.</p>
  }

  const sections = [
    { title: 'Test result', items: entries.filter((entry) => entry.category === 'test') },
    { title: 'Timeline (UTC)', items: entries.filter((entry) => entry.category !== 'test' && entry.time) },
    { title: 'Other errors', items: entries.filter((entry) => entry.category !== 'test' && !entry.time) },
  ].filter((section) => section.items.length > 0)

  return (
    <div className={styles.brief}>
      {sections.map((section) => (
        <section key={section.title} className={styles.section}>
          <h4>{section.title}</h4>
          <ul className={styles.list}>
            {section.items.map((entry) => {
              const badge = getBadge(entry)
              return (
                <li key={`${entry.line}-${entry.message}`}>
                  <button type="button" className={styles.entry} onClick={() => onJumpToLine(entry.line)}>
                    <span className={`${styles.badge} ${badge.className}`}>{badge.label}</span>
                    <span className={styles.time}>{formatTime(entry.time)}</span>
                    <span className={entry.source ? styles.source : undefined}>{entry.source}</span>
                    <span className={styles.message}>
                      {entry.message}
                      {entry.count > 1 && <span className={styles.count}> ×{entry.count}</span>}
                      {entry.items && entry.items.length > 0 && (
                        <span className={styles.items}>{entry.items.join(', ')}</span>
                      )}
                    </span>
                    <span className={styles.line}>L{entry.line}</span>
                  </button>
                </li>
              )
            })}
          </ul>
        </section>
      ))}
    </div>
  )
}
