import { Platform } from "react-native";

import { capabilitiesFor, type PlatformOS } from "./platform-core.ts";

export const platformOS = Platform.OS as PlatformOS;
export const isWeb = Platform.OS === "web";
/** What this platform supports, see `platform-core.ts`. */
export const capabilities = capabilitiesFor(platformOS);
