import { useEffect, useMemo, useRef } from "react";
import { View } from "react-native";
import MapView, { Marker, Polyline } from "react-native-maps";

import { colors } from "@/lib/theme";
import type { Gait } from "@/lib/gait";
import { segmentByGait, type GaitPoint } from "@/lib/tracking";

/** Line colors by gait on the light map (canter uses the light-background token). */
export const MAP_GAIT_COLORS: Record<Gait, string> = {
  halt: colors.muted,
  walk: colors.gait.walk,
  trot: colors.gait.trot,
  canter: colors.gait["canter-light"],
};

/**
 * Map of the ride: the track as polyline pieces colored by gait, the start marker and the
 * camera following the newest point. The screen must show a legend (see `GaitLegend`).
 */
export function TrackingMap({ points, height = 280, follow = true }: { points: GaitPoint[]; height?: number; follow?: boolean }) {
  const ref = useRef<MapView>(null);
  const segments = useMemo(() => segmentByGait(points), [points]);
  const first = points[0];
  const last = points[points.length - 1];

  useEffect(() => {
    if (!follow || !last) return;
    ref.current?.animateCamera({ center: { latitude: last.lat, longitude: last.lon } }, { duration: 500 });
  }, [follow, last?.lat, last?.lon]);

  return (
    <View style={{ height }} className="overflow-hidden rounded-card border border-border bg-card">
      <MapView
        ref={ref}
        style={{ flex: 1 }}
        showsUserLocation
        showsCompass={false}
        toolbarEnabled={false}
        initialCamera={
          first ? { center: { latitude: first.lat, longitude: first.lon }, zoom: 16, heading: 0, pitch: 0, altitude: 800 } : undefined
        }
      >
        {segments.map((s, i) => (
          <Polyline
            key={i}
            coordinates={s.coords.map((c) => ({ latitude: c.lat, longitude: c.lon }))}
            strokeColor={MAP_GAIT_COLORS[s.gait]}
            strokeWidth={s.gait === "halt" ? 3 : 6}
            lineCap="round"
            lineJoin="round"
          />
        ))}
        {first ? <Marker coordinate={{ latitude: first.lat, longitude: first.lon }} title="Start" pinColor={colors.primary.DEFAULT} /> : null}
      </MapView>
    </View>
  );
}
