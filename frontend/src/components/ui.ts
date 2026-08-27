/**
 * Class strings shared by elements that must look identical but cannot be the
 * same component: a submit button and a router Link both render as buttons.
 */

const base =
  'inline-block rounded-lg px-4 py-2 text-base transition ' +
  'focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-accent ' +
  'disabled:cursor-not-allowed disabled:opacity-50 no-underline'

export const primaryButton =
  `${base} bg-accent text-white border border-accent hover:brightness-110`

export const ghostButton =
  `${base} border border-edge text-body hover:border-accent`

export const field =
  'w-full rounded-lg border border-edge bg-raised px-3 py-2 text-strong ' +
  'focus:outline-2 focus:outline-offset-1 focus:outline-accent'

export const card = 'rounded-xl border border-edge bg-surface'
