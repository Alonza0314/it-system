import { useEffect, useRef } from 'react'
import SearchableSelect, { type SearchableOption } from '../searchableSelect/SearchableSelect'
import styles from './nf-pr-row.module.css'

export interface PrOption {
  number: number
  title: string
}

// Result of looking up a PR that is not in the fetched list
export type PrLookup =
  | { status: 'loading' }
  | { status: 'found'; title: string; state: string }
  | { status: 'notFound' }
  | { status: 'error' }

interface NfPrRowProps {
  nfOptions: SearchableOption[]
  nf: string
  prOptions: PrOption[]
  pr: string
  isLoadingPrs: boolean
  // Lookups of this row's NF, keyed by PR number
  prLookups: Record<string, PrLookup>
  onLookupPr: (number: string) => void
  onSelectNf: (value: string) => void
  onSelectPr: (value: string) => void
  onRemove: () => void
}

const LOOKUP_DELAY_MS = 300

function parsePrNumber(query: string) {
  return /^#?([1-9]\d*)$/.exec(query)?.[1] ?? null
}

export default function NfPrRow({
  nfOptions,
  nf,
  prOptions,
  pr,
  isLoadingPrs,
  prLookups,
  onLookupPr,
  onSelectNf,
  onSelectPr,
  onRemove,
}: NfPrRowProps) {
  const lookupTimer = useRef<number | undefined>(undefined)

  useEffect(() => () => window.clearTimeout(lookupTimer.current), [])

  const prSearchOptions = prOptions.map((option) => ({
    value: String(option.number),
    label: `#${option.number} ${option.title}`,
  }))

  function describePr(number: string) {
    const lookup = prLookups[number]
    return lookup?.status === 'found' ? `#${number} ${lookup.title} (${lookup.state})` : `#${number}`
  }

  // The fetched list only has the first page of open PRs. A number that is
  // not in it is looked up on GitHub, so closed and merged PRs can be picked
  // and numbers that do not exist cannot.
  function lookupOption(query: string): SearchableOption | null {
    const number = parsePrNumber(query)
    if (!number) {
      return null
    }

    const lookup = prLookups[number]
    if (!lookup || lookup.status === 'loading') {
      return { value: number, label: `Looking up PR #${number}...`, disabled: true }
    }
    if (lookup.status === 'found') {
      return { value: number, label: describePr(number) }
    }
    if (lookup.status === 'notFound') {
      return { value: number, label: `PR #${number} not found`, disabled: true }
    }
    return { value: number, label: `Use PR #${number} (could not look it up)` }
  }

  function handleQueryChange(query: string) {
    window.clearTimeout(lookupTimer.current)

    const number = parsePrNumber(query)
    if (!number || prOptions.some((option) => String(option.number) === number) || prLookups[number]) {
      return
    }
    lookupTimer.current = window.setTimeout(() => onLookupPr(number), LOOKUP_DELAY_MS)
  }

  return (
    <div className={styles.row}>
      <div className={styles.field}>
        <span className={styles.label}>NF</span>
        <SearchableSelect
          options={nfOptions}
          value={nf}
          onChange={onSelectNf}
          placeholder="Select NF"
          searchPlaceholder="Search NF"
          ariaLabel="NF"
        />
      </div>

      <div className={styles.field}>
        <span className={styles.label}>PR</span>
        <SearchableSelect
          options={prSearchOptions}
          value={pr}
          onChange={onSelectPr}
          placeholder={nf ? 'Select PR' : 'Select NF first'}
          searchPlaceholder="Search PR # or title"
          ariaLabel={`${nf.toUpperCase()} PR`}
          disabled={!nf}
          loading={isLoadingPrs}
          formatValue={describePr}
          customOption={lookupOption}
          onQueryChange={handleQueryChange}
        />
      </div>

      <button
        type="button"
        className={styles.removeButton}
        onClick={onRemove}
        aria-label="Remove NF PR"
      >
        ✕
      </button>
    </div>
  )
}
