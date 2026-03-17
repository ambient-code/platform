import { Skeleton } from "@/components/ui/skeleton";

export function CardSkeleton() {
  return (
    <div
      className="flex items-start justify-between gap-3 p-3 border rounded-lg bg-background/50"
      aria-hidden
    >
      <div className="flex-1 min-w-0 space-y-2">
        <div className="flex items-center gap-2">
          <Skeleton className="h-4 w-4 rounded-full flex-shrink-0" />
          <Skeleton className="h-4 w-20" />
        </div>
        <Skeleton className="h-3 w-full max-w-[240px]" />
      </div>
    </div>
  );
}
