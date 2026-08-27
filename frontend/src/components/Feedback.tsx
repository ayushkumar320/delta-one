/** The three states every data-loading page shares. */
export function Loading({ what }: { what: string }) {
  return <p className="muted">Loading {what}…</p>
}

export function Problem({ message }: { message: string }) {
  return (
    <p className="problem" role="alert">
      {message}
    </p>
  )
}

export function Empty({ message }: { message: string }) {
  return <p className="muted">{message}</p>
}
