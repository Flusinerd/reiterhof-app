import { View } from "react-native";
import Svg, { G, Path, Rect, Text as SvgText } from "react-native-svg";

import { Pill, Text } from "@/components/ui";
import { BODY_PARTS, bodyPartLabel, bodyPartShort } from "@/lib/observations";
import { colors } from "@/lib/theme";

// Simple side view of a horse looking to the left. Every region is tappable. The legs carry a
// caption (VL = vorne links, VR = vorne rechts, HL = hinten links, HR = hinten rechts) because
// left and right are ambiguous in a side view. The chips below are the accessible alternative
// (and offer "Sonstiges").
const REGIONS: { key: string; d: string }[] = [
  // Head and neck.
  { key: "head", d: "M14 62 L30 24 Q34 16 44 18 L84 34 L98 44 L92 78 L66 86 L36 88 Q16 84 14 62 Z" },
  // Back: top part of the body.
  { key: "back", d: "M98 44 L124 38 L206 40 Q222 42 224 58 L100 60 Z" },
  // Belly: lower part of the body.
  { key: "belly", d: "M100 60 L224 58 Q226 74 216 92 L104 96 Q96 78 100 60 Z" },
];

// Legs are 24 wide (about 34 px on a phone) so they are easy to hit.
const LEGS: { key: string; x: number; y: number; far: boolean }[] = [
  { key: "front_right", x: 92, y: 96, far: true },
  { key: "front_left", x: 122, y: 96, far: false },
  { key: "hind_right", x: 170, y: 96, far: true },
  { key: "hind_left", x: 200, y: 96, far: false },
];

/**
 * Body part choice (JAN-50): horse pictogram plus chips. Tapping the selected region again
 * clears the choice. `value` is a body part key or null.
 */
export function ObservationBodyPicker({
  value,
  onChange,
}: {
  value: string | null;
  onChange: (value: string | null) => void;
}) {
  const toggle = (key: string) => onChange(value === key ? null : key);
  const fill = (key: string) => (value === key ? colors.primary.DEFAULT : colors.primary.soft);
  return (
    <View className="gap-3">
      <View className="items-center rounded-tile border border-border bg-card py-3">
        <Svg width="100%" height={170} viewBox="0 0 240 170" accessibilityLabel="Pferd, Stelle antippen">
          {REGIONS.map((r) => (
            <Path
              key={r.key}
              d={r.d}
              fill={fill(r.key)}
              stroke={colors.primary.deep}
              strokeWidth={2}
              strokeLinejoin="round"
              onPress={() => toggle(r.key)}
            />
          ))}
          {LEGS.map((l) => (
            <G key={l.key} onPress={() => toggle(l.key)}>
              <Rect
                x={l.x}
                y={l.y}
                width={24}
                height={54}
                rx={6}
                fill={fill(l.key)}
                stroke={colors.primary.deep}
                strokeWidth={2}
                strokeDasharray={l.far ? "4 3" : undefined}
              />
              <SvgText
                x={l.x + 12}
                y={l.y + 33}
                fontSize={12}
                fontWeight="bold"
                textAnchor="middle"
                fill={value === l.key ? colors.card : colors.primary.deeper}
              >
                {bodyPartShort(l.key)}
              </SvgText>
            </G>
          ))}
          <SvgText x={120} y={20} fontSize={11} textAnchor="middle" fill={colors.muted}>
            gestrichelt = abgewandte Seite
          </SvgText>
        </Svg>
        <Text variant="secondary" tone="muted">
          {value ? `Ausgewählt: ${bodyPartLabel(value)}` : "Stelle antippen (optional)"}
        </Text>
      </View>
      <View className="flex-row flex-wrap gap-2">
        {BODY_PARTS.map((key) => (
          <Pill key={key} label={bodyPartLabel(key)} selected={value === key} onPress={() => toggle(key)} />
        ))}
      </View>
    </View>
  );
}
