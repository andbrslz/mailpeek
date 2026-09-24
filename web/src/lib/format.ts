import type { Address } from "../api";

export function formatBytes(n: number, locale?: string): string {
  if (n < 1024) return `${n} B`;
  const units = ["KB", "MB", "GB"];
  let value = n / 1024;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit++;
  }
  const digits = value < 10 ? 1 : 0;
  const number = new Intl.NumberFormat(locale, {
    minimumFractionDigits: digits,
    maximumFractionDigits: digits,
  }).format(value);
  return `${number} ${units[unit]}`;
}

export function formatListTime(iso: string, locale?: string, now = new Date()): string {
  const d = new Date(iso);
  const sameDay = d.toDateString() === now.toDateString();
  return sameDay
    ? d.toLocaleTimeString(locale, { hour: "2-digit", minute: "2-digit" })
    : d.toLocaleDateString(locale, { month: "short", day: "numeric" });
}

export function formatDateTime(iso: string, locale?: string): string {
  return new Date(iso).toLocaleString(locale, {
    dateStyle: "medium",
    timeStyle: "medium",
  });
}

export const addressLabel = (a: Address, unknown: string) => a.name || a.address || unknown;

export const addressFull = (a: Address) => (a.name ? `${a.name} <${a.address}>` : a.address);

export const addressList = (list: Address[]) => list.map(addressFull).join(", ");

export function fileTypeLabel(filename: string, contentType: string, fallback: string): string {
  const ext = filename.includes(".") ? filename.split(".").pop() : "";
  if (ext && ext.length <= 5) return ext.toUpperCase();
  const subtype = contentType.split("/")[1] ?? fallback;
  return subtype.replace(/^x-/, "").slice(0, 8).toUpperCase();
}
