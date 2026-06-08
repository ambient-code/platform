import { cn } from '@/lib/utils'

export function MetaRow({ label, value, mono }: { label: string; value: React.ReactNode; mono?: boolean }) {
  return (
    <div>
      <dt className="text-muted-foreground">{label}</dt>
      <dd className={cn('mt-0.5', mono && 'font-mono text-xs')}>
        {value ?? <NoValue />}
      </dd>
    </div>
  )
}

export function NoValue() {
  return <span className="text-muted-foreground">&mdash;</span>
}
