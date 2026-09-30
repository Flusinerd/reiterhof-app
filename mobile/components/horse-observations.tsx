import { Card, Text } from "@/components/ui";

/**
 * EXTENSION POINT: list of observations ("Auffälligkeiten") of a horse.
 *
 * The observations feature replaces the body of this component (keep the props): load
 * `GET /api/v1/horses/{horseId}/observations`, render the newest entries as cards and a
 * button "Auffälligkeit melden" for owners and riders with the rule `report_observations`.
 * Until then it shows an empty state so the horse record has its final structure.
 */
export function HorseObservations({ horseId }: { horseId: string }) {
  void horseId;
  return (
    <Card>
      <Text variant="secondary">Keine Auffälligkeiten gemeldet.</Text>
    </Card>
  );
}
