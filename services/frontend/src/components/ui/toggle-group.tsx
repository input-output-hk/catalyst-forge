import * as React from "react"
import * as ToggleGroupPrimitive from "@radix-ui/react-toggle-group"

import { cn } from "@/lib/utils"

// Keep API parity so existing usages that pass `size`/`variant` don't error
// These props are intentionally swallowed and unused here
 type SizeVariant = "default" | "sm" | "lg"
 type Variant = "default" | "outline"

const ToggleGroup = React.forwardRef<
  React.ElementRef<typeof ToggleGroupPrimitive.Root>,
  React.ComponentPropsWithoutRef<typeof ToggleGroupPrimitive.Root> & {
    size?: SizeVariant
    variant?: Variant
  }
>(({ className, size, variant, children, ...props }, ref) => (
  <ToggleGroupPrimitive.Root
    ref={ref}
    className={cn("flex items-center justify-center gap-1", className)}
    {...props}
  >
    {children}
  </ToggleGroupPrimitive.Root>
))

ToggleGroup.displayName = ToggleGroupPrimitive.Root.displayName

const ToggleGroupItem = React.forwardRef<
  React.ElementRef<typeof ToggleGroupPrimitive.Item>,
  React.ComponentPropsWithoutRef<typeof ToggleGroupPrimitive.Item> & {
    size?: SizeVariant
    variant?: Variant
  }
>(({ className, ...props }, ref) => (
  <ToggleGroupPrimitive.Item
    ref={ref}
    className={cn(
      // Base styles
      "inline-flex items-center justify-center rounded-md text-sm font-medium ring-offset-background transition-colors",
      "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
      "disabled:pointer-events-none disabled:opacity-50",
      // Sizing/base appearance (defaults; can be overridden)
      "h-10 px-3 border border-transparent text-muted-foreground",
      // Hover and default pressed styles (can be overridden by `className`)
      "hover:bg-accent hover:text-foreground",
      "data-[state=on]:bg-accent data-[state=on]:text-accent-foreground",
      // Your overrides must come last so they win
      className
    )}
    {...props}
  />
))

ToggleGroupItem.displayName = ToggleGroupPrimitive.Item.displayName

export { ToggleGroup, ToggleGroupItem }
