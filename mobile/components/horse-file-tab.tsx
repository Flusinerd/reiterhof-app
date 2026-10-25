import type { Href } from "expo-router";
import { Activity, FileText, HeartPulse, Pencil, Shirt } from "lucide-react-native";
import { View } from "react-native";

import { HorseEmergencyPreview } from "@/components/horse-emergency-preview";
import { HorseError, HorseLoading } from "@/components/horse-query-state";
import { HorseRiders } from "@/components/horse-riders";
import { Button, Icon, Section, Text } from "@/components/ui";
import { ListRow, RowList } from "@/components/ui/list-row";
import { useDocuments, useEmergency, type Horse } from "@/lib/api/horses";
import { horseRoutes } from "@/lib/horse-format";
import { pushOnce } from "@/lib/navigation";

/**
 * "Akte" of the horse page: emergency card preview, riders with their rules, documents, the links
 * to training profile, blanket plan and reha, helper note and "Bearbeiten".
 */
export function HorseFileTab({ horse: h }: { horse: Horse }) {
  const emergency = useEmergency(h.id);
  const canSeeDocuments = h.can_manage || h.i_ride;
  const documents = useDocuments(h.id, canSeeDocuments);
  const go = (path: string) => pushOnce(path as Href);

  return (
    <View className="gap-6">
      <Section title="Notfallkarte">
        {emergency.data ? (
          <HorseEmergencyPreview card={emergency.data} onPress={() => go(horseRoutes.emergency(h.id))} />
        ) : emergency.isError ? (
          <HorseError error={emergency.error} onRetry={() => emergency.refetch()} />
        ) : (
          <HorseLoading />
        )}
      </Section>

      <Section title="Reitbeteiligungen">
        <HorseRiders horse={h} />
      </Section>

      <Section title="Mehr zum Pferd">
        <RowList>
          {canSeeDocuments ? (
            <ListRow
              leading={<Icon as={FileText} size={20} className="text-muted" />}
              title="Dokumente"
              detail={
                documents.data
                  ? documents.data.length === 0
                    ? "Noch keine Dokumente"
                    : `${documents.data.length} ${documents.data.length === 1 ? "Dokument" : "Dokumente"}`
                  : "Equidenpass, Impfpass, Versicherung"
              }
              accessibilityLabel="Dokumente"
              onPress={() => go(horseRoutes.documents(h.id))}
            />
          ) : null}
          <ListRow
            leading={<Icon as={Activity} size={20} className="text-muted" />}
            title="Trainingsprofil"
            accessibilityLabel="Trainingsprofil"
            onPress={() => go(horseRoutes.trainingProfile(h.id))}
          />
          <ListRow
            leading={<Icon as={Shirt} size={20} className="text-muted" />}
            title="Deckenplan"
            accessibilityLabel="Deckenplan"
            onPress={() => go(horseRoutes.blanketPlan(h.id))}
          />
          <ListRow
            leading={<Icon as={HeartPulse} size={20} className="text-muted" />}
            title="Reha"
            accessibilityLabel="Reha"
            onPress={() => go(horseRoutes.reha(h.id))}
          />
        </RowList>
      </Section>

      {h.helper_note ? (
        <Section title="Hinweise für Helfer">
          <Text variant="body">{h.helper_note}</Text>
        </Section>
      ) : null}

      {h.can_manage ? (
        <Button
          label="Pferd bearbeiten"
          icon={Pencil}
          variant="outline"
          fullWidth
          onPress={() => go(horseRoutes.edit(h.id))}
        />
      ) : null}
    </View>
  );
}
