import { ScrollView, View } from "react-native";

import { Text } from "@/components/ui";
import { hasWeatherDetails, weatherBarFraction, windowLabel, type Weather } from "@/lib/blankets";
import { formatClock } from "@/lib/presence-format";
import { weatherFigures, weatherSentence } from "@/lib/weather-summary";

const BAR_HEIGHT = 44;

/**
 * Forecast for the time a horse is covered (the stable's cover window): one sentence about the
 * rain, three figures in brass (temperature, rain, wind) and an hourly strip with temperature and
 * rain per hour. Sits on the page, no card.
 */
export function WeatherCard({ weather, timeZone }: { weather: Weather; timeZone: string }) {
  const details = hasWeatherDetails(weather);
  const figures = weatherFigures(weather);
  return (
    <View className="gap-3" accessibilityLabel="Wetter im Deckenzeitraum">
      <View className="gap-1">
        <Text variant="eyebrow">{`Deckenzeitraum · ${windowLabel(weather, timeZone)}`}</Text>
        <Text variant="body">{weatherSentence(weather, timeZone)}</Text>
      </View>
      <View className="flex-row border-y border-divider py-3">
        {figures.map((f) => (
          <View
            key={f.label}
            className="flex-1 gap-0.5"
            accessible
            accessibilityLabel={`${f.label} ${f.value} ${f.unit}`}
          >
            <Text variant="caption">{f.label}</Text>
            <View className="flex-row items-baseline gap-1">
              <Text variant="heroNumberSm" tone="brass" className="text-title">
                {f.value}
              </Text>
              <Text variant="secondary">{f.unit}</Text>
            </View>
          </View>
        ))}
      </View>
      {details ? (
        <ScrollView horizontal showsHorizontalScrollIndicator={false} accessibilityLabel="Stündlicher Verlauf">
          <View className="flex-row gap-1">
            {weather.timeline.map((h) => (
              <View
                key={h.time}
                className="w-11 items-center gap-1"
                accessible
                accessibilityLabel={`${formatClock(h.time, timeZone)} Uhr, ${Math.round(h.temp_c)} Grad, ${h.rain_mm} Millimeter Regen`}
              >
                <Text variant="caption">{formatClock(h.time, timeZone).slice(0, 2)}</Text>
                <Text variant="bodyStrong">{Math.round(h.temp_c)}°</Text>
                <View className="justify-end" style={{ height: BAR_HEIGHT }}>
                  <View
                    className="w-3 rounded-pill bg-info"
                    style={{ height: Math.max(h.rain_mm > 0 ? 3 : 0, weatherBarFraction(h.rain_mm) * BAR_HEIGHT) }}
                  />
                </View>
                <Text variant="caption">{h.rain_mm > 0 ? h.rain_mm.toString().replace(".", ",") : ""}</Text>
              </View>
            ))}
          </View>
        </ScrollView>
      ) : null}
    </View>
  );
}
