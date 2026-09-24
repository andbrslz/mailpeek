export const prefersReducedMotion = () =>
  window.matchMedia("(prefers-reduced-motion: reduce)").matches;

export const fullClip = "inset(0px 0px 0px 0px round 0px)";

export function clipTo(el: HTMLElement | null): string | null {
  const r = el?.getBoundingClientRect();
  if (!r || r.width === 0 || r.height === 0) return null;
  const right = window.innerWidth - r.right;
  const bottom = window.innerHeight - r.bottom;
  return `inset(${r.top}px ${right}px ${bottom}px ${r.left}px round 12px)`;
}
