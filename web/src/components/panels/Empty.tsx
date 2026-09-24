import type { ReactNode } from "react";

export function Empty({ children }: { children: ReactNode }) {
  return <div className="p-8 text-center text-sm text-zinc-500">{children}</div>;
}
