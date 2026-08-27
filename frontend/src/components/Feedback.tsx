/** The three states every data-loading page shares. */
export function Loading({ what }: { what: string }) {
  return <p className="text-muted">Loading {what}…</p>
}

export function Problem({ message }: { message: string }) {
  return (
    <p
      role="alert"
      className="my-4 rounded-xl border border-bad bg-bad/10 px-4 py-3 text-red-100"
    >
      {message}
    </p>
  )
}

export function Empty({ message }: { message: string }) {
  return <p className="text-muted">{message}</p>
}
