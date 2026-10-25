import { router, type Href } from "expo-router";

import { createPushGuard } from "./push-once.ts";

const allow = createPushGuard(600);

/**
 * `router.push` for taps on rows and buttons: a second tap on the same target within 600 ms is ignored
 * instead of stacking an identical screen (docs/design-system.md, "Feedback"). Pushes by a notification
 * or an effect keep using `router.push`.
 */
export function pushOnce(href: Href): void {
  if (allow(typeof href === "string" ? href : JSON.stringify(href))) router.push(href);
}
