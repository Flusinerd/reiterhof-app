import { router, type Href } from "expo-router";

import { createPushGuard } from "./push-once.ts";

const allow = createPushGuard(600);

/**
 * Leaves every screen of a horse that was deleted: pops the stack to its first screen and replaces that one
 * with Heute, so no screen of the horse stays below (the back button must not lead to a 404). Also right when
 * the app was opened on a screen of the horse by a link and the tabs are not in the stack.
 */
export function leaveDeletedHorse(): void {
  router.dismissAll();
  router.replace("/(tabs)");
}

/**
 * `router.push` for taps on rows and buttons: a second tap on the same target within 600 ms is ignored
 * instead of stacking an identical screen (docs/design-system.md, "Feedback"). Pushes by a notification
 * or an effect keep using `router.push`.
 */
export function pushOnce(href: Href): void {
  if (allow(typeof href === "string" ? href : JSON.stringify(href))) router.push(href);
}
