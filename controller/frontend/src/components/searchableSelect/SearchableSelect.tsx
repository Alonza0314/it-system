import { useEffect, useMemo, useRef, useState, type KeyboardEvent } from 'react'
import styles from './searchable-select.module.css'

export interface SearchableOption {
  value: string
  label: string
  // Shown but cannot be picked, e.g. "PR #1234 not found"
  disabled?: boolean
}

interface SearchableSelectProps {
  options: SearchableOption[]
  value: string
  onChange: (value: string) => void
  placeholder: string
  searchPlaceholder?: string
  ariaLabel?: string
  disabled?: boolean
  loading?: boolean
  // Shown for a value that is not in options, e.g. a PR typed by number
  formatValue?: (value: string) => string
  // Extra option offered for a query, e.g. "Use PR #1234" for a PR not in the list
  customOption?: (query: string) => SearchableOption | null
  onQueryChange?: (query: string) => void
}

export default function SearchableSelect({
  options,
  value,
  onChange,
  placeholder,
  searchPlaceholder = 'Search',
  ariaLabel,
  disabled = false,
  loading = false,
  formatValue = (item) => item,
  customOption,
  onQueryChange,
}: SearchableSelectProps) {
  const [isOpen, setIsOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [highlight, setHighlight] = useState(0)
  const rootRef = useRef<HTMLDivElement>(null)
  const searchRef = useRef<HTMLInputElement>(null)
  const listRef = useRef<HTMLUListElement>(null)

  const filtered = useMemo(() => {
    const keyword = query.trim().toLowerCase()
    const matches = keyword
      ? options.filter((option) => option.label.toLowerCase().includes(keyword))
      : options
    const custom = customOption?.(query.trim())
    if (!custom) {
      return matches
    }

    // The query names one value exactly (e.g. PR #1), so put it first, using
    // the listed option when there is one
    const exact = options.find((option) => option.value === custom.value) ?? custom
    return [exact, ...matches.filter((option) => option.value !== custom.value)]
  }, [options, query, customOption])

  const selectedLabel = value
    ? options.find((option) => option.value === value)?.label ?? formatValue(value)
    : ''

  useEffect(() => {
    if (!isOpen) {
      return
    }

    searchRef.current?.focus()

    function handleClickOutside(event: MouseEvent) {
      if (rootRef.current && !rootRef.current.contains(event.target as Node)) {
        setIsOpen(false)
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    return () => {
      document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [isOpen])

  useEffect(() => {
    listRef.current?.querySelector(`[data-index="${highlight}"]`)?.scrollIntoView({ block: 'nearest' })
  }, [highlight])

  function toggleOpen() {
    if (!isOpen) {
      setQuery('')
      setHighlight(0)
    }
    setIsOpen(!isOpen)
  }

  function select(option: SearchableOption) {
    if (option.disabled) {
      return
    }
    onChange(option.value)
    setIsOpen(false)
  }

  function handleKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (event.key === 'ArrowDown') {
      event.preventDefault()
      setHighlight((prev) => Math.min(prev + 1, filtered.length - 1))
    } else if (event.key === 'ArrowUp') {
      event.preventDefault()
      setHighlight((prev) => Math.max(prev - 1, 0))
    } else if (event.key === 'Enter') {
      event.preventDefault()
      if (filtered[highlight]) {
        select(filtered[highlight])
      }
    } else if (event.key === 'Escape') {
      setIsOpen(false)
    }
  }

  return (
    <div className={styles.root} ref={rootRef}>
      <button
        type="button"
        className={styles.trigger}
        onClick={toggleOpen}
        disabled={disabled}
        aria-label={ariaLabel}
        aria-haspopup="listbox"
        aria-expanded={isOpen}
      >
        <span className={selectedLabel ? styles.value : styles.placeholder}>
          {selectedLabel || placeholder}
        </span>
        <span className={styles.chevron} aria-hidden="true">▾</span>
      </button>

      {isOpen && (
        <div className={styles.popup}>
          <input
            ref={searchRef}
            type="search"
            className={styles.search}
            placeholder={searchPlaceholder}
            value={query}
            onChange={(event) => {
              setQuery(event.target.value)
              setHighlight(0)
              onQueryChange?.(event.target.value.trim())
            }}
            onKeyDown={handleKeyDown}
          />
          <ul className={styles.list} role="listbox" ref={listRef}>
            {loading && <li className={styles.hint}>Loading...</li>}
            {!loading && filtered.length === 0 && <li className={styles.hint}>No match</li>}
            {filtered.map((option, index) => (
              <li
                key={option.value}
                data-index={index}
                role="option"
                aria-selected={option.value === value}
                aria-disabled={option.disabled}
                className={[
                  styles.option,
                  index === highlight ? styles.highlighted : '',
                  option.value === value ? styles.selected : '',
                  option.disabled ? styles.disabled : '',
                ].join(' ')}
                onMouseEnter={() => setHighlight(index)}
                onClick={() => select(option)}
              >
                {option.label}
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  )
}
