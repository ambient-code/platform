"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import {
  FolderOpen,
  Home,
  Plug,
  Plus,
  Settings,
  Terminal,
} from "lucide-react";
import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandShortcut,
} from "@/components/ui/command";
import { useProjects } from "@/services/queries/use-projects";

export function CommandPalette() {
  const [open, setOpen] = React.useState(false);
  const router = useRouter();
  const { data: projects } = useProjects();

  React.useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (e.key === "k" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        setOpen((prev) => !prev);
      }
    };

    document.addEventListener("keydown", down);
    return () => document.removeEventListener("keydown", down);
  }, []);

  const runCommand = React.useCallback(
    (command: () => void) => {
      setOpen(false);
      command();
    },
    []
  );

  const projectList = projects ?? [];

  return (
    <CommandDialog
      open={open}
      onOpenChange={setOpen}
      showCloseButton={false}
    >
      <CommandInput placeholder="Type a command or search..." />
      <CommandList>
        <CommandEmpty>No results found.</CommandEmpty>

        {projectList.length > 0 && (
          <CommandGroup heading="Projects">
            {projectList.slice(0, 5).map((project) => (
              <CommandItem
                key={project.name}
                value={`project ${project.name} ${project.displayName}`}
                onSelect={() =>
                  runCommand(() =>
                    router.push(
                      `/projects/${encodeURIComponent(project.name)}/sessions`
                    )
                  )
                }
              >
                <FolderOpen />
                <span>{project.displayName || project.name}</span>
              </CommandItem>
            ))}
          </CommandGroup>
        )}

        <CommandGroup heading="Actions">
          <CommandItem
            value="new session"
            onSelect={() =>
              runCommand(() => router.push("/"))
            }
          >
            <Plus />
            <span>New Session</span>
          </CommandItem>
        </CommandGroup>

        <CommandGroup heading="Navigation">
          <CommandItem
            value="home dashboard"
            onSelect={() => runCommand(() => router.push("/"))}
          >
            <Home />
            <span>Home</span>
            <CommandShortcut>Home</CommandShortcut>
          </CommandItem>
          <CommandItem
            value="projects list"
            onSelect={() => runCommand(() => router.push("/projects"))}
          >
            <Terminal />
            <span>Projects</span>
          </CommandItem>
          <CommandItem
            value="integrations connections"
            onSelect={() => runCommand(() => router.push("/integrations"))}
          >
            <Plug />
            <span>Integrations</span>
          </CommandItem>
          {projectList.length > 0 && (
            <CommandItem
              value="project settings"
              onSelect={() =>
                runCommand(() =>
                  router.push(
                    `/projects/${encodeURIComponent(projectList[0].name)}/settings`
                  )
                )
              }
            >
              <Settings />
              <span>Settings</span>
            </CommandItem>
          )}
        </CommandGroup>
      </CommandList>
    </CommandDialog>
  );
}
