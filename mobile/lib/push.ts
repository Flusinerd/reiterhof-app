import Constants from "expo-constants";
import * as Device from "expo-device";
import * as Notifications from "expo-notifications";
import { Platform } from "react-native";

export type PushRegistration = {
  /** Expo push token, e.g. `ExponentPushToken[xxxx]`. */
  token: string;
  platform: "ios" | "android";
};

export type PushResult =
  | { ok: true; registration: PushRegistration }
  | { ok: false; reason: "not_a_device" | "permission_denied" | "no_project_id" | "error" };

// Android requires a channel before any notification is shown.
async function ensureAndroidChannel(): Promise<void> {
  if (Platform.OS !== "android") return;
  await Notifications.setNotificationChannelAsync("default", {
    name: "Erinnerungen",
    importance: Notifications.AndroidImportance.DEFAULT,
  });
}

/**
 * Asks for notification permission (if not yet decided) and returns the Expo
 * push token. `useDeviceSetup` sends it to `POST /me/push-tokens`. Never throws; failures are reported through `reason`.
 */
export async function registerForPush(): Promise<PushResult> {
  if (!Device.isDevice) return { ok: false, reason: "not_a_device" };
  if (Platform.OS !== "ios" && Platform.OS !== "android") {
    return { ok: false, reason: "not_a_device" };
  }
  try {
    await ensureAndroidChannel();
    const existing = await Notifications.getPermissionsAsync();
    let status = existing.status;
    if (status !== "granted") {
      status = (await Notifications.requestPermissionsAsync()).status;
    }
    if (status !== "granted") return { ok: false, reason: "permission_denied" };

    const projectId =
      Constants.expoConfig?.extra?.eas?.projectId ?? Constants.easConfig?.projectId;
    if (typeof projectId !== "string" || projectId === "") {
      return { ok: false, reason: "no_project_id" };
    }
    const { data } = await Notifications.getExpoPushTokenAsync({ projectId });
    return { ok: true, registration: { token: data, platform: Platform.OS } };
  } catch {
    return { ok: false, reason: "error" };
  }
}
