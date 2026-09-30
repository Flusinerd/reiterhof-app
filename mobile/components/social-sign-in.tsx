import Constants from "expo-constants";
import * as AppleAuthentication from "expo-apple-authentication";
import * as Google from "expo-auth-session/providers/google";
import * as WebBrowser from "expo-web-browser";
import { useEffect, useState } from "react";
import { Platform } from "react-native";

import { Button } from "@/components/ui";
import { ApiError, errorMessage } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { capabilities } from "@/lib/platform";

// Completes the browser session when the app is reopened by the OAuth redirect.
WebBrowser.maybeCompleteAuthSession();

type Extra = { googleWebClientId?: string; googleIosClientId?: string; googleAndroidClientId?: string };
const extra = (Constants.expoConfig?.extra ?? {}) as Extra;

/** Client ID of the current platform; empty until the owner configures it in app.json (expo.extra). */
const platformClientId =
  Platform.OS === "ios" ? extra.googleIosClientId : Platform.OS === "android" ? extra.googleAndroidClientId : extra.googleWebClientId;

type Props = { onError: (message: string | null) => void };

/** "Mit Google anmelden". Without a configured client ID it explains that instead of crashing. */
export function GoogleSignInButton({ onError }: Props) {
  if (!platformClientId) {
    return (
      <Button
        variant="outline"
        label="Mit Google anmelden"
        fullWidth
        onPress={() => onError(errorMessage(new ApiError(0, "not_configured", "")))}
      />
    );
  }
  return <ConfiguredGoogleButton onError={onError} />;
}

function ConfiguredGoogleButton({ onError }: Props) {
  const { signInWithGoogle } = useAuth();
  const [busy, setBusy] = useState(false);
  const [request, response, promptAsync] = Google.useIdTokenAuthRequest({
    webClientId: extra.googleWebClientId,
    iosClientId: extra.googleIosClientId,
    androidClientId: extra.googleAndroidClientId,
  });

  useEffect(() => {
    if (!response) return;
    if (response.type === "success") {
      const idToken = response.params.id_token;
      if (!idToken) {
        onError(errorMessage(null));
        setBusy(false);
        return;
      }
      signInWithGoogle(idToken)
        .catch((e) => onError(errorMessage(e)))
        .finally(() => setBusy(false));
    } else {
      setBusy(false);
      if (response.type === "error") onError(errorMessage(null));
    }
    // signInWithGoogle and onError are stable enough; re-running on them would replay the response
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [response]);

  return (
    <Button
      variant="outline"
      label="Mit Google anmelden"
      fullWidth
      loading={busy}
      disabled={!request}
      onPress={() => {
        onError(null);
        setBusy(true);
        promptAsync().catch(() => setBusy(false));
      }}
    />
  );
}

/** "Mit Apple anmelden": iOS only, renders nothing on Android. */
export function AppleSignInButton({ onError }: Props) {
  if (!capabilities.appleSignIn) return null; // web: Sign in with Apple is not set up
  return <NativeAppleSignInButton onError={onError} />;
}

function NativeAppleSignInButton({ onError }: Props) {
  const { signInWithApple } = useAuth();
  const [available, setAvailable] = useState(false);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (Platform.OS !== "ios") return;
    AppleAuthentication.isAvailableAsync().then(setAvailable, () => setAvailable(false));
  }, []);

  if (!available) return null;

  return (
    <Button
      variant="outline"
      label="Mit Apple anmelden"
      fullWidth
      loading={busy}
      onPress={async () => {
        onError(null);
        setBusy(true);
        try {
          const credential = await AppleAuthentication.signInAsync({
            requestedScopes: [
              AppleAuthentication.AppleAuthenticationScope.FULL_NAME,
              AppleAuthentication.AppleAuthenticationScope.EMAIL,
            ],
          });
          if (!credential.identityToken) throw new Error("Apple returned no identity token");
          // Apple only delivers the name on the very first sign-in, so pass it along now.
          const name = [credential.fullName?.givenName, credential.fullName?.familyName].filter(Boolean).join(" ");
          await signInWithApple(credential.identityToken, name || undefined);
        } catch (e) {
          // ERR_REQUEST_CANCELED: the user closed the sheet, no error message needed
          if ((e as { code?: string }).code !== "ERR_REQUEST_CANCELED") onError(errorMessage(e));
        } finally {
          setBusy(false);
        }
      }}
    />
  );
}
