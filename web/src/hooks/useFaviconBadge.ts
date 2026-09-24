import { useEffect } from "react";

const iconUrl = "/favicon.svg";
const size = 64;
let icon: Promise<HTMLImageElement> | undefined;

function loadIcon(): Promise<HTMLImageElement> {
  icon ??= new Promise((resolve, reject) => {
    const img = new Image();
    img.onload = () => resolve(img);
    img.onerror = reject;
    img.src = iconUrl;
  });
  return icon;
}

function iconLink(): HTMLLinkElement {
  let link = document.querySelector<HTMLLinkElement>('link[rel="icon"]');
  if (!link) {
    link = document.createElement("link");
    link.rel = "icon";
    document.head.appendChild(link);
  }
  return link;
}

async function drawBadge(count: number): Promise<string> {
  const img = await loadIcon();
  const canvas = document.createElement("canvas");
  canvas.width = canvas.height = size;
  const ctx = canvas.getContext("2d");
  if (!ctx) throw new Error("canvas unavailable");
  ctx.drawImage(img, 0, 0, size, size);

  const r = 19;
  const x = size - r;
  const y = r;
  ctx.beginPath();
  ctx.arc(x, y, r, 0, Math.PI * 2);
  ctx.fillStyle = "#ef4444";
  ctx.fill();
  ctx.lineWidth = 4;
  ctx.strokeStyle = "#ffffff";
  ctx.stroke();

  ctx.fillStyle = "#ffffff";
  ctx.font = `bold ${count > 9 ? 20 : 26}px system-ui, sans-serif`;
  ctx.textAlign = "center";
  ctx.textBaseline = "middle";
  ctx.fillText(count > 9 ? "9+" : String(count), x, y + 1);
  return canvas.toDataURL("image/png");
}

export function useFaviconBadge(count: number, title = "Mailpeek") {
  useEffect(() => {
    document.title = count > 0 ? `(${count}) ${title}` : title;
    const link = iconLink();
    if (count === 0) {
      link.type = "image/svg+xml";
      link.href = iconUrl;
      return;
    }
    let cancelled = false;
    drawBadge(count)
      .then((href) => {
        if (cancelled) return;
        link.type = "image/png";
        link.href = href;
      })
      .catch(() => undefined);
    return () => {
      cancelled = true;
    };
  }, [count, title]);
}
