-- Push notifications go to APNs and FCM directly (JAN-88); the Expo push service is gone.
-- Expo push tokens from older builds cannot be delivered to any more, so they are dropped.
-- The app registers the native device token on its next start.
DELETE FROM push_tokens WHERE token LIKE 'ExponentPushToken[%' OR token LIKE 'ExpoPushToken[%';
