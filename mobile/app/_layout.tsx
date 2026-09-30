import "../global.css";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useFonts } from "expo-font";
import { Redirect, Stack, useSegments } from "expo-router";
import * as SplashScreen from "expo-splash-screen";
import { StatusBar } from "expo-status-bar";
import { useEffect } from "react";
import { ActivityIndicator, View } from "react-native";
import { SafeAreaProvider } from "react-native-safe-area-context";

import { ConsentOnboarding } from "@/components/consent-onboarding";
import { Button, Screen, Text } from "@/components/ui";
import { AuthProvider, useAuth } from "@/lib/auth";
import { fontAssets } from "@/lib/fonts";
import "@/lib/geofence"; // registers the background geofence task at app start
import "@/lib/tracking-location"; // registers the ride tracking task at app start
import { useNotificationNavigation } from "@/lib/notifications"; // also sets the foreground presentation
import { useDeviceSetup } from "@/lib/use-push-registration";
import { colors } from "@/lib/theme";

// Keep the splash screen until the fonts are ready (no flash of system font).
SplashScreen.preventAutoHideAsync();

const queryClient = new QueryClient();

export default function RootLayout() {
  const [fontsLoaded, fontError] = useFonts(fontAssets);
  const ready = fontsLoaded || fontError !== null;

  useEffect(() => {
    if (ready) SplashScreen.hideAsync();
  }, [ready]);

  if (!ready) return null;

  return (
    <SafeAreaProvider>
      <QueryClientProvider client={queryClient}>
        <AuthProvider>
          <StatusBar style="dark" />
          <AuthGate />
        </AuthProvider>
      </QueryClientProvider>
    </SafeAreaProvider>
  );
}

/**
 * Sends the user where they belong:
 * no session -> sign-in, session without stable -> join, otherwise the tabs.
 * `(auth)/*` and `auth/verify` (the magic link target) are the only screens without a stable.
 */
function AuthGate() {
  const { status, hasStable, refresh } = useAuth();
  useDeviceSetup(); // push token registration + geofence re-arm after sign-in
  useNotificationNavigation(); // tapping a push opens its screen
  const segments = useSegments() as string[];
  const inAuthGroup = segments[0] === "(auth)";
  const inVerify = segments[0] === "auth";
  const inLegal = segments[0] === "legal"; // privacy text and imprint: readable before sign-in
  const inSettings = segments[0] === "settings"; // privacy settings: export and deletion need no stable

  const stack = (
    <Stack screenOptions={{ headerShown: false, contentStyle: { backgroundColor: colors.background } }} />
  );

  if (status === "loading") {
    return (
      <View className="flex-1 items-center justify-center bg-background">
        <ActivityIndicator />
      </View>
    );
  }
  if (status === "error") {
    return (
      <Screen scroll={false} className="justify-center">
        <Text variant="title">Server nicht erreichbar</Text>
        <Text variant="body" tone="muted">
          Bitte prüfe deine Internetverbindung und versuche es erneut.
        </Text>
        <Button label="Erneut versuchen" onPress={refresh} />
      </Screen>
    );
  }

  let redirect = null;
  if (status === "signedOut") {
    if (!inAuthGroup && !inVerify && !inLegal) redirect = <Redirect href="/(auth)/sign-in" />;
  } else if (!hasStable) {
    if (!(inAuthGroup && segments[1] === "join") && !inLegal && !inSettings) redirect = <Redirect href="/(auth)/join" />;
  } else if (inAuthGroup || inVerify) {
    redirect = <Redirect href="/(tabs)" />;
  }

  return (
    <>
      {stack}
      {redirect}
      <ConsentOnboarding />
    </>
  );
}
