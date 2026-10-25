// Pure helpers of the stable's training equipment and the exercises' set-up
// (docs/domains/training-planning.md, "Equipment and alternatives"): labels, the defaults of the
// locations, the admin's editing, the day's tri-state "Aufbau heute" and what is available. No React
// Native imports (unit-tested with node --test). German UI text.

import { DISCIPLINES, exerciseLibraries } from "./training.ts";

// --- keys and labels -------------------------------------------------------------------

/** Facility equipment: planned against the stable (what is set up in the hall, the arena, ...). */
export const FACILITY_KEYS = ["poles", "cavaletti", "jumps", "obstacles"] as const;

/** Personal equipment: always available, never planned against the stable. */
export const PERSONAL_KEYS = [
  "lunge",
  "side_reins",
  "double_lunge",
  "kappzaum",
  "tarp",
  "umbrella",
  "cones",
  "flag",
  "whip",
  "neck_ring",
  "trailer",
] as const;

const LABELS: Record<string, string> = {
  poles: "Stangen",
  cavaletti: "Cavaletti",
  jumps: "Sprünge",
  obstacles: "Gelassenheitsparcours",
  lunge: "Longe",
  side_reins: "Ausbindezügel",
  double_lunge: "Doppellonge",
  kappzaum: "Kappzaum",
  tarp: "Plane",
  umbrella: "Schirm",
  cones: "Hütchen",
  flag: "Flatterband",
  whip: "Gerte",
  neck_ring: "Halsring",
  trailer: "Anhänger",
};

/** German word of an equipment key; an unknown key is made readable ("side_reins" -> "Side reins"). */
export function equipmentLabel(key: string): string {
  const known = LABELS[key];
  if (known) return known;
  const words = key.replace(/_/g, " ");
  return words ? words.charAt(0).toUpperCase() + words.slice(1) : words;
}

/** "Stangen, Cavaletti"; "" for none. */
export function equipmentList(keys: readonly string[] | null | undefined): string {
  return (keys ?? []).map(equipmentLabel).join(", ");
}

/** The facility keys the server names (`keys.facility`), else the fixed list. */
export function facilityKeysOf(stable?: { keys?: { facility?: string[] } } | null): readonly string[] {
  const named = stable?.keys?.facility;
  return named && named.length > 0 ? named : FACILITY_KEYS;
}

/** The facility keys among the given ones, in the order of the fixed list, each once. */
export function facilityOf(equipment: readonly string[] | null | undefined, facility: readonly string[] = FACILITY_KEYS): string[] {
  return facility.filter((k) => (equipment ?? []).includes(k));
}

// --- locations -------------------------------------------------------------------------

/** A location is an activity; hack and walker have none. */
export type LocationKey = "hall" | "arena" | "lunge" | "jumping" | "groundwork";

export const LOCATIONS: readonly { key: LocationKey; label: string }[] = [
  { key: "hall", label: "Halle" },
  { key: "arena", label: "Platz" },
  { key: "lunge", label: "Longierhalle" },
  { key: "jumping", label: "Springplatz" },
  { key: "groundwork", label: "Bodenarbeitsplatz" },
];

/** What the stable has at a location; `shared` means set-up needs an agreement with others. */
export type LocationEquipment = { equipment: string[]; shared: boolean };

/** GET /stables/training-equipment. `set` is false while nobody saved the equipment (the locations are the defaults). */
export type StableEquipment = {
  set: boolean;
  locations: Partial<Record<string, LocationEquipment>>;
  keys?: { facility?: string[]; personal?: string[] };
};

/** What a location has when nobody entered anything (the backend's defaults). */
export const DEFAULT_LOCATIONS: Readonly<Record<LocationKey, LocationEquipment>> = {
  hall: { equipment: ["poles"], shared: false },
  arena: { equipment: ["poles"], shared: false },
  lunge: { equipment: [], shared: false },
  jumping: { equipment: ["poles", "jumps"], shared: false },
  groundwork: { equipment: ["poles"], shared: false },
};

/**
 * All five locations: what the server sent, completed with the defaults for locations it left out,
 * the equipment of each in the order of the fixed list.
 */
export function locationsWithDefaults(
  locations: Partial<Record<string, LocationEquipment>> | null | undefined,
  facility: readonly string[] = FACILITY_KEYS,
): Record<LocationKey, LocationEquipment> {
  const out = {} as Record<LocationKey, LocationEquipment>;
  for (const { key } of LOCATIONS) {
    const given = locations?.[key];
    const base = given ?? DEFAULT_LOCATIONS[key];
    out[key] = { equipment: facilityOf(base.equipment, facility), shared: !!base.shared };
  }
  return out;
}

/** Adds the key to the location's equipment or removes it. */
export function toggleEquipment(location: LocationEquipment, key: string, facility: readonly string[] = FACILITY_KEYS): LocationEquipment {
  const has = location.equipment.includes(key);
  const next = has ? location.equipment.filter((k) => k !== key) : [...location.equipment, key];
  return { ...location, equipment: facilityOf(next, facility) };
}

/** "Stangen, Sprünge", "Nichts zum Aufbauen"; with "· geteilt" when the location is shared. */
export function locationSummary(location: LocationEquipment): string {
  const what = equipmentList(location.equipment) || "Nichts zum Aufbauen";
  return location.shared ? `${what} · geteilt` : what;
}

/** Body of PUT /stables/training-equipment: every location with its equipment and `shared`. */
export function equipmentBody(
  locations: Record<LocationKey, LocationEquipment>,
  facility: readonly string[] = FACILITY_KEYS,
): { locations: Record<LocationKey, LocationEquipment> } {
  return { locations: locationsWithDefaults(locations, facility) };
}

/** True when the two sets of locations hold the same equipment and `shared` flags. */
export function sameEquipment(a: Record<LocationKey, LocationEquipment>, b: Record<LocationKey, LocationEquipment>): boolean {
  return LOCATIONS.every(({ key }) => {
    const x = a[key];
    const y = b[key];
    return x.shared === y.shared && x.equipment.length === y.equipment.length && x.equipment.every((k, i) => k === y.equipment[i]);
  });
}

// --- "Aufbau heute": the day's tri-state -----------------------------------------------

export type SetupChoice = "default" | "possible" | "impossible";

export const SETUP_CHOICES: readonly { value: SetupChoice; label: string }[] = [
  { value: "default", label: "Standard" },
  { value: "possible", label: "Möglich" },
  { value: "impossible", label: "Nicht möglich" },
];

/** `equipment_ok` of a day as the choice shown: null (or absent) is the location's default. */
export function setupChoice(equipmentOk: boolean | null | undefined): SetupChoice {
  if (equipmentOk === true) return "possible";
  if (equipmentOk === false) return "impossible";
  return "default";
}

/** The `equipment_ok` a choice stands for: true, false, or null for the default. */
export function equipmentOkOf(choice: SetupChoice): boolean | null {
  if (choice === "possible") return true;
  if (choice === "impossible") return false;
  return null;
}

/** The line under the choice. */
export function setupHint(choice: SetupChoice): string {
  if (choice === "possible") return "Aufbau ist heute erlaubt, auch an einem geteilten Ort.";
  if (choice === "impossible") return "Heute wird ohne Aufbau geplant.";
  return "So, wie es im Stall eingetragen ist.";
}

// --- what is available -----------------------------------------------------------------

export const MISSING_LABEL = "Ausstattung fehlt";

/** An exercise is available unless the API says `available: false` (an absent field counts as available). */
export function isAvailable(item: { available?: boolean | null }): boolean {
  return item.available !== false;
}

/** What a horse is trained with: its discipline and the activities its profile allows (both optional). */
export type HorseUse = { discipline?: string; activities?: readonly string[] };

/**
 * The locations whose activities use the library (hall and arena for dressage, ...); hack has none.
 * For a horse only the locations of its allowed activities count, with its own discipline.
 */
export function locationsOfLibrary(library: string, horse?: HorseUse): LocationKey[] {
  const disciplines = horse?.discipline ? [horse.discipline] : DISCIPLINES.map((d) => d.value);
  return LOCATIONS.filter(
    (l) => (!horse?.activities || horse.activities.includes(l.key)) && disciplines.some((d) => exerciseLibraries(l.key, d).includes(library)),
  ).map((l) => l.key);
}

export type ChipStatus = "ok" | "shared" | "missing";

export type EquipmentChip = {
  key: string;
  label: string;
  status: ChipStatus;
  /** "Stangen", "Stangen · nach Absprache" or "Stangen · fehlt". */
  text: string;
};

/**
 * The equipment chips of an exercise. Personal equipment is always there. Facility equipment is
 * `ok` when a location that uses the library has it without sharing, `shared` when only a shared
 * location has it (set-up needs an agreement), else `missing`. Without data about the stable all
 * chips are `ok`. With a horse only the locations it is trained at count.
 */
export function equipmentChips(
  equipment: readonly string[] | null | undefined,
  stable: StableEquipment | null | undefined,
  library: string,
  horse?: HorseUse,
): EquipmentChip[] {
  const facility = facilityKeysOf(stable);
  const locations = stable ? locationsWithDefaults(stable.locations, facility) : null;
  const where = locationsOfLibrary(library, horse);
  return (equipment ?? []).map((key) => {
    let status: ChipStatus = "ok";
    if (locations && facility.includes(key) && where.length > 0) {
      const has = where.map((l) => locations[l]).filter((loc) => loc.equipment.includes(key));
      if (has.some((loc) => !loc.shared)) status = "ok";
      else status = has.length > 0 ? "shared" : "missing";
    }
    const label = equipmentLabel(key);
    const text = status === "missing" ? `${label} · fehlt` : status === "shared" ? `${label} · nach Absprache` : label;
    return { key, label, status, text };
  });
}

/** "Bei euch fehlt: Stangen, Cavaletti." when chips are missing, else null. */
export function missingText(chips: readonly EquipmentChip[]): string | null {
  const missing = chips.filter((c) => c.status === "missing").map((c) => c.label);
  return missing.length > 0 ? `Bei euch fehlt: ${missing.join(", ")}.` : null;
}
