import { useEffect } from "react";
import { unlockAudio } from "../lib/sound";
import { useStoredToggle } from "./useStoredToggle";

export function useSoundSetting() {
  const [enabled, toggle] = useStoredToggle("mailpeek:sound", true);

  useEffect(() => {
    if (!enabled) return;
    const unlock = () => unlockAudio();
    window.addEventListener("pointerdown", unlock);
    window.addEventListener("keydown", unlock);
    return () => {
      window.removeEventListener("pointerdown", unlock);
      window.removeEventListener("keydown", unlock);
    };
  }, [enabled]);

  return { enabled, toggle };
}
