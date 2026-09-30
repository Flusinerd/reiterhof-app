/**
 * Web variant of `lib/notifications.ts`. expo-notifications has no notification-response API on
 * web (`getLastNotificationResponse` throws), so nothing is registered here. A tapped web push
 * opens its screen through the service worker's `notificationclick` handler (`public/sw.js`),
 * which opens or focuses the app at the target URL; the router then shows that route.
 */
export function useNotificationNavigation(): void {}
