-- Consent for the map display (JAN-85). The ride map loads its tiles from Apple Maps (iOS),
-- Google Maps (Android) or OpenFreeMap (web); the provider sees the viewer's IP address and
-- the map area, which is where the ride took place. The app asks before it loads a map.
ALTER TABLE consents DROP CONSTRAINT consents_kind_check;
ALTER TABLE consents ADD CONSTRAINT consents_kind_check
    CHECK (kind IN ('location_geofence', 'location_tracking', 'maps', 'presence_sharing', 'photos', 'push'));
