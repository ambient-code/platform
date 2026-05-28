"use client";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { LogIn } from "lucide-react";

type SessionExpiredDialogProps = {
  open: boolean;
};

export function SessionExpiredDialog({ open }: SessionExpiredDialogProps) {
  const handleLogin = () => {
    const returnTo = window.location.pathname + window.location.search;
    window.location.href = `/api/auth/sso/login?returnTo=${encodeURIComponent(returnTo)}`;
  };

  return (
    <Dialog open={open}>
      <DialogContent
        className="sm:max-w-md"
        showCloseButton={false}
        onPointerDownOutside={(e) => e.preventDefault()}
        onEscapeKeyDown={(e) => e.preventDefault()}
      >
        <DialogHeader>
          <DialogTitle>Session expired</DialogTitle>
          <DialogDescription>
            Your session has expired. Any monitored sessions will resume when you
            log back in.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button onClick={handleLogin} className="w-full">
            <LogIn className="mr-2 h-4 w-4" />
            Log in
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
