import { useCallback, useEffect, useRef, useState } from "react";

export function useConfirmAction(action: () => void, ms = 3000) {
  const [armed, setArmed] = useState(false);
  const timer = useRef<number | undefined>(undefined);
  useEffect(() => () => window.clearTimeout(timer.current), []);

  const trigger = useCallback(() => {
    window.clearTimeout(timer.current);
    if (armed) {
      setArmed(false);
      action();
      return;
    }
    setArmed(true);
    timer.current = window.setTimeout(() => setArmed(false), ms);
  }, [armed, action, ms]);

  return { armed, trigger };
}
