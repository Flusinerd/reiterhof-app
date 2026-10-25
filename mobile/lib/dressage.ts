// Pure helpers for practice tests and freestyles (docs/domains/dressage-tests.md): the arena with its
// letters, scaling to the screen, paths, durations, the texts of a plan. No React Native imports
// (unit-tested, see dressage.test.ts).

export type ArenaSize = "20x40" | "20x60";
export type TestKind = "practice" | "freestyle";
export type Gait = "walk" | "trot" | "canter" | "halt";
export type Rein = "left" | "right" | "either" | "both";
export type Point = [number, number];

export const ARENA_SIZES: readonly { value: ArenaSize; label: string }[] = [
  { value: "20x40", label: "20×40" },
  { value: "20x60", label: "20×60" },
];

/** Classes of a practice test, lowest first. */
export const PRACTICE_CLASSES = ["E", "A", "L", "L**"] as const;
/** Classes offered for a freestyle when the catalog does not name any. */
export const FALLBACK_FREESTYLE_CLASSES = ["A", "L*", "L**"] as const;

export function arenaLength(arena: ArenaSize): number {
  return arena === "20x60" ? 60 : 40;
}

export function arenaLabel(arena: string): string {
  return arena === "20x60" ? "20×60" : "20×40";
}

// --- letters ---------------------------------------------------------------------------------

export type ArenaLetter = { letter: string; x: number; y: number; side: "left" | "right" | "centre" | "end" };

const WIDTH = 20;
const LONG: Record<ArenaSize, { left: string[]; right: string[]; centre: string[] }> = {
  "20x40": { left: ["K", "E", "H"], right: ["F", "B", "M"], centre: ["D", "X", "G"] },
  "20x60": { left: ["K", "V", "E", "S", "H"], right: ["F", "P", "B", "R", "M"], centre: ["D", "L", "X", "I", "G"] },
};

/** Distances of the long side letters from A: 6 m from the corner, then every 14 m (20×40) or 12 m (20×60). */
export function letterOffsets(arena: ArenaSize): number[] {
  const n = LONG[arena].left.length;
  const step = n === 3 ? 14 : 12;
  return Array.from({ length: n }, (_, i) => 6 + i * step);
}

/** All letters with metre coordinates: A at (10, 0), C at (10, L), x across (0 = left seen from A). */
export function arenaLetters(arena: ArenaSize): ArenaLetter[] {
  const l = arenaLength(arena);
  const set = LONG[arena];
  const out: ArenaLetter[] = [
    { letter: "A", x: WIDTH / 2, y: 0, side: "end" },
    { letter: "C", x: WIDTH / 2, y: l, side: "end" },
  ];
  letterOffsets(arena).forEach((y, i) => {
    out.push({ letter: set.left[i]!, x: 0, y, side: "left" });
    out.push({ letter: set.right[i]!, x: WIDTH, y, side: "right" });
    out.push({ letter: set.centre[i]!, x: WIDTH / 2, y, side: "centre" });
  });
  return out;
}

// --- scaling --------------------------------------------------------------------------------

export type Fit = {
  /** Pixels per metre. */
  scale: number;
  width: number;
  height: number;
  /** Space around the fence for the letters, in pixels. */
  pad: number;
  toPx: (x: number, y: number) => Point;
};

/** Metres around the fence that the letters take. */
const LETTER_MARGIN_M = 2.4;

/**
 * Fits the arena into a box. A is at the bottom, C at the top, the left long side (seen from A) on the left.
 * The result never exceeds the box and keeps the proportions.
 */
export function fitArena(arena: ArenaSize, maxWidth: number, maxHeight: number): Fit {
  const l = arenaLength(arena);
  const scale = Math.max(1, Math.min(maxWidth / (WIDTH + 2 * LETTER_MARGIN_M), maxHeight / (l + 2 * LETTER_MARGIN_M)));
  const pad = LETTER_MARGIN_M * scale;
  return {
    scale,
    pad,
    width: WIDTH * scale + 2 * pad,
    height: l * scale + 2 * pad,
    toPx: (x, y) => [pad + x * scale, pad + (l - y) * scale],
  };
}

/** Pixel position of a letter, pushed `out` metres away from the fence for the letters at the edge. */
export function letterPosition(fit: Fit, letter: ArenaLetter, out = 1.5): Point {
  const dx = letter.side === "left" ? -out : letter.side === "right" ? out : 0;
  const dy = letter.side === "end" ? (letter.letter === "A" ? -out : out) : 0;
  return fit.toPx(letter.x + dx, letter.y + dy);
}

const r1 = (n: number) => Math.round(n * 10) / 10;

/** SVG path data ("M x y L x y …") of a polyline in metres; repeated points are dropped. */
export function pathToSvg(points: readonly Point[], fit: Fit): string {
  const parts: string[] = [];
  let last = "";
  for (const [x, y] of points) {
    const [px, py] = fit.toPx(x, y);
    const key = `${r1(px)} ${r1(py)}`;
    if (key === last) continue;
    last = key;
    parts.push(`${parts.length === 0 ? "M" : "L"} ${key}`);
  }
  return parts.join(" ");
}

// --- plan -----------------------------------------------------------------------------------

export type PlanLine = {
  n: number;
  /** Key of the figure variant; the server needs it to rebuild an edited plan (PUT), so it must survive a round trip. */
  option?: string;
  letters: string;
  text: string;
  gait: Gait | string;
  tempo?: string;
  rein?: string;
  seconds: number;
  figure?: string;
  path: Point[];
};

/** `rule` marks a rule of the Kür leaflet (a failing one is an error, the server refuses to store it); the others are advice. */
export type PlanCheck = { key: string; ok: boolean; rule: boolean; text: string };
export type PlanRequirement = { key: string; label: string; count: number; met: boolean };

export type Plan = {
  kind: TestKind;
  class: string;
  arena: ArenaSize;
  seed: number;
  lines: PlanLine[];
  seconds: number;
  reins: { left: number; right: number };
  gaits: { walk: number; trot: number; canter: number };
  music: { gait: string; seconds: number }[];
  checks: PlanCheck[];
  requirements: PlanRequirement[];
};

const num = (v: unknown, d = 0) => (typeof v === "number" && Number.isFinite(v) ? v : d);
const arr = <T>(v: unknown): T[] => (Array.isArray(v) ? (v as T[]) : []);

/**
 * Reads a plan from the server: the body of a preview is the plan itself, a stored test carries it as
 * `plan` next to `id` and `title`. Missing parts become empty so that an older or newer backend does
 * not crash the view.
 */
export function normalizePlan(body: unknown): Plan | null {
  if (typeof body !== "object" || body === null) return null;
  const root = body as Record<string, unknown>;
  const p = (typeof root.plan === "object" && root.plan !== null ? root.plan : root) as Record<string, unknown>;
  if (!Array.isArray(p.lines)) return null;
  const lines = arr<Record<string, unknown>>(p.lines).map((l, i) => ({
    n: num(l.n, i + 1),
    option: typeof l.option === "string" && l.option !== "" ? l.option : undefined,
    letters: String(l.letters ?? ""),
    text: String(l.text ?? ""),
    gait: String(l.gait ?? ""),
    tempo: typeof l.tempo === "string" ? l.tempo : undefined,
    rein: typeof l.rein === "string" ? l.rein : undefined,
    seconds: num(l.seconds),
    figure: typeof l.figure === "string" ? l.figure : undefined,
    path: arr<unknown>(l.path)
      .filter((q): q is Point => Array.isArray(q) && q.length >= 2)
      .map((q) => [num(q[0]), num(q[1])] as Point),
  }));
  const reins = (p.reins ?? {}) as Record<string, unknown>;
  const gaits = (p.gaits ?? {}) as Record<string, unknown>;
  return {
    kind: (p.kind ?? root.kind) === "freestyle" ? "freestyle" : "practice",
    class: String(p.class ?? root.class ?? ""),
    arena: (p.arena ?? root.arena) === "20x60" ? "20x60" : "20x40",
    seed: num(p.seed ?? root.seed),
    lines,
    seconds: num(p.seconds, lines.reduce((s, l) => s + l.seconds, 0)),
    reins: { left: num(reins.left), right: num(reins.right) },
    gaits: { walk: num(gaits.walk), trot: num(gaits.trot), canter: num(gaits.canter) },
    music: arr<Record<string, unknown>>(p.music).map((m) => ({ gait: String(m.gait ?? ""), seconds: num(m.seconds) })),
    checks: arr<Record<string, unknown>>(p.checks).map((c) => ({ key: String(c.key ?? ""), ok: c.ok === true, rule: c.rule !== false, text: String(c.text ?? "") })),
    requirements: arr<Record<string, unknown>>(p.requirements).map((r) => ({
      key: String(r.key ?? ""),
      label: String(r.label ?? r.key ?? ""),
      count: num(r.count, 1),
      met: r.met === true,
    })),
  };
}

/**
 * What a 403 of the test routes means in German: the tests of a horse are for the owner, admins and riders who
 * may log sessions; deleting is for the owner, admins and the author of the test.
 */
export function forbiddenText(action: "use" | "delete" = "use"): string {
  return action === "delete"
    ? "Löschen dürfen nur die Besitzerin oder der Besitzer, Admins und wer die Aufgabe erstellt hat."
    : "Aufgaben und Küren dieses Pferdes sehen und erstellen nur die Besitzerin oder der Besitzer, Admins und Reiter, die Einheiten eintragen dürfen.";
}

/** "m:ss" (3:05); negative or invalid values count as 0. */
export function formatSeconds(seconds: number): string {
  const s = Math.max(0, Math.round(Number.isFinite(seconds) ? seconds : 0));
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, "0")}`;
}

/** A duration for a screen reader: "1 Minute 5 Sekunden". */
export function spokenSeconds(seconds: number): string {
  const s = Math.max(0, Math.round(seconds));
  const m = Math.floor(s / 60);
  const rest = s % 60;
  const minutes = m === 0 ? "" : `${m} ${m === 1 ? "Minute" : "Minuten"}`;
  const secs = rest === 0 && m > 0 ? "" : `${rest} ${rest === 1 ? "Sekunde" : "Sekunden"}`;
  return [minutes, secs].filter(Boolean).join(" ");
}

/** "3. A–X: Einreiten …" for lists and the share text. */
export function lineText(line: Pick<PlanLine, "n" | "letters" | "text">): string {
  return `${line.n}. ${line.letters ? `${line.letters}: ` : ""}${line.text}`;
}

/** Numbers the lines 1…n again (after reordering or removing). */
export function renumber(lines: readonly PlanLine[]): PlanLine[] {
  return lines.map((l, i) => ({ ...l, n: i + 1 }));
}

/** Moves an item by `delta` places; a copy of the list when it cannot move. */
export function moveLine<T>(lines: readonly T[], index: number, delta: number): T[] {
  const to = index + delta;
  if (index < 0 || index >= lines.length || to < 0 || to >= lines.length) return [...lines];
  const next = [...lines];
  const [item] = next.splice(index, 1);
  next.splice(to, 0, item!);
  return next;
}

export function removeLine<T>(lines: readonly T[], index: number): T[] {
  return lines.filter((_, i) => i !== index);
}

/** Share of each rein in whole percent; null when nothing was ridden. */
export function reinPercent(reins: { left: number; right: number }): { left: number; right: number } | null {
  const total = reins.left + reins.right;
  if (!(total > 0)) return null;
  const left = Math.round((reins.left / total) * 100);
  return { left, right: 100 - left };
}

/** "Beide Hände ausgewogen: links 48 %, rechts 52 %"; balanced means each rein has 40 to 60 percent. */
export function reinBalanceText(reins: { left: number; right: number }): string {
  const p = reinPercent(reins);
  if (!p) return "";
  const detail = `links ${p.left} %, rechts ${p.right} %`;
  if (p.left >= 40 && p.left <= 60) return `Beide Hände ausgewogen: ${detail}`;
  return `${p.left > p.right ? "Linke" : "Rechte"} Hand überwiegt: ${detail}`;
}

/** Rules and advice of a plan, the failing ones first (they are what the rider has to look at). */
export function splitChecks(checks: readonly PlanCheck[]): { rules: PlanCheck[]; advice: PlanCheck[] } {
  const ordered = (list: PlanCheck[]) => [...list.filter((k) => !k.ok), ...list.filter((k) => k.ok)];
  return { rules: ordered(checks.filter((k) => k.rule)), advice: ordered(checks.filter((k) => !k.rule)) };
}

/** One row of the checklist of required movements. */
export type RequirementRow = { key: string; text: string; met: boolean };

/** "Mitteltrab" or "Volte im Galopp (2×)" with its state; open ones first, so that the gaps are on top. */
export function requirementRows(reqs: readonly PlanRequirement[]): RequirementRow[] {
  const rows = reqs.map((r) => ({ key: r.key, text: r.count > 1 ? `${r.label} (${r.count}×)` : r.label, met: r.met }));
  return [...rows.filter((r) => !r.met), ...rows.filter((r) => r.met)];
}

export function requirementSummary(reqs: readonly PlanRequirement[]): string {
  if (reqs.length === 0) return "";
  const met = reqs.filter((r) => r.met).length;
  return met === reqs.length ? "Alle Anforderungen erfüllt" : `${met} von ${reqs.length} Anforderungen erfüllt`;
}

export function gaitLabel(gait: string): string {
  switch (gait) {
    case "walk":
      return "Schritt";
    case "trot":
      return "Trab";
    case "canter":
      return "Galopp";
    case "halt":
      return "Halt";
    default:
      return gait;
  }
}

export type MusicPart = { gait: string; label: string; seconds: number; from: number; to: number };

/** Groups consecutive segments of one gait into music parts and gives each its start and end second. */
export function musicParts(music: readonly { gait: string; seconds: number }[]): MusicPart[] {
  const parts: MusicPart[] = [];
  let at = 0;
  for (const m of music) {
    const s = Math.max(0, m.seconds);
    const last = parts[parts.length - 1];
    if (last && last.gait === m.gait) {
      last.seconds += s;
      last.to = at + s;
    } else {
      parts.push({ gait: m.gait, label: gaitLabel(m.gait), seconds: s, from: at, to: at + s });
    }
    at += s;
  }
  return parts;
}

export function musicPartText(p: MusicPart): string {
  return `${p.label} · ${formatSeconds(p.seconds)} (${formatSeconds(p.from)}–${formatSeconds(p.to)})`;
}

/** Default title: "Übungsaufgabe E · 20×40" or "Kür L* · 20×60". */
export function defaultTitle(kind: TestKind, cls: string, arena: string): string {
  return `${kind === "freestyle" ? "Kür" : "Übungsaufgabe"} ${cls} · ${arenaLabel(arena)}`;
}

/** Plain text for the share sheet. */
export function shareText(title: string, plan: Plan): string {
  const out = [title, `${arenaLabel(plan.arena)} · ${formatSeconds(plan.seconds)} Min.`, ""];
  for (const l of plan.lines) out.push(`${lineText(l)} (${formatSeconds(l.seconds)})`);
  const balance = reinBalanceText(plan.reins);
  if (balance) out.push("", balance);
  if (plan.kind === "freestyle") {
    const music = musicParts(plan.music);
    if (music.length > 0) out.push("", "Musikteile:", ...music.map((p) => `- ${musicPartText(p)}`));
    if (plan.requirements.length > 0) {
      out.push("", "Anforderungen:", ...requirementRows(plan.requirements).map((r) => `${r.met ? "[x]" : "[ ]"} ${r.text}`));
    }
  }
  out.push("", "Erstellt mit Stallfunk");
  return out.join("\n");
}

// --- playback -------------------------------------------------------------------------------

function pathLength(points: readonly Point[]): number {
  let len = 0;
  for (let i = 1; i < points.length; i++) len += Math.hypot(points[i]![0] - points[i - 1]![0], points[i]![1] - points[i - 1]![1]);
  return len;
}

/** The point on a polyline at a fraction (0..1) of its length. */
export function pointAlong(points: readonly Point[], fraction: number): Point | null {
  if (points.length === 0) return null;
  if (points.length === 1) return points[0]!;
  const total = pathLength(points);
  if (total === 0) return points[0]!;
  let want = Math.min(1, Math.max(0, fraction)) * total;
  for (let i = 1; i < points.length; i++) {
    const a = points[i - 1]!;
    const b = points[i]!;
    const seg = Math.hypot(b[0] - a[0], b[1] - a[1]);
    if (want <= seg && seg > 0) return [a[0] + ((b[0] - a[0]) * want) / seg, a[1] + ((b[1] - a[1]) * want) / seg];
    want -= seg;
  }
  return points[points.length - 1]!;
}

/** Where the rider is `t` seconds into the test: the line (index) and the point; null for a plan without paths. */
export function positionAt(lines: readonly PlanLine[], t: number): { index: number; point: Point } | null {
  let start = 0;
  for (let i = 0; i < lines.length; i++) {
    const l = lines[i]!;
    const end = start + Math.max(0, l.seconds);
    if (t < end || i === lines.length - 1) {
      const fraction = l.seconds > 0 ? (t - start) / l.seconds : 1;
      const point = pointAlong(l.path, fraction);
      return point ? { index: i, point } : null;
    }
    start = end;
  }
  return null;
}

// --- editor of the requirements -----------------------------------------------------------------

export type RequirementMovement = { key: string; count: number; rein: Rein };
export type RequirementSet = {
  id: string;
  class: string;
  label: string;
  movements: RequirementMovement[];
  min_seconds: number;
  max_seconds: number;
};

export const REIN_OPTIONS: readonly { value: Rein; label: string }[] = [
  { value: "either", label: "Egal" },
  { value: "left", label: "Links" },
  { value: "right", label: "Rechts" },
  { value: "both", label: "Beide" },
];

/** Adds a movement (count 1, either rein) or removes it when it is in the list already. */
export function toggleMovement(list: readonly RequirementMovement[], key: string): RequirementMovement[] {
  return list.some((m) => m.key === key) ? list.filter((m) => m.key !== key) : [...list, { key, count: 1, rein: "either" }];
}

/** Changes count (kept between 1 and 9) or rein of one movement. */
export function updateMovement(
  list: readonly RequirementMovement[],
  key: string,
  patch: Partial<Omit<RequirementMovement, "key">>,
): RequirementMovement[] {
  return list.map((m) => {
    if (m.key !== key) return m;
    const next = { ...m, ...patch };
    return { ...next, count: Math.max(1, Math.min(9, next.count)) };
  });
}

/** "3:30 bis 4:00 Min." for the time window of a set. */
export function windowText(minSeconds: number, maxSeconds: number): string {
  return `${formatSeconds(minSeconds)} bis ${formatSeconds(maxSeconds)} Min.`;
}

/** The window as the editor holds it: whole seconds, max never below min. */
export function clampWindow(min: number, max: number): { min: number; max: number } {
  const lo = Math.max(0, Math.round(min));
  return { min: lo, max: Math.max(lo, Math.round(max)) };
}

/** Whether the editor's content can be sent: at least one movement and a window with min above 0 and max not below min. */
export function requirementsReady(movements: readonly RequirementMovement[], min: number, max: number): boolean {
  return movements.length > 0 && min > 0 && max >= min;
}

/** "3:30 bis 4:00 Min. · 9 Lektionen" for a saved set. */
export function setSummary(set: RequirementSet): string {
  const n = set.movements.length;
  return `${windowText(set.min_seconds, set.max_seconds)} · ${n} ${n === 1 ? "Lektion" : "Lektionen"}`;
}

/** The saved sets of one class; the order of the server is kept. */
export function setsOfClass(sets: readonly RequirementSet[], cls: string): RequirementSet[] {
  return sets.filter((s) => s.class === cls);
}

// --- catalog ------------------------------------------------------------------------------------

export type CatalogMovement = { key: string; label: string; gait: string; tempo?: string; class: string | null };
export type CatalogClass = { key: string; label: string };

export type Catalog = { movements: CatalogMovement[]; classes: CatalogClass[]; requirement_sets: RequirementSet[] };

export function normalizeSet(s: Record<string, unknown>): RequirementSet {
  return {
    id: String(s.id ?? ""),
    class: String(s.class ?? ""),
    label: String(s.label ?? ""),
    movements: arr<Record<string, unknown>>(s.movements).map((m) => ({
      key: String(m.key ?? ""),
      count: Math.max(1, num(m.count, 1)),
      rein: (["left", "right", "either", "both"].includes(String(m.rein)) ? m.rein : "either") as Rein,
    })),
    min_seconds: num(s.min_seconds),
    max_seconds: num(s.max_seconds),
  };
}

/** Reads GET /dressage/catalog: classes as `{key, label}`, movements (class may be empty), saved sets. */
export function normalizeCatalog(body: unknown): Catalog {
  const root = (typeof body === "object" && body !== null ? body : {}) as Record<string, unknown>;
  const classes = arr<unknown>(root.classes).map((c) => {
    if (typeof c === "string") return { key: c, label: c };
    const o = c as { key?: unknown; label?: unknown };
    return { key: String(o.key ?? ""), label: String(o.label ?? o.key ?? "") };
  });
  const movements = arr<Record<string, unknown>>(root.movements).map((m) => {
    const cls = m.class ?? m.lowest_class;
    return {
      key: String(m.key ?? ""),
      label: String(m.label ?? m.key ?? ""),
      gait: String(m.gait ?? ""),
      tempo: typeof m.tempo === "string" ? m.tempo : undefined,
      class: typeof cls === "string" && cls !== "" ? cls : null,
    };
  });
  return {
    movements: movements.filter((m) => m.key !== ""),
    classes: classes.filter((c) => c.key !== ""),
    requirement_sets: arr<Record<string, unknown>>(root.requirement_sets).map(normalizeSet),
  };
}

/** "Trab · ab Klasse A" or "Trab · Klasse nicht belegt". */
export function movementHint(m: CatalogMovement): string {
  return `${gaitLabel(m.gait)} · ${m.class ? `ab Klasse ${m.class}` : "Klasse nicht belegt"}`;
}

// --- classes ------------------------------------------------------------------------------------

const RANK = ["E", "A", "L", "L*", "L**", "M*", "M**", "S"];

export function classRank(cls: string): number {
  const i = RANK.indexOf(cls);
  return i < 0 ? RANK.length : i;
}

/**
 * The highest class the person is offered, from the profile: the goal level when set, else the current
 * level (beginner up to A, intermediate up to L**, advanced everything). null means no limit.
 */
export function classCeiling(level: string | null | undefined, goal: string | null | undefined): string | null {
  const g = goal || level;
  if (g === "beginner") return "A";
  if (g === "intermediate") return "L**";
  return null;
}

/** The classes that are not above the ceiling. */
export function allowedClasses<T extends string>(all: readonly T[], ceiling: string | null): T[] {
  if (!ceiling) return [...all];
  return all.filter((c) => classRank(c) <= classRank(ceiling));
}

/** The preselected class: from the horse's level (beginner E, intermediate L, advanced L**), the nearest lower one offered. */
export function defaultClass(all: readonly string[], level: string | null | undefined): string {
  const want = level === "advanced" ? "L**" : level === "intermediate" ? "L" : "E";
  if (all.includes(want)) return want;
  const lower = all.filter((c) => classRank(c) <= classRank(want)).pop();
  return lower ?? all[0] ?? want;
}

/** The official sources the app links to; their texts are never shown. */
export const OFFICIAL_LINKS: readonly { title: string; detail: string; url: string }[] = [
  { title: "FN Verlag", detail: "Aufgabenheft und Regelwerke", url: "https://www.fnverlag.de/" },
  { title: "Pferdesport Deutschland", detail: "Merkblatt Dressur-Kür", url: "https://www.pferdesport-deutschland.de/" },
];
