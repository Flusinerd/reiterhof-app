import { GAITS } from "./types.ts";
import type { Gait, GaitWindow } from "./types.ts";

export type GaitShare = { minutes: number; percent: number };

export type GaitShares = {
  totalMinutes: number;
  shares: Record<Gait, GaitShare>;
};

export type ShareOptions = {
  /** Prefer the user-corrected label over the prediction (default true). */
  useLabels?: boolean;
  /** Leave halt out of the total, so the percentages describe the riding only. */
  excludeHalt?: boolean;
};

/**
 * Time per gait. Each window counts for its `weightMs` (the hop length), so
 * overlapping windows are not counted twice. Percentages sum to 100 (or all
 * are 0 for an empty input).
 */
export function gaitShares(windows: readonly GaitWindow[], options: ShareOptions = {}): GaitShares {
  const useLabels = options.useLabels ?? true;
  const ms: Record<Gait, number> = { halt: 0, walk: 0, trot: 0, canter: 0 };
  for (const w of windows) {
    const g = useLabels && w.label ? w.label : w.gait;
    if (options.excludeHalt && g === "halt") continue;
    ms[g] += w.weightMs;
  }
  const total = GAITS.reduce((s, g) => s + ms[g], 0);
  const shares = {} as Record<Gait, GaitShare>;
  for (const g of GAITS) {
    shares[g] = {
      minutes: ms[g] / 60000,
      percent: total > 0 ? (ms[g] / total) * 100 : 0,
    };
  }
  return { totalMinutes: total / 60000, shares };
}
