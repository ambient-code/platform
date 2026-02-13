"use client";

import { Handle, Position, type HandleType } from "@xyflow/react";
import { type ReactNode } from "react";
import { Card } from "@/components/ui/card";
import { cn } from "@/lib/utils";

type HandleConfig = {
  type: HandleType;
  position: Position;
};

type BaseNodeProps = {
  label: string;
  color: "emerald" | "blue" | "orange" | "purple" | "yellow";
  icon: ReactNode;
  selected: boolean;
  children?: ReactNode;
  handles: HandleConfig[];
};

const colorMap = {
  emerald: {
    border: "border-emerald-600",
    selectedBorder: "border-emerald-400",
    headerBg: "bg-emerald-950/60",
    iconColor: "text-emerald-400",
  },
  blue: {
    border: "border-blue-600",
    selectedBorder: "border-blue-400",
    headerBg: "bg-blue-950/60",
    iconColor: "text-blue-400",
  },
  orange: {
    border: "border-orange-600",
    selectedBorder: "border-orange-400",
    headerBg: "bg-orange-950/60",
    iconColor: "text-orange-400",
  },
  purple: {
    border: "border-purple-600",
    selectedBorder: "border-purple-400",
    headerBg: "bg-purple-950/60",
    iconColor: "text-purple-400",
  },
  yellow: {
    border: "border-yellow-600",
    selectedBorder: "border-yellow-400",
    headerBg: "bg-yellow-950/60",
    iconColor: "text-yellow-400",
  },
};

export function BaseNode({
  label,
  color,
  icon,
  selected,
  children,
  handles,
}: BaseNodeProps) {
  const colors = colorMap[color];

  return (
    <Card
      className={cn(
        "w-[260px] border-2 shadow-md transition-all duration-200",
        selected ? colors.selectedBorder : colors.border,
        selected && "shadow-lg"
      )}
    >
      <div
        className={cn(
          "flex items-center gap-2 px-3 py-2 rounded-t-lg",
          colors.headerBg
        )}
      >
        <span className={colors.iconColor}>{icon}</span>
        <span className="text-sm font-medium text-foreground truncate">
          {label}
        </span>
      </div>

      {children && <div className="px-3 py-2 text-xs">{children}</div>}

      {handles.map((handle, i) => (
        <Handle
          key={`${handle.type}-${handle.position}-${i}`}
          type={handle.type}
          position={handle.position}
          className={cn(
            "!w-3 !h-3 !border-2 !border-background",
            handle.type === "source" ? "!bg-emerald-500" : "!bg-blue-500"
          )}
        />
      ))}
    </Card>
  );
}
