import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import * as storage from "./storage";

import { CONSENTS_KEY, privacyApi } from "./api/privacy";
import { useAuth } from "./auth";
import { isGranted, type ConsentKind } from "./consent-core.ts";
import { disableGeofence } from "./geofence";

/** The consents of the signed-in user (`GET /me/consents`), refetched after every change. */
export function useConsents() {
  const { status } = useAuth();
  return useQuery({
    queryKey: CONSENTS_KEY,
    queryFn: privacyApi.consents,
    enabled: status === "signedIn",
  });
}

/** True/false once loaded, `undefined` while unknown. */
export function useHasConsent(kind: ConsentKind): boolean | undefined {
  const { data } = useConsents();
  return data ? isGranted(data.items, kind) : undefined;
}

/**
 * Grants or revokes one consent. A revocation also switches off what runs on this device:
 * the geofence stops when the location consent is withdrawn. (Push tokens and the presence
 * visibility are cleaned up by the server.)
 */
export function useSetConsent() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ kind, granted }: { kind: ConsentKind; granted: boolean }) => {
      const item = await privacyApi.setConsent(kind, granted);
      if (!granted && kind === "location_geofence") await disableGeofence();
      return item;
    },
    onSettled: () => queryClient.invalidateQueries({ queryKey: CONSENTS_KEY }),
  });
}

// --- "asked once" flags -----------------------------------------------------------------------
// The server keeps no row for a consent that was declined without ever being granted, so the
// first-run sheets remember per device that they were shown.

const askedKey = (kind: ConsentKind) => `reiterhof.consent.asked.${kind}`;

export async function wasAsked(kind: ConsentKind): Promise<boolean> {
  try {
    return (await storage.getItem(askedKey(kind))) === "1";
  } catch {
    return false;
  }
}

export async function markAsked(kind: ConsentKind): Promise<void> {
  try {
    await storage.setItem(askedKey(kind), "1");
  } catch {
    // shown again at the next start; harmless
  }
}
