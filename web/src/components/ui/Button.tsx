import type { ButtonHTMLAttributes } from 'react'
import { Link, type LinkProps } from 'react-router'
import { buttonClass, type ButtonVariant } from './buttonStyles'
import { Icon, type IconName } from './Icon'

type Common = { variant?: ButtonVariant; icon?: IconName }

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & Common

/** A Tonal button: one filled button per view, tonal for the strong second choice. */
export function Button({ variant = 'filled', icon, className = '', type = 'button', children, ...props }: ButtonProps) {
  return (
    <button type={type} className={buttonClass(variant, !!icon, className)} {...props}>
      {icon && <Icon name={icon} size={18} />}
      {children}
    </button>
  )
}

type LinkButtonProps = LinkProps & Common

/** A link that looks like a button, for actions that navigate. */
export function ButtonLink({ variant = 'filled', icon, className = '', children, ...props }: LinkButtonProps) {
  return (
    <Link className={buttonClass(variant, !!icon, className)} {...props}>
      {icon && <Icon name={icon} size={18} />}
      {children}
    </Link>
  )
}
