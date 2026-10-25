import assert from "node:assert/strict";
import { test } from "node:test";

import {
  DEFAULT_LOCATIONS,
  FACILITY_KEYS,
  LOCATIONS,
  MISSING_LABEL,
  PERSONAL_KEYS,
  SETUP_CHOICES,
  availableSet,
  equipmentBody,
  equipmentChips,
  equipmentLabel,
  equipmentList,
  equipmentOkOf,
  equipmentOverview,
  facilityKeysOf,
  facilityOf,
  isAvailable,
  locationLabel,
  locationSummary,
  locationsOfLibrary,
  locationsWithDefaults,
  missingText,
  sameEquipment,
  setupChoice,
  setupHint,
  toggleEquipment,
  type StableEquipment,
} from "./training-equipment.ts";

test("every key of the fixed lists has a German label", () => {
  assert.deepEqual(FACILITY_KEYS.map(equipmentLabel), ["Stangen", "Cavaletti", "Sprünge", "Gelassenheitsparcours"]);
  assert.deepEqual(PERSONAL_KEYS.map(equipmentLabel), [
    "Longe",
    "Ausbindezügel",
    "Doppellonge",
    "Kappzaum",
    "Plane",
    "Schirm",
    "Hütchen",
    "Flatterband",
    "Gerte",
    "Halsring",
    "Anhänger",
  ]);
  assert.equal(equipmentLabel("flutter_curtain"), "Flutter curtain", "unknown keys stay readable");
  assert.equal(equipmentLabel(""), "");
  assert.equal(equipmentList(["poles", "jumps"]), "Stangen, Sprünge");
  assert.equal(equipmentList([]), "");
  assert.equal(equipmentList(undefined), "");
});

test("the five locations are the activities with a place; labels are German", () => {
  assert.deepEqual(LOCATIONS.map((l) => l.key), ["hall", "arena", "lunge", "jumping", "groundwork"]);
  assert.equal(locationLabel("hall"), "Halle");
  assert.equal(locationLabel("arena"), "Platz");
  assert.equal(locationLabel("lunge"), "Longierhalle");
  assert.equal(locationLabel("jumping"), "Springplatz");
  assert.equal(locationLabel("groundwork"), "Bodenarbeitsplatz");
  assert.equal(locationLabel("hack"), "hack");
});

test("locations without data take the backend's defaults, in the order of the fixed list", () => {
  assert.deepEqual(DEFAULT_LOCATIONS.hall, { equipment: ["poles"], shared: false });
  const all = locationsWithDefaults(undefined);
  assert.deepEqual(all.hall, { equipment: ["poles"], shared: false });
  assert.deepEqual(all.arena, { equipment: ["poles"], shared: false });
  assert.deepEqual(all.lunge, { equipment: [], shared: false });
  assert.deepEqual(all.jumping, { equipment: ["poles", "jumps"], shared: false });
  assert.deepEqual(all.groundwork, { equipment: ["poles"], shared: false });
  const given = locationsWithDefaults({ hall: { equipment: ["jumps", "cavaletti", "poles", "gibtsnicht"], shared: true } });
  assert.deepEqual(given.hall, { equipment: ["poles", "cavaletti", "jumps"], shared: true });
  assert.deepEqual(given.arena, DEFAULT_LOCATIONS.arena, "a location the server left out takes its default");
  assert.deepEqual(locationsWithDefaults({ lunge: { equipment: [], shared: false } }).lunge, { equipment: [], shared: false });
});

test("the admin toggles the equipment of a location and sees a summary", () => {
  const hall = { equipment: ["poles"], shared: false };
  assert.deepEqual(toggleEquipment(hall, "cavaletti").equipment, ["poles", "cavaletti"]);
  assert.deepEqual(toggleEquipment({ equipment: ["poles", "jumps"], shared: false }, "cavaletti").equipment, ["poles", "cavaletti", "jumps"]);
  assert.deepEqual(toggleEquipment(hall, "poles").equipment, []);
  assert.equal(toggleEquipment({ ...hall, shared: true }, "jumps").shared, true);
  assert.deepEqual(hall, { equipment: ["poles"], shared: false }, "the input stays untouched");
  assert.equal(locationSummary({ equipment: ["poles", "cavaletti"], shared: false }), "Stangen, Cavaletti");
  assert.equal(locationSummary({ equipment: [], shared: false }), "Nichts zum Aufbauen");
  assert.equal(locationSummary({ equipment: ["poles"], shared: true }), "Stangen · geteilt");
  assert.equal(locationSummary({ equipment: [], shared: true }), "Nichts zum Aufbauen · geteilt");
  assert.equal(
    equipmentOverview(locationsWithDefaults(undefined)),
    "Halle: Stangen · Platz: Stangen · Longierhalle: nichts · Springplatz: Stangen, Sprünge · Bodenarbeitsplatz: Stangen",
  );
});

test("the body of the PUT names every location; changes are detected", () => {
  const all = locationsWithDefaults(undefined);
  assert.deepEqual(equipmentBody(all), { locations: all });
  assert.deepEqual(Object.keys(equipmentBody(all).locations), ["hall", "arena", "lunge", "jumping", "groundwork"]);
  assert.equal(sameEquipment(all, locationsWithDefaults(undefined)), true);
  assert.equal(sameEquipment(all, { ...all, lunge: { equipment: ["poles"], shared: false } }), false);
  assert.equal(sameEquipment(all, { ...all, hall: { ...all.hall, shared: true } }), false);
  assert.equal(facilityKeysOf(undefined).length, 4);
  assert.deepEqual(facilityKeysOf({ keys: { facility: ["poles", "jumps"] } }), ["poles", "jumps"]);
  assert.deepEqual(facilityKeysOf({ keys: { facility: [] } }), FACILITY_KEYS);
  assert.deepEqual(facilityOf(["lunge", "jumps", "poles"]), ["poles", "jumps"], "personal keys are not facility keys");
  assert.deepEqual(facilityOf(undefined), []);
});

test("the tri-state of a day maps to equipment_ok and back", () => {
  assert.deepEqual(SETUP_CHOICES.map((c) => c.label), ["Standard", "Möglich", "Nicht möglich"]);
  assert.equal(setupChoice(null), "default");
  assert.equal(setupChoice(undefined), "default");
  assert.equal(setupChoice(true), "possible");
  assert.equal(setupChoice(false), "impossible");
  assert.equal(equipmentOkOf("default"), null);
  assert.equal(equipmentOkOf("possible"), true);
  assert.equal(equipmentOkOf("impossible"), false);
  for (const c of SETUP_CHOICES) assert.equal(setupChoice(equipmentOkOf(c.value)), c.value);
  assert.match(setupHint("default"), /eingetragen/);
  assert.match(setupHint("possible"), /erlaubt/);
  assert.match(setupHint("impossible"), /ohne Aufbau/);
});

test("exercises are available unless the API says they are not", () => {
  assert.equal(MISSING_LABEL, "Ausstattung fehlt");
  assert.equal(isAvailable({ available: true }), true);
  assert.equal(isAvailable({ available: false }), false);
  assert.equal(isAvailable({}), true, "older backend: no field, nothing is dimmed");
  assert.equal(isAvailable({ available: null }), true);
});

test("the available set follows the day's choice and the sharing of the location", () => {
  const hall = { equipment: ["poles", "cavaletti"], shared: false };
  const shared = { equipment: ["poles", "cavaletti"], shared: true };
  assert.deepEqual(availableSet(hall, null), ["poles", "cavaletti"]);
  assert.deepEqual(availableSet(hall, true), ["poles", "cavaletti"]);
  assert.deepEqual(availableSet(hall, false), [], "no set-up today");
  assert.deepEqual(availableSet(shared, null), [], "a shared place needs an agreement");
  assert.deepEqual(availableSet(shared, undefined), []);
  assert.deepEqual(availableSet(shared, true), ["poles", "cavaletti"], "...which the day can state");
  assert.deepEqual(availableSet(shared, false), []);
  assert.deepEqual(availableSet(undefined, true), []);
  assert.deepEqual(availableSet({ equipment: ["lunge", "poles"], shared: false }, null), ["poles"], "only facility equipment is planned against the stable");
});

test("which locations use a library", () => {
  assert.deepEqual(locationsOfLibrary("dressage"), ["hall", "arena"]);
  assert.deepEqual(locationsOfLibrary("cavaletti"), ["hall", "arena", "lunge", "jumping"]);
  assert.deepEqual(locationsOfLibrary("jumping"), ["hall", "arena", "jumping"]);
  assert.deepEqual(locationsOfLibrary("western"), ["hall", "arena"]);
  assert.deepEqual(locationsOfLibrary("lunge"), ["lunge"]);
  assert.deepEqual(locationsOfLibrary("groundwork"), ["groundwork"]);
  assert.deepEqual(locationsOfLibrary("hack"), []);
  assert.deepEqual(locationsOfLibrary("unbekannt"), []);
});

test("equipment chips mark what the stable lacks and what needs an agreement", () => {
  const stable: StableEquipment = {
    set: true,
    locations: {
      hall: { equipment: ["poles", "cavaletti"], shared: true },
      arena: { equipment: ["poles"], shared: false },
      lunge: { equipment: [], shared: false },
      jumping: { equipment: ["jumps", "poles"], shared: false },
      groundwork: { equipment: ["poles"], shared: false },
    },
  };
  // Cavaletti are used in hall, arena, lunge and jumping: only the shared hall has them.
  const cav = equipmentChips(["cavaletti", "lunge"], stable, "cavaletti");
  assert.deepEqual(cav.map((c) => [c.key, c.status, c.text]), [
    ["cavaletti", "shared", "Cavaletti · nach Absprache"],
    ["lunge", "ok", "Longe"],
  ]);
  // Poles: the arena has them without sharing.
  assert.equal(equipmentChips(["poles"], stable, "cavaletti")[0]!.status, "ok");
  // Jumps in the dressage library's places (hall, arena): nowhere.
  const jumps = equipmentChips(["jumps"], stable, "dressage")[0]!;
  assert.equal(jumps.status, "missing");
  assert.equal(jumps.text, "Sprünge · fehlt");
  assert.equal(equipmentChips(["jumps"], stable, "jumping")[0]!.status, "ok", "the jumping place has them");
  assert.equal(missingText(equipmentChips(["poles", "jumps", "obstacles"], stable, "dressage")), "Bei euch fehlt: Sprünge, Gelassenheitsparcours.");
  assert.equal(missingText(cav), null, "shared is not missing");
  assert.equal(missingText([]), null);
  // A library without a place (hack) and unknown data: nothing is claimed missing.
  assert.equal(equipmentChips(["obstacles"], stable, "hack")[0]!.status, "ok");
  assert.equal(equipmentChips(["poles", "jumps"], undefined, "dressage").every((c) => c.status === "ok"), true);
  assert.equal(equipmentChips(["poles"], null, "dressage")[0]!.text, "Stangen");
  assert.deepEqual(equipmentChips(undefined, stable, "dressage"), []);
  assert.deepEqual(equipmentChips([], stable, "dressage"), []);
  // Locations the server left out take the defaults: the arena has poles.
  assert.equal(equipmentChips(["poles"], { set: false, locations: {} }, "dressage")[0]!.status, "ok");
});
