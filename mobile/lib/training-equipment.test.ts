import assert from "node:assert/strict";
import { test } from "node:test";

import {
  DEFAULT_LOCATIONS,
  FACILITY_KEYS,
  LOCATIONS,
  MISSING_LABEL,
  PERSONAL_KEYS,
  SETUP_CHOICES,
  equipmentBody,
  equipmentChips,
  equipmentLabel,
  equipmentList,
  equipmentOkOf,
  facilityKeysOf,
  facilityOf,
  isAvailable,
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
  assert.deepEqual(LOCATIONS.map((l) => l.label), ["Halle", "Platz", "Longierhalle", "Springplatz", "Bodenarbeitsplatz"]);
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

test("the PUT keeps the keys of the server's facility list, not only the app's fixed list", () => {
  const server = [...FACILITY_KEYS, "water_tray"];
  const all = locationsWithDefaults(undefined, server);
  const withNew = { ...all, hall: { equipment: ["poles", "water_tray"], shared: false } };
  assert.deepEqual(equipmentBody(withNew, server).locations.hall.equipment, ["poles", "water_tray"]);
  assert.deepEqual(equipmentBody(withNew).locations.hall.equipment, ["poles"], "the fixed list alone would drop it");
});

test("the chips of a horse only look at the places it is trained at", () => {
  const stable: StableEquipment = {
    set: true,
    locations: {
      hall: { equipment: ["cavaletti"], shared: false },
      arena: { equipment: ["cavaletti"], shared: false },
      lunge: { equipment: [], shared: false },
    },
  };
  assert.equal(equipmentChips(["cavaletti"], stable, "cavaletti")[0]!.status, "ok", "any location has them");
  const lungeOnly = { discipline: "dressage", activities: ["lunge"] };
  assert.equal(equipmentChips(["cavaletti"], stable, "cavaletti", lungeOnly)[0]!.status, "missing", "the lunge ring has none");
  assert.deepEqual(locationsOfLibrary("cavaletti", lungeOnly), ["lunge"]);
  assert.deepEqual(locationsOfLibrary("jumping", { discipline: "dressage" }), ["jumping"], "dressage horses use the jumping place only for jumping");
});
