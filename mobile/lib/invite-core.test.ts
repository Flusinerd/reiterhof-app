import assert from "node:assert/strict";
import { test } from "node:test";

import { inviteExpiry, inviteMessage } from "./invite-core.ts";

const invite = { code: "ABCD-EFGH", expires_at: "2026-10-07T22:30:00Z", max_uses: 10 };

test("expiry is the date in the stable's time zone", () => {
  // 22:30 UTC is already 8 October in Berlin.
  assert.equal(inviteExpiry(invite.expires_at, "Europe/Berlin"), "08.10.");
  assert.equal(inviteExpiry(invite.expires_at, "UTC"), "07.10.");
});

test("message names the stable, the address and the code", () => {
  assert.equal(
    inviteMessage(invite, "Stallgasse B", "https://stallfunk.de", "Europe/Berlin"),
    "Komm in den Stall „Stallgasse B“ bei Stallfunk:\n" +
      "1. https://stallfunk.de öffnen und mit deiner E-Mail anmelden\n" +
      "2. Einladungscode eingeben: ABCD-EFGH\n" +
      "Gültig bis 08.10.",
  );
});
