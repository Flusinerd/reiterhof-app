// Pure helpers for editing blanket rules (JAN-30): drafts as the form holds them, parsing
// with German error texts, reordering. No React Native imports (unit-tested with node --test).

import { RAIN_STEPS, rainStrength, type RainStrength, type Rule } from "./blankets.ts";

export type RainChoice = "any" | "rain" | "dry";
export type { RainStrength };

/** One rule while it is edited: numbers are still text so that "" and "-" can be typed. */
export type RuleDraft = {
  /** Stable key for the list, not sent to the server. */
  key: string;
  temp_min: string;
  temp_max: string;
  rain: RainChoice;
  /** Only used with rain = "rain". */
  rain_strength: RainStrength;
  /** Own amounts for rain_strength = "exact", as text like the temperatures. */
  rain_min_mm: string;
  rain_max_mm: string;
  blanket_id: string | null;
  note: string;
};

export type RuleValue = {
  temp_min: number | null;
  temp_max: number | null;
  rain: boolean | null;
  rain_min_mm: number | null;
  rain_max_mm: number | null;
  blanket_id: string | null;
  note: string;
};

export const MAX_RULES = 30;
export const MAX_NOTE = 200;
export const TEMP_LIMIT = 60;
export const RAIN_LIMIT_MM = 500;

let counter = 0;
function nextKey(): string {
  counter += 1;
  return `rule-${counter}`;
}

function numberText(v: number | null): string {
  return v === null ? "" : String(v).replace(".", ",");
}

export function toDraft(
  rule: Pick<Rule, "temp_min" | "temp_max" | "rain" | "blanket_id" | "note"> &
    Partial<Pick<Rule, "rain_min_mm" | "rain_max_mm">>,
): RuleDraft {
  const min = rule.rain === true ? (rule.rain_min_mm ?? null) : null;
  const max = rule.rain === true ? (rule.rain_max_mm ?? null) : null;
  const strength = rainStrength({ rain_min_mm: min, rain_max_mm: max });
  return {
    key: nextKey(),
    temp_min: numberText(rule.temp_min),
    temp_max: numberText(rule.temp_max),
    rain: rule.rain === true ? "rain" : rule.rain === false ? "dry" : "any",
    rain_strength: strength,
    rain_min_mm: strength === "exact" ? numberText(min) : "",
    rain_max_mm: strength === "exact" ? numberText(max) : "",
    blanket_id: rule.blanket_id,
    note: rule.note,
  };
}

export function emptyDraft(): RuleDraft {
  return {
    key: nextKey(),
    temp_min: "",
    temp_max: "",
    rain: "any",
    rain_strength: "any",
    rain_min_mm: "",
    rain_max_mm: "",
    blanket_id: null,
    note: "",
  };
}

/**
 * Patch for choosing a rain strength. Switching to "exact" starts from the amounts of the step
 * chosen before (so "Stark" becomes "ab 8 mm" to adjust), unless amounts were typed already.
 */
export function chooseStrength(draft: RuleDraft, strength: RainStrength): Partial<RuleDraft> {
  if (strength !== "exact" || draft.rain_min_mm !== "" || draft.rain_max_mm !== "") return { rain_strength: strength };
  const prev = draft.rain_strength === "any" || draft.rain_strength === "exact" ? null : RAIN_STEPS[draft.rain_strength];
  return {
    rain_strength: strength,
    rain_min_mm: numberText(prev?.min ?? null),
    rain_max_mm: numberText(prev?.max ?? null),
  };
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
    return { ok: false, error: `${label}: Temperatur als Zahl, z. B. 5 oder -2,5.` };
  }
  if ((min !== null && Math.abs(min) > TEMP_LIMIT) || (max !== null && Math.abs(max) > TEMP_LIMIT)) {
    return { ok: false, error: `${label}: Temperatur zwischen -${TEMP_LIMIT} und ${TEMP_LIMIT} °C.` };
  }
  if (min !== null && max !== null && min >= max) {
    return { ok: false, error: `${label}: „Ab“ muss kleiner sein als „Unter“.` };
  }
  let rainMin: number | null = null;
  let rainMax: number | null = null;
  if (draft.rain === "rain" && draft.rain_strength === "exact") {
    const a = parseTemp(draft.rain_min_mm);
    const b = parseTemp(draft.rain_max_mm);
    if (a === undefined || b === undefined) {
      return { ok: false, error: `${label}: Regenmenge als Zahl, z. B. 2 oder 0,5.` };
    }
    if ((a !== null && (a < 0 || a > RAIN_LIMIT_MM)) || (b !== null && (b < 0 || b > RAIN_LIMIT_MM))) {
      return { ok: false, error: `${label}: Regenmenge zwischen 0 und ${RAIN_LIMIT_MM} mm.` };
    }
    if (a !== null && b !== null && a >= b) {
      return { ok: false, error: `${label}: Regenmenge: „Ab“ muss kleiner sein als „Unter“.` };
    }
    rainMin = a;
    rainMax = b;
  } else if (draft.rain === "rain") {
    const strength = draft.rain_strength;
    if (strength !== "any" && strength !== "exact") {
      rainMin = RAIN_STEPS[strength].min;
      rainMax = RAIN_STEPS[strength].max;
    }
  }
  const note = draft.note.trim();
  if (note.length > MAX_NOTE) {
    return { ok: false, error: `${label}: Wunsch: höchstens ${MAX_NOTE} Zeichen.` };
  }
  return {
    ok: true,
    value: {
      temp_min: min,
      temp_max: max,
      rain: draft.rain === "rain" ? true : draft.rain === "dry" ? false : null,
      rain_min_mm: rainMin,
      rain_max_mm: rainMax,
      blanket_id: draft.blanket_id,
      note,
    },
  };
}

/** Parses all drafts; the first problem wins. */
export function parseDrafts(drafts: readonly RuleDraft[]): { ok: true; value: RuleValue[] } | { ok: false; error: string } {
  if (drafts.length > MAX_RULES) return { ok: false, error: `Höchstens ${MAX_RULES} Regeln.` };
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
  // Amounts only exist on rain rules; an outer rule with amounts covers only rain rules inside them.
  // Amounts are never negative, so an open lower bound is 0.
  if ((outer.rain_min_mm ?? 0) > (inner.rain_min_mm ?? 0)) return false;
  if (hi(outer.rain_max_mm) < hi(inner.rain_max_mm)) return false;
  return true;
}
