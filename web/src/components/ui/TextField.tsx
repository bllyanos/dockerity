import { useId, type InputHTMLAttributes } from 'react'
import { Icon, type IconName } from './Icon'

type Props = InputHTMLAttributes<HTMLInputElement> & {
  label: string
  /** Help text under the field. */
  supporting?: string
  /** Replaces the supporting text and marks the field invalid. */
  error?: string | null
  leadingIcon?: IconName
}

/**
 * An outlined text field with a floating label. The label sits in the border
 * notch, whose fill comes from --field-notch (set by the container it is on).
 */
export function TextField({ label, supporting, error, leadingIcon, id, className = '', placeholder, ...input }: Props) {
  const autoId = useId()
  const inputId = id ?? input.name ?? autoId
  const helpId = `${inputId}-help`
  const help = error || supporting
  // With a placeholder the label stays floated so the two never overlap.
  const resting = placeholder
    ? ''
    : `peer-placeholder-shown:top-1/2 peer-placeholder-shown:bg-transparent peer-placeholder-shown:px-0 peer-placeholder-shown:body-large ${leadingIcon ? 'peer-placeholder-shown:left-12' : 'peer-placeholder-shown:left-4'}`

  return (
    <div className={`flex min-w-0 flex-col gap-1 ${className}`}>
      <div className="relative">
        {leadingIcon && (
          <span className="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-on-surface-variant">
            <Icon name={leadingIcon} size={24} />
          </span>
        )}
        <input
          id={inputId}
          placeholder={placeholder ?? ' '}
          aria-invalid={error ? true : undefined}
          aria-describedby={help ? helpId : undefined}
          className={`peer h-14 w-full rounded-xs bg-transparent body-large text-on-surface caret-primary outline-none placeholder:text-on-surface-variant/70
            ${leadingIcon ? 'pr-4 pl-12' : 'px-4'}
            ${error ? 'shadow-[inset_0_0_0_2px_var(--color-error)]' : 'shadow-[inset_0_0_0_1px_var(--color-outline)] hover:shadow-[inset_0_0_0_1px_var(--color-on-surface)] focus:shadow-[inset_0_0_0_3px_var(--color-primary)]'}
            disabled:opacity-38`}
          {...input}
        />
        <label htmlFor={inputId}
          className={`pointer-events-none absolute top-0 left-3 -translate-y-1/2 bg-[var(--field-notch,var(--color-surface))] px-1 body-small transition-all duration-150 ease-emphasized
            ${error ? 'text-error' : 'text-on-surface-variant peer-focus:text-primary'}
            ${resting} peer-focus:top-0 peer-focus:left-3 peer-focus:px-1 peer-focus:body-small peer-focus:bg-[var(--field-notch,var(--color-surface))]`}>
          {label}
        </label>
      </div>
      {help && (
        <p id={helpId} className={`px-4 body-small ${error ? 'text-error' : 'text-on-surface-variant'}`}>{help}</p>
      )}
    </div>
  )
}
