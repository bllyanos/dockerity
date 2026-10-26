export type ButtonVariant = 'filled' | 'tonal' | 'outlined' | 'text'

const variants: Record<ButtonVariant, string> = {
  filled: 'bg-primary text-on-primary hover:shadow-elevation-1',
  tonal: 'bg-secondary-container text-on-secondary-container',
  outlined: 'text-primary shadow-[inset_0_0_0_1px_var(--color-outline)]',
  text: 'text-primary',
}

/** Classes for a Tonal button, shared by <Button> and link-styled buttons. */
export function buttonClass(variant: ButtonVariant, hasIcon: boolean, extra = ''): string {
  const pad = variant === 'text' ? 'px-3' : hasIcon ? 'pr-6 pl-4' : 'px-6'
  return [
    'state-layer inline-flex h-10 shrink-0 cursor-pointer items-center justify-center gap-2 rounded-full whitespace-nowrap label-large no-underline',
    'transition-shadow duration-200 ease-emphasized',
    'disabled:cursor-default disabled:text-on-surface/38 disabled:shadow-none',
    variant === 'text' ? '' : 'disabled:bg-on-surface/12',
    pad,
    variants[variant],
    extra,
  ].join(' ')
}
