import { useLocalSearchParams, useRouter } from "expo-router";
import { useEffect, useRef, useState } from "react";

import { Button, Card, Hero, Input, Screen, SectionLabel, Text } from "@/components/ui";
import { api, errorMessage } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import {
  RESEND_COOLDOWN_SECONDS,
  cooldownRemainingSeconds,
  formatLoginCode,
  isCompleteLoginCode,
  loginCodeErrorMessage,
  normalizeLoginCode,
  resendLabel,
} from "@/lib/login-code";

/**
 * Shown after "Code senden". The mail carries a 6-digit code and the magic link. The code is
 * the way in when the link opens in another app or browser than the one the user signs in to
 * (an installed PWA on iOS has separate storage from Safari); the link keeps working through
 * `app/auth/verify.tsx`.
 */
export default function CheckEmail() {
  const router = useRouter();
  const { email } = useLocalSearchParams<{ email?: string }>();
  const { signInWithCode } = useAuth();
  const [code, setCode] = useState("");
  const [signingIn, setSigningIn] = useState(false);
  const [codeError, setCodeError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<{ text: string; error: boolean } | null>(null);
  const submitting = useRef(false);
  // The mail was requested just before this screen opened.
  const [availableAt, setAvailableAt] = useState(() => Date.now() + RESEND_COOLDOWN_SECONDS * 1000);
  const [secondsLeft, setSecondsLeft] = useState(RESEND_COOLDOWN_SECONDS);

  useEffect(() => {
    const tick = () => setSecondsLeft(cooldownRemainingSeconds(availableAt, Date.now()));
    tick();
    const id = setInterval(tick, 1000);
    return () => clearInterval(id);
  }, [availableAt]);

  async function submit(value: string) {
    if (!email || submitting.current || !isCompleteLoginCode(value)) return;
    submitting.current = true;
    setSigningIn(true);
    setCodeError(null);
    try {
      // The root layout moves on to the join screen or the tabs once the session exists.
      await signInWithCode(email, value);
    } catch (e) {
      setCodeError(loginCodeErrorMessage(e));
      setCode("");
    } finally {
      submitting.current = false;
      setSigningIn(false);
    }
  }

  function onChangeCode(input: string) {
    const digits = normalizeLoginCode(input);
    setCode(digits);
    setCodeError(null);
    if (isCompleteLoginCode(digits)) void submit(digits);
  }

  async function resend() {
    if (!email || secondsLeft > 0) return;
    setBusy(true);
    setMessage(null);
    try {
      await api.requestMagicLink(email);
      setAvailableAt(Date.now() + RESEND_COOLDOWN_SECONDS * 1000);
      setCode("");
      setCodeError(null);
      setMessage({ text: "Neuer Code und Link sind unterwegs. Der alte Code ist ungültig.", error: false });
    } catch (e) {
      setMessage({ text: errorMessage(e), error: true });
    } finally {
      setBusy(false);
    }
  }

  return (
    <Screen back keyboardShouldPersistTaps="handled">
      <Hero title="Postfach prüfen" description="Code eintippen oder Link in der Mail antippen." />

      {email ? (
        <>
          <SectionLabel>Code</SectionLabel>
          <Card className="gap-3">
            <Input
              value={formatLoginCode(code)}
              onChangeText={onChangeCode}
              placeholder="123 456"
              accessibilityLabel="6-stelliger Code"
              keyboardType="number-pad"
              inputMode="numeric"
              textContentType="oneTimeCode"
              autoComplete="one-time-code"
              autoCapitalize="none"
              autoCorrect={false}
              autoFocus
              className="h-14 text-center text-title"
              style={{ letterSpacing: 4 }}
            />
            <Button
              label="Anmelden"
              fullWidth
              loading={signingIn}
              disabled={!isCompleteLoginCode(code)}
              onPress={() => submit(code)}
            />
            {codeError ? (
              <Text variant="bodySm" tone="danger" accessibilityRole="alert">
                {codeError}
              </Text>
            ) : null}
            <Text variant="bodySm" tone="muted">
              Gesendet an {email}. 15 Minuten gültig.
            </Text>
          </Card>
        </>
      ) : null}

      <Card className="gap-3">
        <Text variant="bodySm" tone="muted">
          Nichts angekommen? Schau im Spam-Ordner.
        </Text>
        <Button
          variant="secondary"
          label={resendLabel(secondsLeft)}
          fullWidth
          loading={busy}
          disabled={secondsLeft > 0}
          onPress={resend}
        />
        <Button variant="ghost" label="Andere Adresse" fullWidth onPress={() => router.back()} />
        {message ? (
          <Text variant="bodySm" tone={message.error ? "danger" : "primary"} accessibilityRole={message.error ? "alert" : undefined}>
            {message.text}
          </Text>
        ) : null}
      </Card>
    </Screen>
  );
}
