import "maplibre-gl/dist/maplibre-gl.css";

import type { GeoJSONSource, Map as MapLibreMap } from "maplibre-gl";
import { createElement, useEffect, useMemo, useRef, useState } from "react";
import { View } from "react-native";

import { Text } from "@/components/ui";
import type { Gait } from "@/lib/gait";
import { colors } from "@/lib/theme";
import { segmentByGait, type GaitPoint } from "@/lib/tracking";

/** Line colors by gait on the light map (canter uses the light-background token). */
export const MAP_GAIT_COLORS: Record<Gait, string> = {
  halt: colors.muted,
  walk: colors.gait.walk,
  trot: colors.gait.trot,
  canter: colors.gait["canter-light"],
};

/**
 * Base map. OpenFreeMap (https://openfreemap.org): free vector tiles from OpenStreetMap data,
 * no API key, no registration, commercial use allowed. It asks for the attribution below, which
 * the map shows. For a bigger user base the operator should self-host or sponsor it; the
 * OpenStreetMap volunteer tile servers (tile.openstreetmap.org) are not meant for an app's
 * regular traffic (https://operations.osmfoundation.org/policies/tiles/), so they are not used.
 * `EXPO_PUBLIC_MAP_STYLE_URL` swaps the style (any MapLibre style JSON, e.g. a self-hosted one).
 */
const STYLE_URL = process.env.EXPO_PUBLIC_MAP_STYLE_URL || "https://tiles.openfreemap.org/styles/positron";
const ATTRIBUTION =
  '<a href="https://openfreemap.org" target="_blank" rel="noopener">OpenFreeMap</a> © <a href="https://openmaptiles.org/" target="_blank" rel="noopener">OpenMapTiles</a> Data from <a href="https://www.openstreetmap.org/copyright" target="_blank" rel="noopener">OpenStreetMap</a>';

const TRACK_SOURCE = "track";
const POSITION_SOURCE = "position";

type Position = [number, number]; // [lon, lat]

function trackData(points: GaitPoint[]): GeoJSON.FeatureCollection {
  return {
    type: "FeatureCollection",
    features: segmentByGait(points).map((s) => ({
      type: "Feature",
      properties: { color: MAP_GAIT_COLORS[s.gait], width: s.gait === "halt" ? 3 : 6 },
      geometry: { type: "LineString", coordinates: s.coords.map((c): Position => [c.lon, c.lat]) },
    })),
  };
}

function positionData(first: GaitPoint | undefined, last: GaitPoint | undefined): GeoJSON.FeatureCollection {
  const feature = (p: GaitPoint, kind: string): GeoJSON.Feature => ({
    type: "Feature",
    properties: { kind },
    geometry: { type: "Point", coordinates: [p.lon, p.lat] satisfies Position },
  });
  return {
    type: "FeatureCollection",
    features: [...(first ? [feature(first, "start")] : []), ...(last ? [feature(last, "now")] : [])],
  };
}

/**
 * Map of the ride (web): MapLibre GL JS with the track as line pieces colored by gait, a start
 * marker, the newest position and the camera following it. Same props as the native map. The
 * screen must show a legend (see `GaitLegend`).
 */
export function TrackingMap({ points, height = 280, follow = true }: { points: GaitPoint[]; height?: number; follow?: boolean }) {
  const container = useRef<HTMLDivElement | null>(null);
  const mapRef = useRef<MapLibreMap | null>(null);
  const [ready, setReady] = useState(false);
  const [failed, setFailed] = useState(false);
  const first = points[0];
  const last = points[points.length - 1];
  const track = useMemo(() => trackData(points), [points]);
  const firstRef = useRef(first);
  firstRef.current = first;

  // Create the map once. maplibre-gl is loaded on demand so the app's main bundle stays small.
  useEffect(() => {
    let cancelled = false;
    let map: MapLibreMap | null = null;
    void (async () => {
      try {
        const maplibre = await import("maplibre-gl");
        // The worker file is copied to public/maplibre/ by scripts/copy-maplibre-worker.mjs.
        maplibre.setWorkerUrl("/maplibre/maplibre-gl-worker.mjs");
        if (cancelled || !container.current) return;
        const start = firstRef.current;
        map = new maplibre.Map({
          container: container.current,
          style: STYLE_URL,
          center: start ? [start.lon, start.lat] : [10.45, 51.16], // Germany until the first fix
          zoom: start ? 16 : 5,
          attributionControl: false,
          dragRotate: false,
          pitchWithRotate: false,
        });
        map.addControl(new maplibre.AttributionControl({ compact: true, customAttribution: ATTRIBUTION }));
        map.on("load", () => {
          if (cancelled || !map) return;
          map.addSource(TRACK_SOURCE, { type: "geojson", data: { type: "FeatureCollection", features: [] } });
          map.addSource(POSITION_SOURCE, { type: "geojson", data: { type: "FeatureCollection", features: [] } });
          map.addLayer({
            id: "track-line",
            type: "line",
            source: TRACK_SOURCE,
            layout: { "line-cap": "round", "line-join": "round" },
            paint: { "line-color": ["get", "color"], "line-width": ["get", "width"] },
          });
          map.addLayer({
            id: "position-dot",
            type: "circle",
            source: POSITION_SOURCE,
            paint: {
              "circle-radius": ["case", ["==", ["get", "kind"], "start"], 6, 7],
              "circle-color": ["case", ["==", ["get", "kind"], "start"], colors.primary.DEFAULT, colors.info.DEFAULT],
              "circle-stroke-color": "#ffffff",
              "circle-stroke-width": 2,
            },
          });
          mapRef.current = map;
          setReady(true);
        });
        map.on("error", (e) => {
          // A failing tile request must not blank the screen; only a style that never loaded is fatal.
          if (!mapRef.current && !cancelled) console.warn("map:", e.error?.message ?? e);
        });
      } catch {
        if (!cancelled) setFailed(true);
      }
    })();
    return () => {
      cancelled = true;
      mapRef.current = null;
      map?.remove();
    };
  }, []);

  // Update the data and move the camera.
  useEffect(() => {
    const map = mapRef.current;
    if (!ready || !map) return;
    (map.getSource(TRACK_SOURCE) as GeoJSONSource | undefined)?.setData(track);
    (map.getSource(POSITION_SOURCE) as GeoJSONSource | undefined)?.setData(positionData(first, last));
    if (follow && last) map.easeTo({ center: [last.lon, last.lat], duration: 500 });
  }, [ready, track, first, last, follow]);

  if (failed) {
    return (
      <View style={{ height }} className="items-center justify-center rounded-card border border-border bg-card px-6">
        <Text variant="secondary">Die Karte konnte nicht geladen werden.</Text>
      </View>
    );
  }

  return (
    <View style={{ height }} className="overflow-hidden rounded-card border border-border bg-card">
      {createElement("div", { ref: container, style: { position: "absolute", inset: 0 } })}
    </View>
  );
}
