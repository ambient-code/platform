"use client"

import * as React from "react"
import { createPortal } from "react-dom"
import { cn } from "@/lib/utils"

interface PopoverContextType {
  open: boolean
  setOpen: (open: boolean) => void
  triggerRef: React.RefObject<HTMLElement | null>
}

const PopoverContext = React.createContext<PopoverContextType | undefined>(undefined)

function usePopoverContext() {
  const context = React.useContext(PopoverContext)
  if (!context) {
    throw new Error("Popover components must be used within Popover")
  }
  return context
}

interface PopoverProps {
  children: React.ReactNode
  open?: boolean
  onOpenChange?: (open: boolean) => void
}

export function Popover({ children, open: controlledOpen, onOpenChange }: PopoverProps) {
  const [uncontrolledOpen, setUncontrolledOpen] = React.useState(false)
  const triggerRef = React.useRef<HTMLElement>(null)
  
  const open = controlledOpen !== undefined ? controlledOpen : uncontrolledOpen
  const setOpen = onOpenChange || setUncontrolledOpen

  return (
    <PopoverContext.Provider value={{ open, setOpen, triggerRef }}>
      <div className="relative inline">
        {children}
      </div>
    </PopoverContext.Provider>
  )
}

interface PopoverTriggerProps {
  children: React.ReactNode
  asChild?: boolean
  className?: string
}

export function PopoverTrigger({ children, asChild, className }: PopoverTriggerProps) {
  const { open, setOpen, triggerRef } = usePopoverContext()

  const handleClick = (e: React.MouseEvent) => {
    e.stopPropagation()
    setOpen(!open)
  }

  if (asChild && React.isValidElement(children)) {
    return React.cloneElement(children as React.ReactElement<React.HTMLAttributes<HTMLElement> & { ref?: React.Ref<HTMLElement> }>, {
      ref: triggerRef as React.Ref<HTMLElement>,
      onClick: handleClick,
      className: cn((children as React.ReactElement<React.HTMLAttributes<HTMLElement>>).props?.className, className),
    })
  }

  return (
    <button ref={triggerRef as React.RefObject<HTMLButtonElement>} onClick={handleClick} className={className}>
      {children}
    </button>
  )
}

interface PopoverContentProps {
  children: React.ReactNode
  className?: string
  align?: "start" | "center" | "end"
  side?: "top" | "right" | "bottom" | "left"
  sideOffset?: number
}

export function PopoverContent({ 
  children, 
  className, 
  align = "center",
  side = "bottom",
  sideOffset = 0
}: PopoverContentProps) {
  const { open, setOpen, triggerRef } = usePopoverContext()
  const contentRef = React.useRef<HTMLDivElement>(null)
  const [position, setPosition] = React.useState<{ top: number; left: number }>({ top: 0, left: 0 })
  const [mounted, setMounted] = React.useState(false)

  React.useEffect(() => {
    setMounted(true)
  }, [])

  React.useEffect(() => {
    if (!open || !triggerRef.current) return

    const updatePosition = () => {
      const triggerRect = triggerRef.current!.getBoundingClientRect()
      let top = 0
      let left = 0

      // Calculate vertical position based on side
      if (side === "bottom") {
        top = triggerRect.bottom + sideOffset
      } else if (side === "top") {
        top = triggerRect.top - sideOffset
      } else if (side === "left" || side === "right") {
        top = triggerRect.top
      }

      // Calculate horizontal position based on align
      if (align === "start") {
        left = triggerRect.left
      } else if (align === "center") {
        left = triggerRect.left + triggerRect.width / 2
      } else if (align === "end") {
        left = triggerRect.right
      }

      // Adjust for side positioning
      if (side === "left") {
        left = triggerRect.left - sideOffset
      } else if (side === "right") {
        left = triggerRect.right + sideOffset
      }

      setPosition({ top, left })
    }

    updatePosition()
    window.addEventListener("scroll", updatePosition, true)
    window.addEventListener("resize", updatePosition)

    return () => {
      window.removeEventListener("scroll", updatePosition, true)
      window.removeEventListener("resize", updatePosition)
    }
  }, [open, triggerRef, side, align, sideOffset])

  React.useEffect(() => {
    if (!open) return

    const handleClickOutside = (e: MouseEvent) => {
      if (contentRef.current && !contentRef.current.contains(e.target as Node) &&
          triggerRef.current && !triggerRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }

    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        setOpen(false)
      }
    }

    document.addEventListener("mousedown", handleClickOutside)
    document.addEventListener("keydown", handleEscape)

    return () => {
      document.removeEventListener("mousedown", handleClickOutside)
      document.removeEventListener("keydown", handleEscape)
    }
  }, [open, setOpen, triggerRef])

  if (!open || !mounted) return null

  const getTransformOrigin = () => {
    if (side === "bottom") return "top"
    if (side === "top") return "bottom"
    if (side === "left") return "right"
    if (side === "right") return "left"
    return "top"
  }

  const content = (
    <div
      ref={contentRef}
      style={{
        position: "fixed",
        top: `${position.top}px`,
        left: `${position.left}px`,
        transformOrigin: getTransformOrigin(),
        ...(align === "center" && { transform: "translateX(-50%)" }),
        ...(side === "top" && contentRef.current && { transform: `translateY(-${contentRef.current.offsetHeight}px)` }),
        ...(side === "left" && contentRef.current && { transform: `translateX(-${contentRef.current.offsetWidth}px)` }),
      }}
      className={cn(
        "z-50 min-w-[200px] rounded-md border bg-popover p-3 text-popover-foreground shadow-md animate-in fade-in-0 zoom-in-95",
        className
      )}
    >
      {children}
    </div>
  )

  return createPortal(content, document.body)
}

