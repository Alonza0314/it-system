import Switch from '../switch/switch'
import styles from './nf-pr-selector.module.css'

export interface PrOption {
  number: number
  title: string
}

interface NfPrSelectorProps {
  label: string
  checked: boolean
  options: PrOption[]
  selectedPr: string
  disabled?: boolean
  onToggle: (checked: boolean) => void
  onSelectPr: (value: string) => void
}

export default function NfPrSelector({
  label,
  checked,
  options,
  selectedPr,
  disabled = false,
  onToggle,
  onSelectPr,
}: NfPrSelectorProps) {
  return (
    <div className={styles.row}>
      <div className={styles.labelWrap}>
        <p className={styles.label}>{label}</p>
      </div>

      <div className={styles.toggleWrap}>
        <Switch
          checked={checked}
          onChange={onToggle}
          disabled={disabled}
        />
      </div>

      <div className={styles.selectWrap}>
        <select
          className={styles.select}
          value={selectedPr}
          onChange={(event) => onSelectPr(event.target.value)}
          disabled={!checked || disabled}
        >
          <option value="">Select PR</option>
          {options.map((option) => (
            <option key={option.number} value={String(option.number)}>
              #{option.number} {option.title}
            </option>
          ))}
        </select>
      </div>
    </div>
  )
}
