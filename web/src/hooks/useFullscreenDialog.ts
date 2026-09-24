import { useEffect, useLayoutEffect, useRef, type RefObject } from "react";
import { clipTo, fullClip, prefersReducedMotion } from "../lib/motion";

export function useFullscreenDialog({
  origin,
  closing,
  onClose,
  onClosed,
}: {
  origin: RefObject<HTMLElement | null>;
  closing: boolean;
  onClose: () => void;
  onClosed: () => void;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const closeButton = useRef<HTMLButtonElement>(null);
  const callbacks = useRef({ onClose, onClosed });
  useEffect(() => {
    callbacks.current = { onClose, onClosed };
  });

  useLayoutEffect(() => {
    const el = dialog.current;
    if (!el) return;
    const returnFocus = document.activeElement as HTMLElement | null;
    el.showModal();
    closeButton.current?.focus();
    const from = prefersReducedMotion() ? null : clipTo(origin.current);
    if (from) {
      el.animate([{ clipPath: from }, { clipPath: fullClip }], {
        duration: 320,
        easing: "cubic-bezier(0.2, 0.8, 0.2, 1)",
      });
    }
    const onCancel = (e: Event) => {
      e.preventDefault();
      callbacks.current.onClose();
    };
    el.addEventListener("cancel", onCancel);
    return () => {
      el.removeEventListener("cancel", onCancel);
      if (el.open) el.close();
      returnFocus?.focus();
    };
  }, [origin]);

  useEffect(() => {
    if (!closing) return;
    const el = dialog.current;
    const done = () => callbacks.current.onClosed();
    const to = prefersReducedMotion() ? null : clipTo(origin.current);
    if (!el || !to) {
      done();
      return;
    }
    const animation = el.animate([{ clipPath: fullClip }, { clipPath: to }], {
      duration: 260,
      easing: "cubic-bezier(0.4, 0, 1, 1)",
      fill: "forwards",
    });
    const timer = window.setTimeout(done, 700);
    animation.onfinish = () => {
      window.clearTimeout(timer);
      done();
    };
    return () => {
      animation.onfinish = null;
      window.clearTimeout(timer);
    };
  }, [closing, origin]);

  return { dialog, closeButton };
}
