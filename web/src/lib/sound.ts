let ctx: AudioContext | undefined;
let lastPlayed = 0;

export function unlockAudio() {
  try {
    ctx ??= new AudioContext();
    if (ctx.state === "suspended") void ctx.resume();
  } catch {}
}

export function playChime() {
  const now = Date.now();
  if (!ctx || ctx.state !== "running" || now - lastPlayed < 1000) return;
  lastPlayed = now;

  const start = ctx.currentTime;
  const notes: [frequency: number, delay: number][] = [
    [659.25, 0],
    [880, 0.09],
  ];
  for (const [frequency, delay] of notes) {
    const osc = ctx.createOscillator();
    const gain = ctx.createGain();
    const t = start + delay;
    osc.type = "sine";
    osc.frequency.value = frequency;
    gain.gain.setValueAtTime(0.0001, t);
    gain.gain.exponentialRampToValueAtTime(0.12, t + 0.01);
    gain.gain.exponentialRampToValueAtTime(0.0001, t + 0.35);
    osc.connect(gain).connect(ctx.destination);
    osc.start(t);
    osc.stop(t + 0.4);
  }
}
