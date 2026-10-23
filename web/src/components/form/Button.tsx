import type { ButtonHTMLAttributes } from 'react'

type Variant = 'primary' | 'secondary' | 'danger'

const variants: Record<Variant, string> = {
  primary: 'bg-slate-900 text-white hover:bg-slate-700',
  secondary: 'border border-slate-300 bg-white text-slate-700 hover:bg-slate-50',
  danger: 'bg-red-600 text-white hover:bg-red-500',
}

type Props = ButtonHTMLAttributes<HTMLButtonElement> & { variant?: Variant }

export function Button({ variant = 'primary', className = '', ...props }: Props) {
  return (
    <button
      className={`rounded-md px-3 py-2 text-sm font-medium disabled:opacity-50 ${variants[variant]} ${className}`}
      {...props}
    />
  )
}
