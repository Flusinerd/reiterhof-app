// API call and react-query hook for the official DWD weather warnings
// (backend: internal/weather; docs/domains/blankets.md, "Weather warnings").

import { useQuery } from "@tanstack/react-query";

import { authed } from "../api";
import type { WarningsResponse } from "../weather-warnings.ts";

export const weatherKeys = {
  warnings: ["weather", "warnings"] as const,
};

export const weatherApi = {
  warnings: () => authed.get<WarningsResponse>("/api/v1/weather/warnings"),
};

/** Active warnings of the stable, most severe first. The job refreshes them every 15 minutes. */
export const useWeatherWarnings = () =>
  useQuery({ queryKey: weatherKeys.warnings, queryFn: weatherApi.warnings, staleTime: 5 * 60 * 1000 });
