import { View } from "react-native";

import { ActivityIcon } from "@/components/training-activity-icon";
import { Card, Divider, Input, Pill, Switch, Text } from "@/components/ui";
import { ACTIVITIES, activityLabel, type Activity, type ActivityMode } from "@/lib/training";
import type { ProfileDraft } from "@/lib/training-profile";

export type ActivityModeListProps = {
  modes: ProfileDraft["modes"];
  /** False shows the modes as text, without controls (riders). */
  editable: boolean;
  onChange: (activity: Activity, value: { mode: ActivityMode; note: string }) => void;
};

/**
 * One row per activity: a switch for on/off and, while on, the pill "Nur unter Bedingung" which makes
 * the activity conditional and asks for the condition. Used by the setup wizard and the activities editor.
 */
export function ActivityModeList({ modes, editable, onChange }: ActivityModeListProps) {
  return (
    <Card padded={false}>
      {ACTIVITIES.map((a, i) => {
        const { mode, note } = modes[a];
        const label = activityLabel(a);
        return (
          <View key={a}>
            {i > 0 ? <Divider /> : null}
            {editable ? (
              <View className="gap-3 px-4 py-2">
                <View className="flex-row items-center gap-3">
                  <ActivityIcon activity={a} size={20} />
                  <Switch
                    className="flex-1"
                    label={label}
                    value={mode !== "off"}
                    onValueChange={(on) => onChange(a, { mode: on ? "on" : "off", note })}
                  />
                </View>
                {mode !== "off" ? (
                  <View className="gap-2 pb-2">
                    <Pill
                      label="Nur unter Bedingung"
                      selected={mode === "conditional"}
                      onPress={() => onChange(a, { mode: mode === "conditional" ? "on" : "conditional", note })}
                    />
                    {mode === "conditional" ? (
                      <>
                        <Input
                          accessibilityLabel={`Bedingung für ${label}`}
                          placeholder="Bedingung, z. B. nur mit Begleitung"
                          value={note}
                          maxLength={200}
                          onChangeText={(text) => onChange(a, { mode, note: text })}
                        />
                        {note.trim() === "" ? (
                          <Text variant="secondary" tone="danger">
                            Bedingung fehlt.
                          </Text>
                        ) : null}
                      </>
                    ) : null}
                  </View>
                ) : null}
              </View>
            ) : (
              <View className="gap-1 px-4 py-3">
                <View className="min-h-8 flex-row items-center gap-3">
                  <ActivityIcon activity={a} size={20} />
                  <Text variant="bodyStrong" className="flex-1">
                    {label}
                  </Text>
                  {mode === "off" ? <Text variant="secondary">Aus</Text> : null}
                </View>
                {mode === "conditional" ? <Text variant="secondary">Bedingung: {note}</Text> : null}
              </View>
            )}
          </View>
        );
      })}
    </Card>
  );
}
