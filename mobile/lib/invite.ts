import Constants from "expo-constants";
import { Platform, Share } from "react-native";

export { inviteExpiry, inviteMessage, type Invite } from "./invite-core";

/** Address of the web app: the page itself on web, expo.extra.webUrl in the native app. */
export function webUrl(): string {
  if (Platform.OS === "web" && typeof window !== "undefined") return window.location.origin;
  return ((Constants.expoConfig?.extra ?? {}) as { webUrl?: string }).webUrl ?? "https://stallfunk.de";
}

/**
 * Opens the share sheet with the invite text. Browsers without the Web Share API get the
 * text on the clipboard instead. Returns what happened, for a short confirmation.
 */
export async function shareInvite(message: string): Promise<"shared" | "copied" | "dismissed"> {
  if (Platform.OS === "web") {
    const nav = typeof navigator !== "undefined" ? navigator : undefined;
    if (nav && typeof nav.share === "function") {
      try {
        await nav.share({ text: message });
        return "shared";
      } catch (e) {
        if (e instanceof Error && e.name === "AbortError") return "dismissed";
      }
    }
    await nav?.clipboard?.writeText(message);
    return "copied";
  }
  const result = await Share.share({ message });
  return result.action === Share.dismissedAction ? "dismissed" : "shared";
}
