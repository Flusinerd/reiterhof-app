import { useQueryClient } from "@tanstack/react-query";
import { useRouter } from "expo-router";
import { useCallback, useRef, useState, type ReactElement } from "react";
import { View } from "react-native";

import { Button, Sheet, Text } from "@/components/ui";
import { CONSENTS_KEY, privacyApi } from "@/lib/api/privacy";
import { ApiError, errorMessage } from "@/lib/api";
import { CONSENT_COPY, consentErrorMessage, needsPrompt, type ConsentKind } from "@/lib/consent-core";

export type ConsentSheetProps = {
  /** The consent to explain; `null` keeps the sheet closed. */
  kind: ConsentKind | null;
  /** `true` after the consent was granted, `false` when the sheet was declined or closed. */
  onDone: (granted: boolean) => void;
};

/**
 * Plain-language explanation shown before the first use of location, camera/photos, push or
 * presence sharing. "Erlauben" records the consent on the server (with the text version the app
 * shows), "Nicht jetzt" changes nothing. "Mehr erfahren" leads to the privacy text.
 */
export function ConsentSheet({ kind, onDone }: ConsentSheetProps) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  // Keep the last content while the sheet slides out.
  const shown = useRef<ConsentKind | null>(null);
  if (kind) shown.current = kind;
  const copy = shown.current ? CONSENT_COPY[shown.current] : null;

  const finish = (granted: boolean) => {
    setError(null);
    onDone(granted);
  };

  async function accept() {
    if (!kind) return;
    setBusy(true);
    setError(null);
    try {
      await privacyApi.setConsent(kind, true);
      await queryClient.invalidateQueries({ queryKey: CONSENTS_KEY });
      finish(true);
    } catch (err) {
      setError((err instanceof ApiError ? consentErrorMessage(err.code) : null) ?? errorMessage(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Sheet open={kind !== null} onOpenChange={(open) => !open && finish(false)} title={copy?.title} description={copy?.summary}>
      {copy ? (
        <>
          <View className="gap-2">
            {copy.points.map((point) => (
              <View key={point} className="flex-row gap-2">
                <Text variant="body">•</Text>
                <Text variant="body" className="flex-1">
                  {point}
                </Text>
              </View>
            ))}
          </View>
          <Text variant="secondary">{copy.declined}</Text>
          {error ? (
            <Text variant="bodySm" tone="danger" accessibilityRole="alert">
              {error}
            </Text>
          ) : null}
          <Button label={copy.accept} fullWidth loading={busy} onPress={() => void accept()} />
          <Button label="Nicht jetzt" variant="outline" fullWidth disabled={busy} onPress={() => finish(false)} />
          <Button
            label="Mehr erfahren"
            variant="ghost"
            fullWidth
            disabled={busy}
            onPress={() => {
              finish(false);
              router.push("/legal/privacy");
            }}
          />
        </>
      ) : null}
    </Sheet>
  );
}

/**
 * Asks for a consent when it is missing, then lets the caller continue.
 *
 * @example
 * const consent = useConsentPrompt();
 * async function takePhoto() {
 *   if (!(await consent.ensure("photos"))) return; // declined
 *   ...
 * }
 * return <>{...}{consent.sheet}</>;
 */
export function useConsentPrompt(): {
  /** `onBeforePrompt` runs right before the sheet opens, e.g. to close another sheet first. */
  ensure: (kind: ConsentKind, onBeforePrompt?: () => void) => Promise<boolean>;
  sheet: ReactElement;
} {
  const queryClient = useQueryClient();
  const [kind, setKind] = useState<ConsentKind | null>(null);
  const resolver = useRef<((granted: boolean) => void) | null>(null);

  const ensure = useCallback(
    async (wanted: ConsentKind, onBeforePrompt?: () => void) => {
      let items;
      try {
        items = (await queryClient.fetchQuery({ queryKey: CONSENTS_KEY, queryFn: privacyApi.consents, staleTime: 0 })).items;
      } catch {
        items = undefined; // offline: ask; the server call reports the problem if it persists
      }
      if (!needsPrompt(items, wanted)) return true;
      onBeforePrompt?.();
      return new Promise<boolean>((resolve) => {
        resolver.current = resolve;
        setKind(wanted);
      });
    },
    [queryClient],
  );

  const sheet = (
    <ConsentSheet
      kind={kind}
      onDone={(granted) => {
        setKind(null);
        resolver.current?.(granted);
        resolver.current = null;
      }}
    />
  );
  return { ensure, sheet };
}
