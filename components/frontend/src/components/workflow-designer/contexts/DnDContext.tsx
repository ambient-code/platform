"use client";

import { createContext, useState, useContext, type ReactNode } from "react";

type DnDContextValue = {
  type: string | null;
  setType: (type: string | null) => void;
};

const DnDContext = createContext<DnDContextValue>({
  type: null,
  setType: () => {},
});

export function DnDProvider({ children }: { children: ReactNode }) {
  const [type, setType] = useState<string | null>(null);

  return (
    <DnDContext.Provider value={{ type, setType }}>
      {children}
    </DnDContext.Provider>
  );
}

export function useDnD() {
  return useContext(DnDContext);
}
