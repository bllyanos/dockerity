import type { InputHTMLAttributes } from 'react'

type Props = InputHTMLAttributes<HTMLInputElement> & { label: string }

export function TextField({ label, id, ...input }: Props) {
  const inputId = id ?? input.name
  return (
    <label htmlFor={inputId} className="block text-sm">
      <span className="mb-1 block font-medium text-slate-700">{label}</span>
      <input
        id={inputId}
        className="w-full rounded-md border border-slate-300 px-3 py-2 text-slate-900 focus:border-slate-500 focus:outline-none"
        {...input}
      />
    </label>
  )
}
