import SearchableSelect, { type SearchableOption } from '../searchableSelect/SearchableSelect'
import styles from './nf-pr-row.module.css'

export interface PrOption {
  number: number
  title: string
}

interface NfPrRowProps {
  nfOptions: SearchableOption[]
  nf: string
  prOptions: PrOption[]
  pr: string
  isLoadingPrs: boolean
  onSelectNf: (value: string) => void
  onSelectPr: (value: string) => void
  onRemove: () => void
}

// Lets a PR that is not in the fetched list be used by typing its number,
// e.g. a closed PR or one beyond the first page returned by GitHub.
function prNumberOption(query: string): SearchableOption | null {
  const match = /^#?([1-9]\d*)$/.exec(query)
  return match ? { value: match[1], label: `Use PR #${match[1]}` } : null
}

export default function NfPrRow({
  nfOptions,
  nf,
  prOptions,
  pr,
  isLoadingPrs,
  onSelectNf,
  onSelectPr,
  onRemove,
}: NfPrRowProps) {
  const prSearchOptions = prOptions.map((option) => ({
    value: String(option.number),
    label: `#${option.number} ${option.title}`,
  }))

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
          formatValue={(value) => `#${value}`}
          customOption={prNumberOption}
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
