// Pure helpers of the training focuses and unit plans (docs/domains/training-planning.md): the focus
// catalog of GET /training/focuses, the horse's phase and its hint, selection limits, warnings, the
// three-part unit with its display, the block of the week and the show date a Turniervorbereitung
// ends with. No React Native imports (unit-tested with node --test). German UI text.

import { isValidDate } from "./training.ts";

// --- catalog ---------------------------------------------------------------------------

/** One entry of the catalog, GET /training/focuses (static data of the backend). */
export type FocusEntry = {
  key: string;
  label: string;
  /** `scale` (Skala der Ausbildung) or `practice` (Praxis). */
  group: string;
  /** 0 fits every phase; 1 to 3 are Gewöhnung, Schubkraft and Tragkraft. */
  phase: number;
  description: string;
  /** "Woran du es merkst", one sentence. */
  signs: string;
  /** Activities the focus fits. */
  fits: string[];
};

/** A focus of the horse's profile; `until` ("YYYY-MM-DD") ends it. */
export type ProfileFocus = { key: string; until?: string | null };

/** A hint of the profile; `key` is a focus key, or `level_up` for the level goal. */
export type FocusWarning = { key: string; text: string };

/**
 * German labels of the 15 catalog keys, only for places that know a key but not (yet) the catalog:
 * the Heute timeline, a catalog that did not load, an older backend. The catalog's own label wins.
 */
const FALLBACK_LABELS: Record<string, string> = {
  takt: "Takt",
  losgelassenheit: "Losgelassenheit",
  anlehnung: "Anlehnung",
  schwung: "Schwung",
  geraderichten: "Geraderichtung",
  versammlung: "Versammlung",
  durchlaessigkeit: "Durchlässigkeit",
  seitengaenge: "Seitengänge",
  gelassenheit: "Gelassenheit",
  kondition: "Kondition",
  springen: "Springen",
  gelaende: "Gelände",
  vertrauen: "Vertrauen",
  mobilitaet: "Beweglichkeit",
  turnier: "Turniervorbereitung",
};

const capitalize = (s: string) => (s ? s.charAt(0).toUpperCase() + s.slice(1) : s);

/** Label of a focus key: from the catalog, else the fallback table, else the key made readable. */
export function focusLabel(key: string, catalog?: readonly FocusEntry[]): string {
  return catalog?.find((e) => e.key === key)?.label ?? FALLBACK_LABELS[key] ?? capitalize(key.replace(/_/g, " "));
}

/** "Takt, Losgelassenheit" for a list of keys. */
export function focusLabels(keys: readonly string[], catalog?: readonly FocusEntry[]): string {
  return keys.map((k) => focusLabel(k, catalog)).join(", ");
}

const GROUP_LABELS: Record<string, string> = { scale: "Skala der Ausbildung", practice: "Praxis" };
const GROUP_ORDER = ["scale", "practice"];

export function groupLabel(group: string): string {
  return GROUP_LABELS[group] ?? capitalize(group);
}

export type FocusSection = { group: string; label: string; entries: FocusEntry[] };

/**
 * The catalog as sections in display order, "Skala der Ausbildung" before "Praxis" (unknown groups
 * after them); inside a group the catalog's order stays. Groups without entries are left out.
 */
export function focusSections(catalog: readonly FocusEntry[]): FocusSection[] {
  const groups = [...GROUP_ORDER];
  for (const e of catalog) if (!groups.includes(e.group)) groups.push(e.group);
  return groups
    .map((group) => ({ group, label: groupLabel(group), entries: catalog.filter((e) => e.group === group) }))
    .filter((s) => s.entries.length > 0);
}

/**
 * The chips of a day's focus picker: the focuses that fit the activity, plus the current key even
 * when it does not fit (so that it can be seen and changed). Catalog order.
 */
export function focusChoices(
  catalog: readonly FocusEntry[],
  activity: string | null,
  current?: string | null,
): FocusEntry[] {
  return catalog.filter((e) => e.key === current || (activity !== null && (e.fits ?? []).includes(activity)));
}

// --- the horse's phase -----------------------------------------------------------------

export type HorsePhase = "gewoehnung" | "schubkraft" | "tragkraft";

const PHASE_NAMES: Record<HorsePhase, string> = {
  gewoehnung: "Gewöhnungsphase",
  schubkraft: "Entwicklung der Schubkraft",
  tragkraft: "Entwicklung der Tragkraft",
};

const PHASE_CATALOG: Record<HorsePhase, number> = { gewoehnung: 1, schubkraft: 2, tragkraft: 3 };

export function isHorsePhase(value: unknown): value is HorsePhase {
  return typeof value === "string" && value in PHASE_NAMES;
}

/** "Gewöhnungsphase", "Entwicklung der Schubkraft", "Entwicklung der Tragkraft"; null for anything else. */
export function phaseName(phase: string | null | undefined): string | null {
  return isHorsePhase(phase) ? PHASE_NAMES[phase] : null;
}

/** The highest catalog phase a horse in this phase may be given (1, 2 or 3); null when unknown. */
export function catalogPhaseOf(phase: string | null | undefined): number | null {
  return isHorsePhase(phase) ? PHASE_CATALOG[phase] : null;
}

/** True when the focus needs a higher phase than the horse has (phase 0 fits every phase). */
export function isAbovePhase(entry: Pick<FocusEntry, "phase">, catalogPhase: number | null | undefined): boolean {
  return catalogPhase != null && entry.phase > catalogPhase;
}

/**
 * The hint on top of the focus editor, built from the profile's `phase` and `focus_hints` (the scale
 * focuses of the phase that are not chosen yet): "Luna ist in der Gewöhnungsphase: Takt,
 * Losgelassenheit, Anlehnung". Null without a phase (older backend).
 */
export function phaseHint(
  horseName: string,
  phase: string | null | undefined,
  hints: readonly string[] | undefined,
  catalog?: readonly FocusEntry[],
): string | null {
  const name = phaseName(phase);
  if (!name) return null;
  const who = horseName.trim() || "Das Pferd";
  const labels = focusLabels(hints ?? [], catalog);
  return labels ? `${who} ist in der ${name}: ${labels}` : `${who} ist in der ${name}.`;
}

// --- selection -------------------------------------------------------------------------

/** At most 3 focuses on the horse (owner), at most 2 of one's own (owner and riders). */
export const FOCUS_LIMIT_OWNER = 3;
export const FOCUS_LIMIT_RIDER = 2;

/** "2 von 3 gewählt". */
export function selectionCaption(count: number, limit: number): string {
  return `${count} von ${limit} gewählt`;
}

/** What a tap on a row says when the limit is reached. */
export function limitNotice(limit: number): string {
  return `Höchstens ${limit} Schwerpunkte. Nimm erst einen heraus.`;
}

/** Adds the key at the end or removes it; `full` is true when it could not be added (limit). */
export function toggleKey(keys: readonly string[], key: string, limit: number): { keys: string[]; full: boolean } {
  if (keys.includes(key)) return { keys: keys.filter((k) => k !== key), full: false };
  if (keys.length >= limit) return { keys: [...keys], full: true };
  return { keys: [...keys, key], full: false };
}

/** A chosen focus in the editor: `until` is "YYYY-MM-DD" or "" (no end). */
export type FocusDraft = { key: string; until: string };

export function focusDraftsFrom(focuses: readonly ProfileFocus[] | null | undefined): FocusDraft[] {
  return (focuses ?? []).map((f) => ({ key: f.key, until: f.until ?? "" }));
}

/** The `focuses` of PUT /training-profile: `until` only when there is one. */
export function focusInputs(drafts: readonly FocusDraft[]): ProfileFocus[] {
  return drafts.map((d) => (d.until ? { key: d.key, until: d.until } : { key: d.key }));
}

/** Adds the focus (with `until` as its end, "" for none) or removes it. */
export function toggleFocus(
  drafts: readonly FocusDraft[],
  key: string,
  limit: number,
  until = "",
): { drafts: FocusDraft[]; full: boolean } {
  const existing = drafts.find((d) => d.key === key);
  if (existing) return { drafts: drafts.filter((d) => d.key !== key), full: false };
  if (drafts.length >= limit) return { drafts: [...drafts], full: true };
  return { drafts: [...drafts, { key, until }], full: false };
}

/** Moves the focus at `index` up (-1) or down (+1); the first has the highest priority. */
export function moveFocus(drafts: readonly FocusDraft[], index: number, delta: -1 | 1): FocusDraft[] {
  const target = index + delta;
  if (index < 0 || index >= drafts.length || target < 0 || target >= drafts.length) return [...drafts];
  const next = [...drafts];
  const [moved] = next.splice(index, 1);
  if (moved) next.splice(target, 0, moved);
  return next;
}

export function setFocusUntil(drafts: readonly FocusDraft[], key: string, until: string): FocusDraft[] {
  return drafts.map((d) => (d.key === key ? { ...d, until } : d));
}

/** German reason why the selection cannot be saved, or null. */
export function validateFocuses(drafts: readonly FocusDraft[], limit: number): string | null {
  if (drafts.length > limit) return `Höchstens ${limit} Schwerpunkte.`;
  if (new Set(drafts.map((d) => d.key)).size !== drafts.length) return "Jeden Schwerpunkt nur einmal wählen.";
  if (drafts.some((d) => d.until !== "" && !isValidDate(d.until))) return "„Bis“ braucht ein gültiges Datum.";
  return null;
}

/** "DD.MM." of a YYYY-MM-DD date, "" when it is not one. */
function dayMonth(date: string): string {
  const m = /^\d{4}-(\d{2})-(\d{2})$/.exec(date);
  return m ? `${m[2]}.${m[1]}.` : "";
}

/** The profile row: "Losgelassenheit, Turniervorbereitung (bis 28.10.)", or "Keine gesetzt". */
export function focusesSummary(focuses: readonly ProfileFocus[] | null | undefined, catalog?: readonly FocusEntry[]): string {
  if (!focuses || focuses.length === 0) return "Keine gesetzt";
  return focuses
    .map((f) => {
      const until = f.until ? dayMonth(f.until) : "";
      return until ? `${focusLabel(f.key, catalog)} (bis ${until})` : focusLabel(f.key, catalog);
    })
    .join(", ");
}

/** The training tab row "Mein Schwerpunkt". */
export function myFocusSummary(keys: readonly string[] | null | undefined, catalog?: readonly FocusEntry[]): string {
  return keys && keys.length > 0 ? focusLabels(keys, catalog) : "Kein eigener Schwerpunkt";
}

// --- the show a Turniervorbereitung ends with ------------------------------------------

/** The first date from today on (YYYY-MM-DD strings compare as dates); null without one. */
export function nextShowDate(dates: readonly string[], today: string): string | null {
  const upcoming = dates.filter((d) => isValidDate(d) && d >= today).sort();
  return upcoming[0] ?? null;
}

/** The end a newly chosen focus starts with: the next show for `turnier`, else none (""). */
export function defaultUntil(key: string, showDates: readonly string[], today: string): string {
  return key === "turnier" ? (nextShowDate(showDates, today) ?? "") : "";
}

// --- warnings --------------------------------------------------------------------------

/** The `key` of the warning that offers the level change (see lib/training-goal.ts). */
export const LEVEL_UP_KEY = "level_up";

/** The warnings that belong to a focus: all but the level goal's. */
export function focusWarnings(warnings: readonly FocusWarning[] | null | undefined): FocusWarning[] {
  return (warnings ?? []).filter((w) => w.key !== LEVEL_UP_KEY);
}

/** The server's text for a focus, or null. */
export function warningFor(key: string, warnings: readonly FocusWarning[] | null | undefined): string | null {
  return focusWarnings(warnings).find((w) => w.key === key)?.text ?? null;
}

/**
 * The quiet line under a chosen focus: the server's warning text, else (before it is saved) that
 * the focus is above the horse's phase, which the planner then uses at most once a week.
 */
export function focusNote(
  entry: Pick<FocusEntry, "key" | "phase"> | undefined,
  warnings: readonly FocusWarning[] | null | undefined,
  catalogPhase: number | null | undefined,
  horseName: string,
): string | null {
  if (!entry) return null;
  const text = warningFor(entry.key, warnings);
  if (text) return text;
  if (isAbovePhase(entry, catalogPhase)) {
    const who = horseName.trim() || "dem Pferd";
    return `Über der Phase von ${who}: wird höchstens einmal pro Woche geplant.`;
  }
  return null;
}

// --- focus of a day --------------------------------------------------------------------

/**
 * The chip of a day: the label the API sent, else the label of the key, else null. Free text
 * (`focus`) is not a chip.
 */
export function dayFocusLabel(
  day: { focus_key?: string | null; focus_label?: string | null },
  catalog?: readonly FocusEntry[],
): string | null {
  if (day.focus_label) return day.focus_label;
  return day.focus_key ? focusLabel(day.focus_key, catalog) : null;
}

/**
 * The chip and the free text under it ("Eigener Text"): with a chip the text is shown as it is,
 * without one it gets the prefix "Schwerpunkt: ".
 */
export function dayFocusLines(
  day: { focus_key?: string | null; focus_label?: string | null; focus?: string | null },
  catalog?: readonly FocusEntry[],
): { chip: string | null; text: string | null } {
  const chip = dayFocusLabel(day, catalog);
  const free = day.focus?.trim() ?? "";
  if (!free) return { chip, text: null };
  return { chip, text: chip ? free : `Schwerpunkt: ${free}` };
}

/** The focus key a finished session is recorded with, see `sessionFocusKey`. */
export type SlotLike = { date: string; is_today?: boolean; activity?: string | null; focus_key?: string | null };

/**
 * `focus_key` of a session: the focus the ride was started from (the recommendation), else the focus
 * of the day's slot when the slot's activity is the one that was ridden. `date` is the stable-local
 * day of the session; without it the slot of today counts. Undefined when there is no focus.
 */
export function sessionFocusKey(opts: {
  requested?: string | null;
  activity: string;
  date?: string;
  days?: readonly SlotLike[];
}): string | undefined {
  if (opts.requested) return opts.requested;
  const slot = (opts.days ?? []).find((d) => (opts.date ? d.date === opts.date : d.is_today === true));
  return slot?.focus_key && slot.activity === opts.activity ? slot.focus_key : undefined;
}

// --- the unit: Lösen, Arbeiten, Entspannen ---------------------------------------------

export type UnitPartKey = "warmup" | "main" | "cooldown";

export const UNIT_PARTS: readonly { key: UnitPartKey; label: string }[] = [
  { key: "warmup", label: "Lösen" },
  { key: "main", label: "Arbeiten" },
  { key: "cooldown", label: "Entspannen" },
];

export type UnitExercise = { id: string; title: string };

/** One part of a unit as the API sends it; `instead_of` is the exercise the stable's equipment ruled out. */
export type UnitPart = { exercise: UnitExercise | null; minutes: number; instead_of?: UnitExercise | null };

/** The unit plan of a day or a recommendation; a part may be absent. */
export type Unit = { warmup?: UnitPart | null; main?: UnitPart | null; cooldown?: UnitPart | null };

/** `unit` of PUT /week/{day}: the exercise by id (null = none), the minutes of the part. */
export type UnitBody = Partial<Record<UnitPartKey, { exercise_id: string | null; minutes: number }>>;

export type UnitRow = {
  part: UnitPartKey;
  /** "Lösen", "Arbeiten" or "Entspannen". */
  label: string;
  minutes: number;
  exercise: UnitExercise | null;
  /** "statt Trabstangen" when the planner had to leave out an exercise, else null. */
  insteadOf: string | null;
  /** "15 Lösen · Cavaletti auf dem Zirkel". */
  text: string;
  accessibilityLabel: string;
};

/** The rows of a unit in the order Lösen, Arbeiten, Entspannen; absent parts are left out. */
export function unitRows(unit: Unit | null | undefined): UnitRow[] {
  const rows: UnitRow[] = [];
  for (const { key, label } of UNIT_PARTS) {
    const part = unit?.[key];
    if (!part) continue;
    const minutes = Math.round(part.minutes);
    if (!(minutes > 0) && !part.exercise) continue;
    const exercise = part.exercise ?? null;
    const insteadOf = part.instead_of ? `statt ${part.instead_of.title}` : null;
    rows.push({
      part: key,
      label,
      minutes,
      exercise,
      insteadOf,
      text: `${minutes} ${label}${exercise ? ` · ${exercise.title}` : ""}`,
      accessibilityLabel: `${label}, ${minutes} Minuten: ${exercise ? exercise.title : "ohne feste Übung"}${insteadOf ? `, ${insteadOf}` : ""}`,
    });
  }
  return rows;
}

/** Minutes of all parts together. */
export function unitMinutes(unit: Unit | null | undefined): number {
  return unitRows(unit).reduce((sum, r) => sum + r.minutes, 0);
}

/** The main exercise of a day: the unit's, else the single exercise older backends send. */
export function mainExercise(day: { unit?: Unit | null; exercise?: UnitExercise | null }): UnitExercise | null {
  return day.unit?.main?.exercise ?? day.exercise ?? null;
}

/** Body of the unit in PUT /week/{day}. */
export function unitBody(unit: Unit): UnitBody {
  const body: UnitBody = {};
  for (const { key } of UNIT_PARTS) {
    const part = unit[key];
    if (part) body[key] = { exercise_id: part.exercise?.id ?? null, minutes: Math.round(part.minutes) };
  }
  return body;
}

const clamp = (n: number, lo: number, hi: number) => Math.min(hi, Math.max(lo, n));

/**
 * The minutes of Lösen, Arbeiten and Entspannen for a unit of `minutes`, mirrored from the backend
 * (training.UnitSplit, the table "Unit" of docs/domains/training-planning.md): hall, arena and
 * jumping Lösen 33 % (10 to 25, jumping at least 15) and Entspannen 17 % (5 to 10), the rest is
 * Arbeiten (at least 10; a unit too short for the minimums keeps the proportions); lunge 10 / rest /
 * 5, groundwork 5 / rest / 5, hack 10 / rest / 10. Seniors and cold nights each add five minutes to
 * Lösen (at most 30 in all), seniors five to Entspannen too; the time comes off Arbeiten, which keeps
 * its minimum. The parts always add up to `minutes`. Null for activities without a unit plan
 * (walker, rest). The app does not know the age or the night's cold when it re-splits a day, so it
 * passes no extras; the planner's proposals carry them.
 */
export function unitSplit(
  activity: string,
  minutes: number,
  extras: { senior?: boolean; cold?: boolean } = {},
): Record<UnitPartKey, number> | null {
  const total = Math.round(minutes);
  if (!(total > 0)) return null;
  let warmup: number;
  let cooldown: number;
  let minMain: number;
  if (activity === "hall" || activity === "arena" || activity === "jumping") {
    warmup = clamp(Math.round(total * 0.33), 10, 25);
    if (activity === "jumping") warmup = Math.max(warmup, 15);
    cooldown = clamp(Math.round(total * 0.17), 5, 10);
    minMain = 10;
    if (total < warmup + cooldown + minMain) {
      // Too short for the minimums: the proportions of the base split.
      const w = Math.round(total * 0.33);
      const c = Math.round(total * 0.17);
      return { warmup: w, main: total - w - c, cooldown: c };
    }
  } else if (activity === "lunge") {
    [warmup, cooldown, minMain] = [10, 5, 5];
  } else if (activity === "groundwork") {
    [warmup, cooldown, minMain] = [5, 5, 5];
  } else if (activity === "hack") {
    [warmup, cooldown, minMain] = [10, 10, 5];
  } else {
    return null;
  }
  // A unit shorter than its fixed parts: the cool-down is cut first, then the warm-up.
  warmup = Math.min(warmup, total);
  cooldown = Math.min(cooldown, total - warmup);
  let room = total - warmup - cooldown - minMain; // the minutes the extras may take from Arbeiten
  const extraWarm = Math.min((extras.senior ? 5 : 0) + (extras.cold ? 5 : 0), Math.max(30 - warmup, 0), Math.max(room, 0));
  warmup += extraWarm;
  room -= extraWarm;
  if (extras.senior) cooldown += Math.min(5, Math.max(room, 0));
  return { warmup, main: total - warmup - cooldown, cooldown };
}

/**
 * The unit after the activity or the minutes of the day changed: every part gets its new share,
 * the chosen exercises stay. Undefined without a unit or for activities without a unit plan.
 */
export function resplitUnit(unit: Unit | null | undefined, activity: string, minutes: number): Unit | undefined {
  if (!unit) return undefined;
  const split = unitSplit(activity, minutes);
  if (!split) return undefined;
  const next: Unit = {};
  for (const { key } of UNIT_PARTS) {
    if (split[key] > 0) next[key] = { exercise: unit[key]?.exercise ?? null, minutes: split[key] };
  }
  return next;
}

/** The unit without exercises, kept for the minutes (when the activity moves to other libraries). */
export function clearUnitExercises(unit: Unit | null | undefined, activity: string, minutes: number): Unit | undefined {
  if (!unit) return undefined;
  const split = unitSplit(activity, minutes);
  if (!split) return undefined;
  const next: Unit = {};
  for (const { key } of UNIT_PARTS) if (split[key] > 0) next[key] = { exercise: null, minutes: split[key] };
  return next;
}

/**
 * Sets (or with null clears) the exercise of one part and leaves the minutes as they are. A day
 * without a unit gets one made from the split of its minutes; undefined when the activity has no
 * unit plan.
 */
export function setUnitExercise(
  unit: Unit | null | undefined,
  part: UnitPartKey,
  exercise: UnitExercise | null,
  activity: string,
  minutes: number,
): Unit | undefined {
  const base = unit ?? clearUnitExercises({}, activity, minutes);
  if (!base) return undefined;
  const minutesOfPart = base[part]?.minutes ?? unitSplit(activity, minutes)?.[part] ?? 0;
  return { ...base, [part]: { exercise, minutes: minutesOfPart } };
}

/** Whether an exercise of this phase can be used in this part: `any` fits all, no phase counts as `main`. */
export function phaseFitsPart(phase: string | null | undefined, part: UnitPartKey): boolean {
  const own = phase || "main";
  return own === "any" || own === part;
}

// --- the block of the week -------------------------------------------------------------

/** The block of four weeks a week belongs to (weeks 1 to 3 build, week 4 is the deload week). */
export type Block = { week: number; deload: boolean };

export const BLOCK_WEEKS = 4;

/** "Aufbauwoche 2 von 4" or "Entlastungswoche"; null without a block (older backend). */
export function blockText(block: Block | null | undefined): string | null {
  if (!block) return null;
  if (block.deload) return "Entlastungswoche";
  const week = Math.round(block.week);
  return week >= 1 && week <= BLOCK_WEEKS ? `Aufbauwoche ${week} von ${BLOCK_WEEKS}` : null;
}

/** "Wiederantrainieren, Woche 1" while the horse returns to work (`restart_week` > 0). */
export function restartText(week: number | null | undefined): string | null {
  return week != null && week > 0 ? `Wiederantrainieren, Woche ${Math.round(week)}` : null;
}

/** Eyebrow of the training tab: "Training · Fit · Aufbauwoche 2 von 4" (the restart text replaces the block). */
export function trainingEyebrow(
  status: string | null | undefined,
  restartWeek: number | null | undefined,
  block: Block | null | undefined,
): string {
  return ["Training", status, restartText(restartWeek) ?? blockText(block)].filter((p): p is string => !!p).join(" · ");
}
