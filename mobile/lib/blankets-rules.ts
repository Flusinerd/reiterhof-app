// Pure helpers for editing blanket rules (JAN-30): drafts as the form holds them, parsing
// with German error texts, reordering. No React Native imports (unit-tested with node --test).

import type { Rule } from "./blankets.ts";

export type RainChoice = "any" | "rain" | "dry";

/** One rule while it is edited: numbers are still text so that "" and "-" can be typed. */
export type RuleDraft = {
  /** Stable key for the list, not sent to the server. */
  key: string;
  temp_min: string;
  temp_max: string;
  rain: RainChoice;
  blanket_id: string | null;
  note: string;
};

export type RuleValue = {
  temp_min: number | null;
  temp_max: number | null;
  rain: boolean | null;
  blanket_id: string | null;
  note: string;
};

export const MAX_RULES = 30;
export const MAX_NOTE = 200;
export const TEMP_LIMIT = 60;

let counter = 0;
function nextKey(): string {
  counter += 1;
  return `rule-${counter}`;
}

function numberText(v: number | null): string {
  return v === null ? "" : String(v).replace(".", ",");
}

export function toDraft(rule: Pick<Rule, "temp_min" | "temp_max" | "rain" | "blanket_id" | "note">): RuleDraft {
  return {
    key: nextKey(),
    temp_min: numberText(rule.temp_min),
    temp_max: numberText(rule.temp_max),
    rain: rule.rain === true ? "rain" : rule.rain === false ? "dry" : "any",
    blanket_id: rule.blanket_id,
    note: rule.note,
  };
}

export function emptyDraft(): RuleDraft {
  return { key: nextKey(), temp_min: "", temp_max: "", rain: "any", blanket_id: null, note: "" };
}

/** Parses "3", "-2,5" or "" (= open bound). Returns undefined for anything else. */
export function parseTemp(text: string): number | null | undefined {
  const t = text.trim().replace(",", ".");
  if (t === "") return null;
  if (!/^-?\d+(\.\d+)?$/.test(t)) return undefined;
  const n = Number(t);
  return Number.isFinite(n) ? n : undefined;
}

export type Parsed = { ok: true; value: RuleValue } | { ok: false; error: string };

/** Validates one draft the way the server does. `position` is 1-based, used in the error text. */
export function parseDraft(draft: RuleDraft, position: number): Parsed {
  const label = `Regel ${position}`;
  const min = parseTemp(draft.temp_min);
  const max = parseTemp(draft.temp_max);
  if (min === undefined || max === undefined) {
    return { ok: false, error: `${label}: Bitte gib die Temperatur als Zahl an, zum Beispiel 5 oder -2,5.` };
  }
  if ((min !== null && Math.abs(min) > TEMP_LIMIT) || (max !== null && Math.abs(max) > TEMP_LIMIT)) {
    return { ok: false, error: `${label}: Die Temperatur muss zwischen -${TEMP_LIMIT} und ${TEMP_LIMIT} °C liegen.` };
  }
  if (min !== null && max !== null && min >= max) {
    return { ok: false, error: `${label}: „Ab“ muss kleiner sein als „Unter“.` };
  }
  const note = draft.note.trim();
  if (note.length > MAX_NOTE) {
    return { ok: false, error: `${label}: Der Hinweis darf höchstens ${MAX_NOTE} Zeichen lang sein.` };
  }
  return {
    ok: true,
    value: {
      temp_min: min,
      temp_max: max,
      rain: draft.rain === "rain" ? true : draft.rain === "dry" ? false : null,
      blanket_id: draft.blanket_id,
      note,
    },
  };
}

/** Parses all drafts; the first problem wins. */
export function parseDrafts(drafts: readonly RuleDraft[]): { ok: true; value: RuleValue[] } | { ok: false; error: string } {
  if (drafts.length > MAX_RULES) return { ok: false, error: `Es sind höchstens ${MAX_RULES} Regeln möglich.` };
  const out: RuleValue[] = [];
  for (let i = 0; i < drafts.length; i++) {
    const parsed = parseDraft(drafts[i]!, i + 1);
    if (!parsed.ok) return parsed;
    out.push(parsed.value);
  }
  return { ok: true, value: out };
}

/** Moves the item at `index` by `delta` positions (-1 up, +1 down); out of range returns the list unchanged. */
export function moveItem<T>(list: readonly T[], index: number, delta: -1 | 1): T[] {
  const to = index + delta;
  const copy = [...list];
  if (index < 0 || index >= copy.length || to < 0 || to >= copy.length) return copy;
  const [item] = copy.splice(index, 1);
  copy.splice(to, 0, item!);
  return copy;
}

/**
 * Hint about rules that can never apply because an earlier rule already covers every
 * forecast they would match ("Regel 3 wird nie erreicht"). Returns the 1-based positions.
 */
export function unreachableRules(values: readonly RuleValue[]): number[] {
  const out: number[] = [];
  values.forEach((v, i) => {
    const shadowed = values.slice(0, i).some((e) => covers(e, v));
    if (shadowed) out.push(i + 1);
  });
  return out;
}

/** True when every forecast matching `inner` also matches `outer`. */
function covers(outer: RuleValue, inner: RuleValue): boolean {
  const lo = (v: number | null) => (v === null ? -Infinity : v);
  const hi = (v: number | null) => (v === null ? Infinity : v);
  if (lo(outer.temp_min) > lo(inner.temp_min)) return false;
  if (hi(outer.temp_max) < hi(inner.temp_max)) return false;
  if (outer.rain !== null && outer.rain !== inner.rain) return false;
  return true;
}
