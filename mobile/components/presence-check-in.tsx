import { useMutation, useQuery, useQueryClient, type UseQueryResult } from "@tanstack/react-query";
import { useState, type ReactElement } from "react";

import { useConsentPrompt } from "@/components/consent-prompt";
import { api, errorMessage, type PresenceVisibility } from "@/lib/api";
import { PRESENCE_EVENT, PRESENCE_KEY, presenceApi, type PresenceOverview } from "@/lib/api/presence";
import { ME_KEY, useAuth } from "@/lib/auth";
import { useInvalidateOnEvents } from "@/lib/realtime";

export type PresenceCheckIn = {
  /** Who is at the stable, kept current through the realtime stream. */
  overview: UseQueryResult<PresenceOverview>;
  /** `true` while I am checked in at the stable. */
  here: boolean;
  visibility: PresenceVisibility;
  /** Checks in (after the presence consent) or out. */
  arriveOrLeave: (arrive: boolean) => Promise<void>;
  /** Asks for the presence consent; declining sets the visibility to hidden. */
  confirmPresenceSharing: () => Promise<void>;
  changeVisibility: (visibility: PresenceVisibility) => void;
  toggling: boolean;
  changingVisibility: boolean;
  error: string | null;
  /** The consent sheet; render it once in the screen. */
  sheet: ReactElement;
};

/**
 * Checking in and out at the stable, shared by the app menu and the presence screen. Being seen
 * at the stable needs the presence consent (JAN-19); declining keeps the person hidden.
 */
export function usePresenceCheckIn(): PresenceCheckIn {
  const { user } = useAuth();
  const consent = useConsentPrompt();
  const queryClient = useQueryClient();
  const [error, setError] = useState<string | null>(null);

  const overview = useQuery({ queryKey: PRESENCE_KEY, queryFn: presenceApi.overview });
  useInvalidateOnEvents({ [PRESENCE_EVENT]: [PRESENCE_KEY] });

  const toggleVisit = useMutation({
    mutationFn: (arrive: boolean) => (arrive ? presenceApi.checkIn("manual") : presenceApi.checkOut()),
    onMutate: () => setError(null),
    onError: (err) => setError(errorMessage(err)),
    onSettled: () => queryClient.invalidateQueries({ queryKey: PRESENCE_KEY }),
  });

  const visibilityMutation = useMutation({
    mutationFn: (visibility: PresenceVisibility) => api.updateMe({ presence_visibility: visibility }),
    onMutate: () => setError(null),
    onSuccess: (updated) => queryClient.setQueryData(ME_KEY, updated),
    onError: (err) => setError(errorMessage(err)),
    onSettled: () => queryClient.invalidateQueries({ queryKey: PRESENCE_KEY }),
  });

  const visibility = (user?.presence_visibility ?? overview.data?.me.visibility ?? "all") as PresenceVisibility;

  async function confirmPresenceSharing(): Promise<void> {
    if (visibility === "hidden") return;
    if (!(await consent.ensure("presence_sharing"))) visibilityMutation.mutate("hidden");
  }

  async function arriveOrLeave(arrive: boolean): Promise<void> {
    if (arrive) await confirmPresenceSharing();
    toggleVisit.mutate(arrive);
  }

  return {
    overview,
    here: overview.data?.me.open_visit != null,
    visibility,
    arriveOrLeave,
    confirmPresenceSharing,
    changeVisibility: (v) => visibilityMutation.mutate(v),
    toggling: toggleVisit.isPending,
    changingVisibility: visibilityMutation.isPending,
    error,
    sheet: consent.sheet,
  };
}
