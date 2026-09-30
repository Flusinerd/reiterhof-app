import { useEffect, useState } from "react";

import { ConsentSheet } from "@/components/consent-prompt";
import { useAuth } from "@/lib/auth";
import { markAsked, useConsents, wasAsked } from "@/lib/consent";
import { needsPrompt, type ConsentKind } from "@/lib/consent-core";

/** Consents asked once after sign-in, before the feature runs without the user's involvement. */
const FIRST_RUN: readonly ConsentKind[] = ["push"];

/**
 * Mount once inside the auth gate. Shows the explanation for each first-run consent (push) once
 * per device; declining is remembered locally so the sheet does not come back at every start.
 * The switches in Einstellungen > Datenschutz stay available.
 */
export function ConsentOnboarding() {
  const { status, hasStable, user } = useAuth();
  const consents = useConsents();
  const [kind, setKind] = useState<ConsentKind | null>(null);
  const items = consents.data?.items;
  // A consent needs a confirmed age (Art. 8); the age screen comes first.
  const ageConfirmed = user?.age_status === "confirmed";

  useEffect(() => {
    if (status !== "signedIn" || !hasStable || !ageConfirmed || !items || kind) return;
    let alive = true;
    void (async () => {
      for (const k of FIRST_RUN) {
        if (!needsPrompt(items, k) || (await wasAsked(k))) continue;
        if (alive) setKind(k);
        return;
      }
    })();
    return () => {
      alive = false;
    };
  }, [status, hasStable, ageConfirmed, items, kind]);

  return (
    <ConsentSheet
      kind={kind}
      onDone={() => {
        if (kind) void markAsked(kind);
        setKind(null);
      }}
    />
  );
}
