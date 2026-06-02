import { useQuery } from "@tanstack/react-query"

export type CurrentUser = {
  username: string
  name: string
  email: string
  initials: string
}

type MeResponse = {
  authenticated: boolean
  username?: string
  name?: string
  email?: string
  initials?: string
}

async function fetchCurrentUser(): Promise<CurrentUser | null> {
  const res = await fetch("/api/me")
  if (!res.ok) {
    throw new Error(`/api/me returned ${res.status}`)
  }

  const data: MeResponse = await res.json()
  if (!data.authenticated) return null

  return {
    username: data.username ?? "unknown",
    name: data.name ?? "",
    email: data.email ?? "",
    initials: data.initials ?? "?",
  }
}

export function useCurrentUser(): { user: CurrentUser | null; isLoading: boolean } {
  const { data, isLoading } = useQuery({
    queryKey: ["current-user"],
    queryFn: fetchCurrentUser,
    staleTime: 5 * 60 * 1000,
    retry: false,
  })

  return {
    user: data ?? null,
    isLoading,
  }
}
