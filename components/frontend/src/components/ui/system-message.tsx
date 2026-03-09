import React from "react";
import { cn } from "@/lib/utils";

type SystemMessageData = {
  message?: string;
  [key: string]: unknown;
};

export type SystemMessageProps = {
  subtype?: string;
  data: SystemMessageData;
  className?: string;
  borderless?: boolean;
};

export const SystemMessage: React.FC<SystemMessageProps> = ({ data, className }) => {
  const text: string = typeof (data?.message) === 'string' ? data.message : (typeof data === 'string' ? data : JSON.stringify(data ?? {}, null, 2));

  return (
    <div className={cn("my-1 px-2", className)}>
      <p className="text-sm text-muted-foreground/60 italic">
        {text}
      </p>
    </div>
  );
};

export default SystemMessage;
