"use client";

import type { UseFormReturn } from "react-hook-form";

import {
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";

import type { ClaudeAgentOptionsForm } from "../schema";
import { StringListEditor } from "./string-list-editor";

export function SandboxField({ form, disabled }: { form: UseFormReturn<ClaudeAgentOptionsForm>; disabled?: boolean }) {
  return (
    <FormField
      control={form.control}
      name="sandbox"
      render={({ field }) => {
        const val = field.value ?? {
          enabled: false,
          autoAllowBashIfSandboxed: true,
          excludedCommands: [],
          allowUnsandboxedCommands: true,
          enableWeakerNestedSandbox: false,
        };
        const update = (patch: Record<string, unknown>) => field.onChange({ ...val, ...patch });

        return (
          <FormItem className="space-y-4">
            <div className="flex items-center gap-2">
              <Switch checked={val.enabled} onCheckedChange={(c) => update({ enabled: c })} disabled={disabled} />
              <FormLabel>Enable Bash Sandboxing</FormLabel>
            </div>
            {val.enabled && (
              <div className="space-y-4 pl-4 border-l-2 border-muted">
                <div className="flex items-center gap-2">
                  <Switch checked={val.autoAllowBashIfSandboxed ?? true} onCheckedChange={(c) => update({ autoAllowBashIfSandboxed: c })} disabled={disabled} />
                  <Label>Auto-approve bash when sandboxed</Label>
                </div>
                <div className="flex items-center gap-2">
                  <Switch checked={val.allowUnsandboxedCommands ?? true} onCheckedChange={(c) => update({ allowUnsandboxedCommands: c })} disabled={disabled} />
                  <Label>Allow unsandboxed commands</Label>
                </div>
                <div className="flex items-center gap-2">
                  <Switch checked={val.enableWeakerNestedSandbox ?? false} onCheckedChange={(c) => update({ enableWeakerNestedSandbox: c })} disabled={disabled} />
                  <Label>Enable weaker nested sandbox (Docker/Linux)</Label>
                </div>
                <div>
                  <Label>Excluded Commands</Label>
                  <p className="text-xs text-muted-foreground mb-2">Commands that run outside the sandbox</p>
                  <StringListEditor value={val.excludedCommands ?? []} onChange={(v) => update({ excludedCommands: v })} placeholder="command name" />
                </div>
                <div className="space-y-3">
                  <Label>Network</Label>
                  <div className="space-y-2 pl-2">
                    <div className="flex items-center gap-2">
                      <Switch checked={val.network?.allowAllUnixSockets ?? false} disabled={disabled} onCheckedChange={(c) => update({ network: { ...(val.network ?? {}), allowAllUnixSockets: c } })} />
                      <Label>Allow all Unix sockets</Label>
                    </div>
                    <div className="flex items-center gap-2">
                      <Switch checked={val.network?.allowLocalBinding ?? false} disabled={disabled} onCheckedChange={(c) => update({ network: { ...(val.network ?? {}), allowLocalBinding: c } })} />
                      <Label>Allow local port binding (macOS)</Label>
                    </div>
                    <div>
                      <Label className="text-xs text-muted-foreground">Allowed Unix Sockets</Label>
                      <StringListEditor value={val.network?.allowUnixSockets ?? []} onChange={(v) => update({ network: { ...(val.network ?? {}), allowUnixSockets: v } })} placeholder="/var/run/docker.sock" />
                    </div>
                  </div>
                </div>
              </div>
            )}
            <FormMessage />
          </FormItem>
        );
      }}
    />
  );
}
