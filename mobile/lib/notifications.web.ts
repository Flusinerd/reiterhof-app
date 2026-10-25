/**
 * Web variant of `lib/notifications.ts`. expo-notifications has no notification-response API on
 * web (`getLastNotificationResponse` throws), so nothing is registered here. A tapped web push
 * opens its screen through the service worker's `notificationclick` handler (`public/sw.js`),
 * which opens or focuses the app at the target URL; the router then shows that route.
 */
export function useNotificationNavigation(): void {}

/**
 * Web variant of `useRefreshOnPush`: a web push is shown by the service worker and does not reach the
 * page; the reminder center catches up when the tab is focused again (refetch on focus).
 */
export function useRefreshOnPush(): void {}
