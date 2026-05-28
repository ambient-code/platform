'use client'

import { QueryClientProvider } from '@tanstack/react-query'
import { useState } from 'react'
import { getQueryClient } from '@/lib/query-client'

type QueryProviderProps = {
  children: React.ReactNode
}

export function QueryProvider({ children }: QueryProviderProps) {
  const [queryClient] = useState(() => getQueryClient())

  return (
    <QueryClientProvider client={queryClient}>
      {children}
    </QueryClientProvider>
  )
}
