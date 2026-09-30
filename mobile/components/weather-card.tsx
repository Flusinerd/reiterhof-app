import { ScrollView, View } from "react-native";

import { Card, Divider, Text } from "@/components/ui";
import {
  hasWeatherDetails,
  weatherBarFraction,
  weatherFacts,
  windowLabel,
  type Weather,
} from "@/lib/blankets";
import { formatClock } from "@/lib/presence-format";

const BAR_HEIGHT = 44;

/**
 * Forecast for the time a horse is covered (evening until about 12:30): temperature range,
 * rain amount and timing, wind, plus an hourly strip with temperature and rain per hour.
 */
export function WeatherCard({ weather, timeZone }: { weather: Weather; timeZone: string }) {
  const details = hasWeatherDetails(weather);
  return (
    <Card className="gap-4">
      <View className="gap-1">
        <Text variant="bodyStrong">Wetter im Deckenzeitraum</Text>
        <Text variant="secondary">{windowLabel(weather, timeZone)}</Text>
      </View>
      <View className="gap-2">
        {weatherFacts(weather, timeZone).map((f, i) => (
          <View key={f.label}>
            {i > 0 ? <Divider className="mb-2" /> : null}
            <View className="flex-row items-baseline justify-between gap-4">
              <Text variant="secondary">{f.label}</Text>
              <Text variant="bodyStrong" className="flex-shrink text-right">
                {f.value}
              </Text>
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
                    className="w-3 rounded-pill bg-primary"
                    style={{ height: Math.max(h.rain_mm > 0 ? 3 : 0, weatherBarFraction(h.rain_mm) * BAR_HEIGHT) }}
                  />
                </View>
                <Text variant="caption">{h.rain_mm > 0 ? h.rain_mm.toString().replace(".", ",") : ""}</Text>
              </View>
            ))}
          </View>
        </ScrollView>
      ) : null}
    </Card>
  );
}
