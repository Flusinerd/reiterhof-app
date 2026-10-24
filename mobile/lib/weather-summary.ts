// Texts of the weather block (WeatherCard): one sentence and three figures instead of a table
// of label/value pairs. No React Native imports, unit-tested. All user-facing text is German.

import { rainIntensity, rainTiming, type Weather } from "./blankets.ts";

/** German decimal comma, at most one decimal: 4.8 -> "4,8", 3 -> "3". */
function num(v: number): string {
  return (Math.round(v * 10) / 10).toString().replace(".", ",");
}

/**
 * "Trocken im ganzen Zeitraum." or "Mäßiger Regen von 22:00 bis 02:00 Uhr, bis 70 % wahrscheinlich."
 * (snapshots without details have no timing and give only the amount).
 */
export function weatherSentence(weather: Weather, timeZone: string): string {
  if (weather.rain_mm < 0.1) return "Trocken im ganzen Zeitraum.";
  const word = rainIntensity(weather.rain_mm);
  const what = word.charAt(0).toUpperCase() + word.slice(1);
  const timing = weather.rain_from && weather.rain_until ? ` von ${rainTiming(weather, timeZone)}` : "";
  return `${what}${timing}, bis ${weather.rain_probability} % wahrscheinlich.`;
}

export type WeatherFigure = { label: string; value: string; unit: string };

/** The three figures in brass: temperature range, rain amount, strongest wind. */
export function weatherFigures(weather: Weather): WeatherFigure[] {
  const lo = Math.round(weather.night_min_c);
  const hi = Math.round(weather.temp_max_c);
  const dry = weather.rain_mm < 0.1;
  return [
    { label: "Temperatur", value: hi > lo ? `${lo}–${hi}` : `${lo}`, unit: "°C" },
    { label: "Regen", value: dry ? "0" : num(weather.rain_mm), unit: "mm" },
    { label: "Wind bis", value: `${Math.round(weather.wind_kmh)}`, unit: "km/h" },
  ];
}
