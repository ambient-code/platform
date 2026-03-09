"use client"

import { useState } from "react"
import {
  CheckCircle2,
  Plus,
  Replace,
  X,
  Settings,
  Terminal,
  PanelLeft,
  ChevronsUpDown,
  Package,
  Zap,
  Eye,
  ChevronDown,
  Clock,
  AlertCircle,
} from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Input } from "@/components/ui/input"
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { Separator } from "@/components/ui/separator"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Checkbox } from "@/components/ui/checkbox"
import { Label } from "@/components/ui/label"
import { Progress } from "@/components/ui/progress"
import { Skeleton } from "@/components/ui/skeleton"
import { Switch } from "@/components/ui/switch"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

// New components to evaluate
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb"
import {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet"
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
  CommandShortcut,
} from "@/components/ui/command"
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible"
import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from "@/components/ui/hover-card"

// ---------------------------------------------------------------------------
// Tier badge helper
// ---------------------------------------------------------------------------
function TierBadge({ tier }: { tier: "replace" | "add" | "skip" | "installed" }) {
  const config = {
    replace: { label: "Replace", variant: "destructive" as const, icon: Replace },
    add: { label: "Add", variant: "default" as const, icon: Plus },
    skip: { label: "Skip", variant: "secondary" as const, icon: X },
    installed: { label: "Installed", variant: "outline" as const, icon: CheckCircle2 },
  }
  const { label, variant, icon: Icon } = config[tier]
  return (
    <Badge variant={variant} className="gap-1">
      <Icon className="size-3" />
      {label}
    </Badge>
  )
}

// ---------------------------------------------------------------------------
// Component demo cards
// ---------------------------------------------------------------------------

function InstalledComponentsSection() {
  return (
    <div className="space-y-6">
      <div className="space-y-1">
        <h2 className="text-xl font-semibold">Currently Installed</h2>
        <p className="text-sm text-muted-foreground">
          30 shadcn components already in the project, with 218 imports across 62 files.
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
        {/* Button */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-base">Button</CardTitle>
              <div className="flex items-center gap-2">
                <Badge variant="outline" className="text-sm font-normal">51 imports</Badge>
                <TierBadge tier="installed" />
              </div>
            </div>
            <CardDescription>Most used component. 6 variants, 8 sizes.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="flex flex-wrap gap-2">
              <Button size="sm">Default</Button>
              <Button size="sm" variant="secondary">Secondary</Button>
              <Button size="sm" variant="outline">Outline</Button>
              <Button size="sm" variant="ghost">Ghost</Button>
              <Button size="sm" variant="destructive">Destructive</Button>
              <Button size="sm" variant="link">Link</Button>
            </div>
            <div className="flex flex-wrap gap-2">
              <Button size="xs">XS</Button>
              <Button size="sm">SM</Button>
              <Button>Default</Button>
              <Button size="lg">LG</Button>
              <Button size="icon"><Settings className="size-4" /></Button>
            </div>
          </CardContent>
        </Card>

        {/* Badge */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-base">Badge</CardTitle>
              <div className="flex items-center gap-2">
                <Badge variant="outline" className="text-sm font-normal">15 imports</Badge>
                <TierBadge tier="installed" />
              </div>
            </div>
            <CardDescription>Status indicators throughout the app.</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap gap-2">
              <Badge>Default</Badge>
              <Badge variant="secondary">Secondary</Badge>
              <Badge variant="destructive">Destructive</Badge>
              <Badge variant="outline">Outline</Badge>
            </div>
          </CardContent>
        </Card>

        {/* Input */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-base">Input</CardTitle>
              <div className="flex items-center gap-2">
                <Badge variant="outline" className="text-sm font-normal">18 imports</Badge>
                <TierBadge tier="installed" />
              </div>
            </div>
            <CardDescription>Form inputs, search fields, session names.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-2">
            <Input placeholder="Session name..." />
            <Input placeholder="Search sessions..." type="search" />
            <Input disabled placeholder="Disabled" />
          </CardContent>
        </Card>

        {/* Table */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-base">Table</CardTitle>
              <div className="flex items-center gap-2">
                <Badge variant="outline" className="text-sm font-normal">6 imports</Badge>
                <TierBadge tier="installed" />
              </div>
            </div>
            <CardDescription>Session lists, API keys, integrations.</CardDescription>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Session</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="text-right">Duration</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow>
                  <TableCell className="font-medium">fix-auth-bug</TableCell>
                  <TableCell><Badge variant="secondary" className="bg-status-success text-status-success-foreground border-0">Running</Badge></TableCell>
                  <TableCell className="text-right">3m 42s</TableCell>
                </TableRow>
                <TableRow>
                  <TableCell className="font-medium">add-rbac</TableCell>
                  <TableCell><Badge variant="outline">Completed</Badge></TableCell>
                  <TableCell className="text-right">12m 08s</TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </CardContent>
        </Card>

        {/* Tabs */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-base">Tabs</CardTitle>
              <div className="flex items-center gap-2">
                <Badge variant="outline" className="text-sm font-normal">4 imports</Badge>
                <TierBadge tier="installed" />
              </div>
            </div>
            <CardDescription>Workspace sections, session details.</CardDescription>
          </CardHeader>
          <CardContent>
            <Tabs defaultValue="chat">
              <TabsList>
                <TabsTrigger value="chat">Chat</TabsTrigger>
                <TabsTrigger value="artifacts">Artifacts</TabsTrigger>
                <TabsTrigger value="logs">Logs</TabsTrigger>
              </TabsList>
              <TabsContent value="chat" className="text-sm text-muted-foreground pt-2">
                Chat messages would appear here.
              </TabsContent>
              <TabsContent value="artifacts" className="text-sm text-muted-foreground pt-2">
                File artifacts from the session.
              </TabsContent>
              <TabsContent value="logs" className="text-sm text-muted-foreground pt-2">
                Container logs output.
              </TabsContent>
            </Tabs>
          </CardContent>
        </Card>

        {/* Accordion */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-base">Accordion</CardTitle>
              <div className="flex items-center gap-2">
                <Badge variant="outline" className="text-sm font-normal">6 imports</Badge>
                <TierBadge tier="installed" />
              </div>
            </div>
            <CardDescription>Session sidebar sections (workflows, repos, MCP).</CardDescription>
          </CardHeader>
          <CardContent>
            <Accordion type="single" collapsible className="w-full">
              <AccordionItem value="workflows">
                <AccordionTrigger className="text-sm">Workflows</AccordionTrigger>
                <AccordionContent className="text-sm text-muted-foreground">
                  3 workflows configured
                </AccordionContent>
              </AccordionItem>
              <AccordionItem value="repos">
                <AccordionTrigger className="text-sm">Repositories</AccordionTrigger>
                <AccordionContent className="text-sm text-muted-foreground">
                  2 repos attached
                </AccordionContent>
              </AccordionItem>
            </Accordion>
          </CardContent>
        </Card>

        {/* Select */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-base">Select</CardTitle>
              <div className="flex items-center gap-2">
                <Badge variant="outline" className="text-sm font-normal">4 imports</Badge>
                <TierBadge tier="installed" />
              </div>
            </div>
            <CardDescription>Model selection, project settings.</CardDescription>
          </CardHeader>
          <CardContent>
            <Select>
              <SelectTrigger className="w-full">
                <SelectValue placeholder="Select model..." />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="sonnet">claude-sonnet-4-6</SelectItem>
                <SelectItem value="opus">claude-opus-4-6</SelectItem>
                <SelectItem value="haiku">claude-haiku-4-5</SelectItem>
              </SelectContent>
            </Select>
          </CardContent>
        </Card>

        {/* Alert */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-base">Alert</CardTitle>
              <div className="flex items-center gap-2">
                <Badge variant="outline" className="text-sm font-normal">9 imports</Badge>
                <TierBadge tier="installed" />
              </div>
            </div>
            <CardDescription>API key warnings, session errors, info banners.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-2">
            <Alert>
              <AlertCircle className="size-4" />
              <AlertTitle>Session queued</AlertTitle>
              <AlertDescription>Waiting for available runner capacity.</AlertDescription>
            </Alert>
          </CardContent>
        </Card>

        {/* Tooltip */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-base">Tooltip</CardTitle>
              <div className="flex items-center gap-2">
                <Badge variant="outline" className="text-sm font-normal">3 imports</Badge>
                <TierBadge tier="installed" />
              </div>
            </div>
            <CardDescription>Status badges, action buttons.</CardDescription>
          </CardHeader>
          <CardContent>
            <TooltipProvider>
              <div className="flex gap-2">
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button variant="outline" size="icon">
                      <Eye className="size-4" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>View session details</TooltipContent>
                </Tooltip>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button variant="outline" size="icon">
                      <Terminal className="size-4" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>Open terminal</TooltipContent>
                </Tooltip>
              </div>
            </TooltipProvider>
          </CardContent>
        </Card>

        {/* Dropdown Menu */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-base">Dropdown Menu</CardTitle>
              <div className="flex items-center gap-2">
                <Badge variant="outline" className="text-sm font-normal">6 imports</Badge>
                <TierBadge tier="installed" />
              </div>
            </div>
            <CardDescription>Session actions, user menu, context menus.</CardDescription>
          </CardHeader>
          <CardContent>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline">
                  Actions <ChevronDown className="size-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent>
                <DropdownMenuLabel>Session</DropdownMenuLabel>
                <DropdownMenuSeparator />
                <DropdownMenuItem>Clone session</DropdownMenuItem>
                <DropdownMenuItem>Export logs</DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem className="text-destructive">Delete session</DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </CardContent>
        </Card>

        {/* Skeleton */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-base">Skeleton</CardTitle>
              <div className="flex items-center gap-2">
                <Badge variant="outline" className="text-sm font-normal">2 imports</Badge>
                <TierBadge tier="installed" />
              </div>
            </div>
            <CardDescription>Loading states throughout the app.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-2">
            <Skeleton className="h-4 w-3/4" />
            <Skeleton className="h-4 w-1/2" />
            <Skeleton className="h-8 w-full" />
          </CardContent>
        </Card>

        {/* Other installed */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-base">Also Installed</CardTitle>
              <TierBadge tier="installed" />
            </div>
            <CardDescription>Other components in use.</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap gap-2">
              {["Dialog", "Form", "Checkbox", "Switch", "Label", "Textarea", "Progress", "Popover", "Separator", "Avatar", "Resizable", "Toast"].map((name) => (
                <Badge key={name} variant="outline" className="text-sm">{name}</Badge>
              ))}
            </div>
            <div className="mt-4 space-y-3">
              <div className="flex items-center gap-2">
                <Checkbox id="demo" />
                <Label htmlFor="demo" className="text-sm">Enable auto-commit</Label>
              </div>
              <div className="flex items-center gap-2">
                <Switch id="demo-switch" />
                <Label htmlFor="demo-switch" className="text-sm">Dark mode</Label>
              </div>
              <Progress value={67} className="h-2" />
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

function ReplaceComponentsSection() {
  const [isCollapsibleOpen, setIsCollapsibleOpen] = useState(false)

  return (
    <div className="space-y-6">
      <div className="space-y-1">
        <h2 className="text-xl font-semibold flex items-center gap-2">
          <Replace className="size-5" />
          Replace Custom Code
        </h2>
        <p className="text-sm text-muted-foreground">
          Swap hand-rolled implementations with standard shadcn components to reduce maintenance.
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {/* Breadcrumb */}
        <Card className="border-destructive/30">
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-base">Breadcrumb</CardTitle>
              <TierBadge tier="replace" />
            </div>
            <CardDescription>
              Replace custom <code className="text-sm bg-muted px-1 py-0.5 rounded">breadcrumbs.tsx</code> (153 lines) with shadcn standard.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <Breadcrumb>
              <BreadcrumbList>
                <BreadcrumbItem>
                  <BreadcrumbLink href="#">Projects</BreadcrumbLink>
                </BreadcrumbItem>
                <BreadcrumbSeparator />
                <BreadcrumbItem>
                  <BreadcrumbLink href="#">my-workspace</BreadcrumbLink>
                </BreadcrumbItem>
                <BreadcrumbSeparator />
                <BreadcrumbItem>
                  <BreadcrumbPage>fix-auth-bug</BreadcrumbPage>
                </BreadcrumbItem>
              </BreadcrumbList>
            </Breadcrumb>
            <p className="text-sm text-muted-foreground">
              Built-in accessibility, separator customization, and ellipsis support for deep nesting.
            </p>
          </CardContent>
        </Card>

        {/* Pagination */}
        <Card className="border-destructive/30">
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-base">Pagination</CardTitle>
              <TierBadge tier="replace" />
            </div>
            <CardDescription>
              Replace manual prev/next buttons used in session and project lists.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <Pagination>
              <PaginationContent>
                <PaginationItem>
                  <PaginationPrevious href="#" />
                </PaginationItem>
                <PaginationItem>
                  <PaginationLink href="#" isActive>1</PaginationLink>
                </PaginationItem>
                <PaginationItem>
                  <PaginationLink href="#">2</PaginationLink>
                </PaginationItem>
                <PaginationItem>
                  <PaginationLink href="#">3</PaginationLink>
                </PaginationItem>
                <PaginationItem>
                  <PaginationEllipsis />
                </PaginationItem>
                <PaginationItem>
                  <PaginationNext href="#" />
                </PaginationItem>
              </PaginationContent>
            </Pagination>
            <p className="text-sm text-muted-foreground">
              Standard pagination with active state, ellipsis, and responsive labels.
            </p>
          </CardContent>
        </Card>

        {/* Collapsible */}
        <Card className="border-destructive/30">
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-base">Collapsible</CardTitle>
              <TierBadge tier="replace" />
            </div>
            <CardDescription>
              Lighter alternative to Accordion for single toggle sections (e.g. advanced settings).
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <Collapsible open={isCollapsibleOpen} onOpenChange={setIsCollapsibleOpen}>
              <CollapsibleTrigger asChild>
                <Button variant="ghost" size="sm" className="w-full justify-between">
                  Advanced Options
                  <ChevronsUpDown className="size-4" />
                </Button>
              </CollapsibleTrigger>
              <CollapsibleContent className="space-y-2 pt-2">
                <div className="rounded-md border px-3 py-2 text-sm">
                  Max tokens: 4096
                </div>
                <div className="rounded-md border px-3 py-2 text-sm">
                  Temperature: 0.7
                </div>
              </CollapsibleContent>
            </Collapsible>
            <p className="text-sm text-muted-foreground">
              No group constraints like Accordion — each section independent.
            </p>
          </CardContent>
        </Card>

        {/* DataTable concept */}
        <Card className="border-destructive/30">
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-base">DataTable Pattern</CardTitle>
              <TierBadge tier="replace" />
            </div>
            <CardDescription>
              Replace <code className="text-sm bg-muted px-1 py-0.5 rounded">SimpleDataTable</code> (186 lines) with TanStack Table + shadcn pattern for sorting, filtering, column visibility.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="flex items-center gap-2">
              <Input placeholder="Filter sessions..." className="h-8 max-w-[200px]" />
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="outline" size="sm">
                    Columns <ChevronDown className="size-3" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent>
                  <DropdownMenuItem>Name</DropdownMenuItem>
                  <DropdownMenuItem>Status</DropdownMenuItem>
                  <DropdownMenuItem>Duration</DropdownMenuItem>
                  <DropdownMenuItem>Created</DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="cursor-pointer">Name <ChevronsUpDown className="inline size-3" /></TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="text-right cursor-pointer">Duration <ChevronsUpDown className="inline size-3" /></TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow>
                  <TableCell className="font-medium">fix-auth-bug</TableCell>
                  <TableCell><Badge variant="secondary" className="bg-status-success text-status-success-foreground border-0 text-sm">Running</Badge></TableCell>
                  <TableCell className="text-right text-muted-foreground">3m 42s</TableCell>
                </TableRow>
              </TableBody>
            </Table>
            <p className="text-sm text-muted-foreground">
              Adds sorting, filtering, column visibility, and row selection. Requires TanStack Table dependency.
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

function AddComponentsSection() {


  return (
    <div className="space-y-6">
      <div className="space-y-1">
        <h2 className="text-xl font-semibold flex items-center gap-2">
          <Plus className="size-5" />
          Recommended Additions
        </h2>
        <p className="text-sm text-muted-foreground">
          New components that would improve UX without existing replacements.
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {/* Sheet */}
        <Card className="border-primary/30">
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-base">Sheet / Drawer</CardTitle>
              <div className="flex items-center gap-2">
                <Badge variant="outline" className="text-sm font-normal bg-primary/10">High Value</Badge>
                <TierBadge tier="add" />
              </div>
            </div>
            <CardDescription>
              Slide-out panels for mobile navigation, session details, and settings.
              Currently no mobile drawer story.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <Sheet>
              <SheetTrigger asChild>
                <Button variant="outline" size="sm">
                  <PanelLeft className="size-4" />
                  Open Sheet
                </Button>
              </SheetTrigger>
              <SheetContent>
                <SheetHeader>
                  <SheetTitle>Session Details</SheetTitle>
                  <SheetDescription>
                    View session configuration and metadata.
                  </SheetDescription>
                </SheetHeader>
                <div className="space-y-4 p-4">
                  <div className="space-y-1">
                    <p className="text-sm font-medium">Session</p>
                    <p className="text-sm text-muted-foreground">fix-auth-bug</p>
                  </div>
                  <Separator />
                  <div className="space-y-1">
                    <p className="text-sm font-medium">Model</p>
                    <p className="text-sm text-muted-foreground">claude-sonnet-4-6</p>
                  </div>
                  <Separator />
                  <div className="space-y-1">
                    <p className="text-sm font-medium">Status</p>
                    <Badge variant="secondary" className="bg-status-success text-status-success-foreground border-0">Running</Badge>
                  </div>
                  <Separator />
                  <div className="space-y-1">
                    <p className="text-sm font-medium">Duration</p>
                    <p className="text-sm text-muted-foreground">3m 42s</p>
                  </div>
                </div>
              </SheetContent>
            </Sheet>
            <p className="text-sm text-muted-foreground">
              Accessible slide-out panel with overlay. Supports top/right/bottom/left positioning.
            </p>
          </CardContent>
        </Card>

        {/* Command */}
        <Card className="border-primary/30">
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-base">Command Palette</CardTitle>
              <div className="flex items-center gap-2">
                <Badge variant="outline" className="text-sm font-normal bg-primary/10">High Value</Badge>
                <TierBadge tier="add" />
              </div>
            </div>
            <CardDescription>
              Searchable command palette for sessions, actions, and navigation.
              Would enhance the AutocompletePopover in chat.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <Command className="rounded-lg border">
              <CommandInput placeholder="Search sessions, actions..." />
              <CommandList>
                <CommandEmpty>No results found.</CommandEmpty>
                <CommandGroup heading="Sessions">
                  <CommandItem>
                    <Terminal className="size-4" />
                    <span>fix-auth-bug</span>
                    <CommandShortcut>Running</CommandShortcut>
                  </CommandItem>
                  <CommandItem>
                    <Terminal className="size-4" />
                    <span>add-rbac-support</span>
                    <CommandShortcut>Completed</CommandShortcut>
                  </CommandItem>
                </CommandGroup>
                <CommandSeparator />
                <CommandGroup heading="Actions">
                  <CommandItem>
                    <Plus className="size-4" />
                    <span>New Session</span>
                    <CommandShortcut>N</CommandShortcut>
                  </CommandItem>
                  <CommandItem>
                    <Settings className="size-4" />
                    <span>Settings</span>
                    <CommandShortcut>,</CommandShortcut>
                  </CommandItem>
                </CommandGroup>
              </CommandList>
            </Command>
            <p className="text-sm text-muted-foreground">
              Keyboard-navigable, filterable, groupable. Built on cmdk. Supports dialog mode for <kbd className="border bg-muted px-1 rounded text-sm">K</kbd> shortcut.
            </p>
          </CardContent>
        </Card>

        {/* HoverCard */}
        <Card className="border-primary/30">
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-base">HoverCard</CardTitle>
              <div className="flex items-center gap-2">
                <Badge variant="outline" className="text-sm font-normal">Medium Value</Badge>
                <TierBadge tier="add" />
              </div>
            </div>
            <CardDescription>
              Preview session details on hover in session tables without opening a modal.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <HoverCard>
              <HoverCardTrigger asChild>
                <Button variant="link" className="h-auto p-0 text-sm">
                  fix-auth-bug
                </Button>
              </HoverCardTrigger>
              <HoverCardContent className="w-72">
                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <h4 className="text-sm font-semibold">fix-auth-bug</h4>
                    <Badge variant="secondary" className="bg-status-success text-status-success-foreground border-0 text-sm">Running</Badge>
                  </div>
                  <p className="text-sm text-muted-foreground">
                    Fixing authentication token refresh logic in the middleware handler.
                  </p>
                  <div className="flex items-center gap-4 text-sm text-muted-foreground">
                    <span className="flex items-center gap-1"><Clock className="size-3" /> 3m 42s</span>
                    <span className="flex items-center gap-1"><Package className="size-3" /> claude-sonnet-4-6</span>
                  </div>
                </div>
              </HoverCardContent>
            </HoverCard>
            <p className="text-sm text-muted-foreground">
              Hover to see details. Great for information density without clicks.
            </p>
          </CardContent>
        </Card>

        {/* Sonner */}
        <Card className="border-primary/30">
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-base">Sonner (Toast)</CardTitle>
              <div className="flex items-center gap-2">
                <Badge variant="outline" className="text-sm font-normal">Medium Value</Badge>
                <TierBadge tier="add" />
              </div>
            </div>
            <CardDescription>
              Better toast UX than current Radix Toast — smooth animations, stacking, promise toasts.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="rounded-lg border p-3 space-y-2">
              <div className="flex items-start gap-3">
                <CheckCircle2 className="size-4 text-status-success-foreground mt-0.5 shrink-0" />
                <div className="space-y-1">
                  <p className="text-sm font-medium">Session created</p>
                  <p className="text-sm text-muted-foreground">fix-auth-bug is now running</p>
                </div>
              </div>
            </div>
            <p className="text-sm text-muted-foreground">
              Drop-in replacement. Supports success/error/loading states, stacking, and promise-based toasts.
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

function SkipComponentsSection() {
  const skipComponents = [
    { name: "Calendar / Date Picker", reason: "No date input needs currently" },
    { name: "Context Menu", reason: "Right-click menus add discoverability problems" },
    { name: "Menubar", reason: "Not a desktop app — unnecessary complexity" },
    { name: "NavigationMenu", reason: "Current nav works fine for the page count" },
    { name: "Input OTP", reason: "No OTP flows in the app" },
    { name: "Slider", reason: "No range input needs" },
    { name: "Carousel", reason: "Not applicable to the app's UI patterns" },
    { name: "Chart", reason: "No charting needs yet — use a dedicated library if needed" },
  ]

  return (
    <div className="space-y-6">
      <div className="space-y-1">
        <h2 className="text-xl font-semibold flex items-center gap-2">
          <X className="size-5" />
          Skip
        </h2>
        <p className="text-sm text-muted-foreground">
          Components that don&apos;t match current needs.
        </p>
      </div>

      <Card>
        <CardContent className="pt-6">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Component</TableHead>
                <TableHead>Why Skip</TableHead>
                <TableHead className="w-24"></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {skipComponents.map((comp) => (
                <TableRow key={comp.name}>
                  <TableCell className="font-medium">{comp.name}</TableCell>
                  <TableCell className="text-muted-foreground">{comp.reason}</TableCell>
                  <TableCell>
                    <TierBadge tier="skip" />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  )
}

function ObservationsSection() {
  return (
    <div className="space-y-6">
      <div className="space-y-1">
        <h2 className="text-xl font-semibold flex items-center gap-2">
          <Zap className="size-5" />
          Other Observations
        </h2>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Integration Card Duplication</CardTitle>
            <CardDescription>Not a shadcn issue, but a big cleanup opportunity.</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-2 text-sm">
              <p className="text-muted-foreground">
                GitHub, GitLab, Jira, and Google Drive connection cards are each ~290 lines with heavy duplication.
              </p>
              <div className="flex flex-wrap gap-1">
                <Badge variant="outline" className="text-sm">github-connection-card.tsx</Badge>
                <Badge variant="outline" className="text-sm">gitlab-connection-card.tsx</Badge>
                <Badge variant="outline" className="text-sm">jira-connection-card.tsx</Badge>
                <Badge variant="outline" className="text-sm">google-drive-connection-card.tsx</Badge>
              </div>
              <p className="text-muted-foreground">
                Extract a shared <code className="bg-muted px-1 py-0.5 rounded text-sm">IntegrationCard</code> wrapper to cut ~800 lines.
              </p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">Custom Message Components</CardTitle>
            <CardDescription>Domain-specific — shadcn can&apos;t help here, and that&apos;s fine.</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-2 text-sm text-muted-foreground">
              <div className="flex flex-wrap gap-1">
                <Badge variant="outline" className="text-sm">message.tsx (272 lines)</Badge>
                <Badge variant="outline" className="text-sm">tool-message.tsx (736 lines)</Badge>
                <Badge variant="outline" className="text-sm">stream-message.tsx</Badge>
                <Badge variant="outline" className="text-sm">thinking-message.tsx</Badge>
                <Badge variant="outline" className="text-sm">system-message.tsx</Badge>
              </div>
              <p>
                These are highly specialized chat components. Keep as-is.
              </p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">Form Handling</CardTitle>
            <CardDescription>Already solid — no changes needed.</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-2 text-sm text-muted-foreground">
              <p>react-hook-form + zod validation + shadcn Form components are well integrated. 18+ form-heavy dialogs all use this pattern consistently.</p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">Design System</CardTitle>
            <CardDescription>Strong foundation already in place.</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-2 text-sm text-muted-foreground">
              <p>OKLch color space, CSS variables, semantic status colors, role-based colors, dark mode, Geist fonts. The design system is well-structured.</p>
              <div className="flex gap-2 mt-2">
                <div className="size-6 rounded bg-primary" title="Primary" />
                <div className="size-6 rounded bg-secondary" title="Secondary" />
                <div className="size-6 rounded bg-destructive" title="Destructive" />
                <div className="size-6 rounded bg-status-success" title="Success" />
                <div className="size-6 rounded bg-status-error" title="Error" />
                <div className="size-6 rounded bg-status-warning" title="Warning" />
                <div className="size-6 rounded bg-status-info" title="Info" />
                <div className="size-6 rounded bg-muted" title="Muted" />
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Summary stats
// ---------------------------------------------------------------------------
function SummaryStats() {
  return (
    <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
      <Card>
        <CardContent className="pt-6 text-center">
          <div className="text-3xl font-bold">30</div>
          <div className="text-sm text-muted-foreground">Installed</div>
        </CardContent>
      </Card>
      <Card className="border-destructive/30">
        <CardContent className="pt-6 text-center">
          <div className="text-3xl font-bold text-destructive">4</div>
          <div className="text-sm text-muted-foreground">Replace</div>
        </CardContent>
      </Card>
      <Card className="border-primary/30">
        <CardContent className="pt-6 text-center">
          <div className="text-3xl font-bold text-primary">4</div>
          <div className="text-sm text-muted-foreground">Add</div>
        </CardContent>
      </Card>
      <Card>
        <CardContent className="pt-6 text-center">
          <div className="text-3xl font-bold text-muted-foreground">8</div>
          <div className="text-sm text-muted-foreground">Skip</div>
        </CardContent>
      </Card>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Main page
// ---------------------------------------------------------------------------
export default function EvaluatePage() {
  return (
    <TooltipProvider>
      <div className="min-h-screen bg-background">
        <div className="mx-auto max-w-6xl px-4 py-8 space-y-10">
          {/* Header */}
          <div className="space-y-2">
            <h1 className="text-3xl font-bold tracking-tight">
              shadcn/ui Component Audit
            </h1>
            <p className="text-muted-foreground">
              Evaluation of shadcn components for the Ambient Code Platform frontend.
              <br />
              <span className="text-sm">
                New York style &middot; neutral base &middot; OKLch colors &middot; Lucide icons &middot; Tailwind v4
              </span>
            </p>
          </div>

          <SummaryStats />

          <Tabs defaultValue="installed" className="space-y-6">
            <TabsList>
              <TabsTrigger value="installed">
                Installed <Badge variant="outline" className="ml-1.5 text-sm h-5">30</Badge>
              </TabsTrigger>
              <TabsTrigger value="replace">
                Replace <Badge variant="destructive" className="ml-1.5 text-sm h-5">4</Badge>
              </TabsTrigger>
              <TabsTrigger value="add">
                Add <Badge className="ml-1.5 text-sm h-5">4</Badge>
              </TabsTrigger>
              <TabsTrigger value="skip">
                Skip <Badge variant="secondary" className="ml-1.5 text-sm h-5">8</Badge>
              </TabsTrigger>
              <TabsTrigger value="observations">
                Observations
              </TabsTrigger>
            </TabsList>

            <TabsContent value="installed">
              <InstalledComponentsSection />
            </TabsContent>
            <TabsContent value="replace">
              <ReplaceComponentsSection />
            </TabsContent>
            <TabsContent value="add">
              <AddComponentsSection />
            </TabsContent>
            <TabsContent value="skip">
              <SkipComponentsSection />
            </TabsContent>
            <TabsContent value="observations">
              <ObservationsSection />
            </TabsContent>
          </Tabs>
        </div>
      </div>
    </TooltipProvider>
  )
}
