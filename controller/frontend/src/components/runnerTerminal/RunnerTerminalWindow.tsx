import { useCallback, useEffect, useRef, useState } from 'react'
import type { PointerEvent as ReactPointerEvent } from 'react'
import { createPortal } from 'react-dom'
import { Terminal } from '@xterm/xterm'
import type { ITheme } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import '@fontsource/jetbrains-mono/400.css'
import '@fontsource/jetbrains-mono/700.css'
import {
  TERMINAL_SUBPROTOCOL,
  TERMINAL_TICKET_PREFIX,
  apiErrorMessage,
  describeTerminalClose,
  parseTerminalMessage,
  terminalApi,
  terminalWebSocketUrl,
} from '../../utils/terminal'
import styles from './runner-terminal-window.module.css'

type ConnectionState = 'connecting' | 'connected' | 'closed' | 'error'

interface Geometry {
  x: number
  y: number
  width: number
  height: number
}

interface RunnerTerminalWindowProps {
  runner: string
  onGoingTask: number
  // position in the stack of open windows, to cascade new ones
  offset: number
  zIndex: number
  onFocus: () => void
  onClose: () => void
}

const FONT_FAMILY = '"JetBrains Mono", ui-monospace, SFMono-Regular, Menlo, Consolas, "Liberation Mono", monospace'

// macOS dark-mode system colors, on the site's dark gray
const THEME: ITheme = {
  background: '#1c1c1e',
  foreground: '#e5e5ea',
  cursor: '#0a84ff',
  cursorAccent: '#1c1c1e',
  selectionBackground: 'rgba(10, 132, 255, 0.35)',
  black: '#3a3a3c',
  red: '#ff6961',
  green: '#30d158',
  yellow: '#ffd60a',
  blue: '#409cff',
  magenta: '#da8fff',
  cyan: '#70d7ff',
  white: '#d1d1d6',
  brightBlack: '#8e8e93',
  brightRed: '#ff8a80',
  brightGreen: '#5ee07f',
  brightYellow: '#ffe45e',
  brightBlue: '#70b4ff',
  brightMagenta: '#e6b1ff',
  brightCyan: '#9be6ff',
  brightWhite: '#f5f5f7',
}

const EDGE = 8
const MIN_WIDTH = 300
const MIN_HEIGHT = 200

const STATE_CLASS: Record<ConnectionState, string> = {
  connecting: styles.stateConnecting,
  connected: styles.stateConnected,
  closed: styles.stateClosed,
  error: styles.stateError,
}

const STATE_LABEL: Record<ConnectionState, string> = {
  connecting: 'Connecting…',
  connected: 'Connected',
  closed: 'Disconnected',
  error: 'Failed',
}

function viewport() {
  return { width: window.innerWidth, height: window.innerHeight }
}

function initialGeometry(offset: number): Geometry {
  const vp = viewport()
  if (vp.width <= 640) {
    const height = Math.min(420, Math.round(vp.height * 0.6))
    return { x: EDGE, y: Math.max(EDGE, vp.height - height - EDGE), width: vp.width - EDGE * 2, height }
  }

  const width = Math.min(720, vp.width - EDGE * 2)
  const height = Math.min(440, vp.height - EDGE * 2)
  const step = (offset % 5) * 28
  return {
    x: Math.max(EDGE, vp.width - width - 24 - step),
    y: Math.max(EDGE, vp.height - height - 24 - step),
    width,
    height,
  }
}

// Keep the whole window on screen.
function clampGeometry(g: Geometry): Geometry {
  const vp = viewport()
  const width = Math.max(Math.min(MIN_WIDTH, vp.width - EDGE * 2), Math.min(g.width, vp.width - EDGE * 2))
  const height = Math.max(Math.min(MIN_HEIGHT, vp.height - EDGE * 2), Math.min(g.height, vp.height - EDGE * 2))
  const x = Math.min(Math.max(EDGE, g.x), vp.width - width - EDGE)
  const y = Math.min(Math.max(EDGE, g.y), vp.height - height - EDGE)
  return { x, y, width, height }
}

export default function RunnerTerminalWindow({ runner, onGoingTask, offset, zIndex, onFocus, onClose }: RunnerTerminalWindowProps) {
  const hostRef = useRef<HTMLDivElement>(null)
  const termRef = useRef<Terminal | null>(null)
  const fitRef = useRef<FitAddon | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const dragRef = useRef<{ kind: 'move' | 'resize'; startX: number; startY: number; start: Geometry } | null>(null)

  const [geometry, setGeometry] = useState<Geometry>(() => clampGeometry(initialGeometry(offset)))
  const [state, setState] = useState<ConnectionState>('connecting')
  const [detail, setDetail] = useState('')
  const [attempt, setAttempt] = useState(0)
  const [termReady, setTermReady] = useState(false)

  // one xterm per window, opened once the bundled font has loaded so that
  // xterm measures the cell size with it
  useEffect(() => {
    const host = hostRef.current
    if (!host) {
      return
    }

    let disposed = false
    let dispose = () => {}

    document.fonts.load(`13px ${FONT_FAMILY}`).catch(() => {}).finally(() => {
      if (disposed) {
        return
      }

      const term = new Terminal({
        cursorBlink: true,
        cursorStyle: 'bar',
        cursorInactiveStyle: 'outline',
        fontFamily: FONT_FAMILY,
        fontSize: 13,
        lineHeight: 1.2,
        scrollback: 5000,
        theme: THEME,
      })
      const fit = new FitAddon()
      term.loadAddon(fit)
      term.open(host)
      fit.fit()

      termRef.current = term
      fitRef.current = fit

      let timer = 0
      const observer = new ResizeObserver(() => {
        window.clearTimeout(timer)
        timer = window.setTimeout(() => {
          try {
            fit.fit()
          } catch {
            // the window is closing
          }
        }, 80)
      })
      observer.observe(host)

      const resizeListener = term.onResize(({ cols, rows }) => {
        const ws = wsRef.current
        if (ws && ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: 'resize', cols, rows }))
        }
      })

      dispose = () => {
        window.clearTimeout(timer)
        observer.disconnect()
        resizeListener.dispose()
        term.dispose()
        termRef.current = null
        fitRef.current = null
      }
      setTermReady(true)
    })

    return () => {
      disposed = true
      dispose()
    }
  }, [])

  // one session per attempt; closing the window or leaving the page ends it
  useEffect(() => {
    const term = termRef.current
    const fit = fitRef.current
    if (!termReady || !term || !fit) {
      return
    }

    let cancelled = false
    let ready = false
    let exited = false
    let errorMessage = ''
    let ws: WebSocket | null = null
    const encoder = new TextEncoder()

    setState('connecting')
    setDetail('')
    if (attempt > 0) {
      term.write('\r\n\x1b[90m--- reconnecting: a new session starts ---\x1b[0m\r\n')
    }

    const send = (bytes: Uint8Array<ArrayBuffer>) => {
      if (ready && ws && ws.readyState === WebSocket.OPEN) {
        ws.send(bytes)
      }
    }
    const dataListener = term.onData((data) => send(encoder.encode(data)))
    const binaryListener = term.onBinary((data) => send(Uint8Array.from(data, (c) => c.charCodeAt(0) & 0xff)))

    async function connect() {
      try {
        fit!.fit()
      } catch {
        // not laid out yet; the server defaults to 80x24
      }

      let ticket = ''
      try {
        const response = await terminalApi.createTerminalTicket({ runner, cols: term!.cols, rows: term!.rows })
        ticket = response.data.ticket || ''
      } catch (error: unknown) {
        if (!cancelled) {
          setState('error')
          setDetail(apiErrorMessage(error, 'Failed to open a terminal'))
        }
        return
      }
      if (cancelled) {
        return
      }

      ws = new WebSocket(terminalWebSocketUrl(), [TERMINAL_SUBPROTOCOL, `${TERMINAL_TICKET_PREFIX}${ticket}`])
      ws.binaryType = 'arraybuffer'
      wsRef.current = ws

      ws.onmessage = (event: MessageEvent) => {
        if (typeof event.data !== 'string') {
          term!.write(new Uint8Array(event.data as ArrayBuffer))
          return
        }

        const msg = parseTerminalMessage(event.data)
        if (!msg) {
          return
        }
        if (msg.type === 'ready') {
          ready = true
          setState('connected')
          term!.focus()
          ws?.send(JSON.stringify({ type: 'resize', cols: term!.cols, rows: term!.rows }))
        } else if (msg.type === 'exit') {
          exited = true
          term!.write(`\r\n\x1b[90m[shell exited with code ${msg.code ?? 0}]\x1b[0m\r\n`)
        } else if (msg.type === 'error') {
          errorMessage = msg.message || 'Terminal error'
          term!.write(`\r\n\x1b[31m${errorMessage}\x1b[0m\r\n`)
        }
      }

      ws.onclose = (event: CloseEvent) => {
        if (wsRef.current === ws) {
          wsRef.current = null
        }
        if (cancelled) {
          return
        }

        if (exited) {
          setState('closed')
          setDetail('Shell exited')
        } else if (errorMessage) {
          setState('error')
          setDetail(errorMessage)
        } else if (!ready && event.code === 1006) {
          setState('error')
          setDetail('The controller refused the connection')
        } else {
          setState(event.code === 1000 ? 'closed' : 'error')
          setDetail(describeTerminalClose(event.code, event.reason))
        }
      }
    }

    void connect()

    return () => {
      cancelled = true
      dataListener.dispose()
      binaryListener.dispose()
      if (ws) {
        if (wsRef.current === ws) {
          wsRef.current = null
        }
        ws.close(1000, 'window closed')
      }
    }
  }, [runner, attempt, termReady])

  useEffect(() => {
    const onWindowResize = () => setGeometry((g) => clampGeometry(g))
    window.addEventListener('resize', onWindowResize)
    return () => window.removeEventListener('resize', onWindowResize)
  }, [])

  const startDrag = useCallback((kind: 'move' | 'resize') => (event: ReactPointerEvent<HTMLElement>) => {
    if (event.button !== 0) {
      return
    }
    if (kind === 'move' && (event.target as HTMLElement).closest('button')) {
      return
    }

    event.preventDefault()
    event.currentTarget.setPointerCapture(event.pointerId)
    dragRef.current = { kind, startX: event.clientX, startY: event.clientY, start: geometry }
    onFocus()
  }, [geometry, onFocus])

  const onDrag = useCallback((event: ReactPointerEvent<HTMLElement>) => {
    const drag = dragRef.current
    if (!drag) {
      return
    }

    const dx = event.clientX - drag.startX
    const dy = event.clientY - drag.startY
    if (drag.kind === 'move') {
      setGeometry(clampGeometry({ ...drag.start, x: drag.start.x + dx, y: drag.start.y + dy }))
    } else {
      const vp = viewport()
      setGeometry(clampGeometry({
        ...drag.start,
        width: Math.min(Math.max(MIN_WIDTH, drag.start.width + dx), vp.width - drag.start.x - EDGE),
        height: Math.min(Math.max(MIN_HEIGHT, drag.start.height + dy), vp.height - drag.start.y - EDGE),
      }))
    }
  }, [])

  const endDrag = useCallback((event: ReactPointerEvent<HTMLElement>) => {
    if (dragRef.current) {
      dragRef.current = null
      event.currentTarget.releasePointerCapture(event.pointerId)
    }
  }, [])

  const canRetry = state === 'closed' || state === 'error'

  return createPortal(
    <section
      className={styles.window}
      style={{ left: geometry.x, top: geometry.y, width: geometry.width, height: geometry.height, zIndex }}
      role="dialog"
      aria-label={`Terminal on ${runner}`}
      onPointerDownCapture={onFocus}
      data-runner={runner}
      data-state={state}
    >
      <header
        className={styles.titleBar}
        onPointerDown={startDrag('move')}
        onPointerMove={onDrag}
        onPointerUp={endDrag}
        onPointerCancel={endDrag}
      >
        <svg className={styles.titleIcon} viewBox="0 0 24 24" aria-hidden="true" focusable="false">
          <path d="M4.3 6.3 9.9 12l-5.6 5.7-1.4-1.4L7.1 12 2.9 7.7l1.4-1.4zM11 16h10v2H11v-2z" fill="currentColor" />
        </svg>
        <span className={styles.title}>{runner}</span>
        <span className={`${styles.state} ${STATE_CLASS[state]}`} aria-live="polite">
          <span className={styles.dot} aria-hidden="true" />
          {STATE_LABEL[state]}
        </span>
        <span className={styles.spacer} />
        {canRetry && (
          <button type="button" className={styles.textButton} onClick={() => setAttempt((n) => n + 1)}>
            {state === 'error' ? 'Retry' : 'Reconnect'}
          </button>
        )}
        <button type="button" className={styles.closeButton} onClick={onClose} aria-label={`Close terminal on ${runner}`}>
          <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
            <path d="M6.4 5 12 10.6 17.6 5 19 6.4 13.4 12 19 17.6 17.6 19 12 13.4 6.4 19 5 17.6 10.6 12 5 6.4z" fill="currentColor" />
          </svg>
        </button>
      </header>

      {onGoingTask !== 0 && (
        <p className={styles.warning}>
          This runner is running task #{onGoingTask}. Commands here can affect the test, and containers are removed when the task ends.
        </p>
      )}
      {canRetry && detail && (
        <p className={state === 'error' ? styles.error : styles.notice} role="status">{detail}</p>
      )}

      <div className={styles.terminalHost} ref={hostRef} />

      <div
        className={styles.resizeHandle}
        onPointerDown={startDrag('resize')}
        onPointerMove={onDrag}
        onPointerUp={endDrag}
        onPointerCancel={endDrag}
        aria-hidden="true"
      />
    </section>,
    document.body,
  )
}
