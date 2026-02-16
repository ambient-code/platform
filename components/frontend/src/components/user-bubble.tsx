"use client";

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { useCurrentUser } from "@/services/queries";
import { useApiSource } from "@/contexts/api-source-context";

export function UserBubble() {
  const { isApiServer } = useApiSource();
  const { data: me, isLoading } = useCurrentUser({ enabled: !isApiServer });

  const initials = (me?.displayName || me?.username || me?.email || "?")
    .split(/[\s@._-]+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((s) => s[0]?.toUpperCase())
    .join("");

  if (isApiServer) {
    return (
      <div className="inline-flex items-center gap-2 p-1 pr-2">
        <Avatar>
          <AvatarFallback>AP</AvatarFallback>
        </Avatar>
        <span className="hidden sm:block text-sm text-muted-foreground">API User</span>
      </div>
    );
  }

  if (isLoading || !me) return <div className="w-8 h-8 rounded-full bg-muted animate-pulse" />;

  if (!me.authenticated) {
    return (
      <span className="inline-flex items-center justify-center whitespace-nowrap text-sm font-medium">Sign in</span>
    );
  }

  return (
    <div className="inline-flex items-center gap-2 p-1 pr-2">
      <Avatar>
        <AvatarImage alt={me.displayName || initials} />
        <AvatarFallback>{initials || "?"}</AvatarFallback>
      </Avatar>
      <span className="hidden sm:block text-sm text-muted-foreground">{me.displayName}</span>
    </div>
  );
}


