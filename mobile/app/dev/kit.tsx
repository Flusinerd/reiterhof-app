import { Check, Plus } from "lucide-react-native";
import { useState } from "react";
import { View } from "react-native";

import {
  Avatar,
  Badge,
  Button,
  Card,
  Divider,
  Hero,
  Icon,
  Pill,
  Screen,
  SectionLabel,
  Sheet,
  Switch,
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
  Text,
  ToggleGroup,
  ToggleGroupItem,
} from "@/components/ui";
import { COLOR_KEYS } from "@/lib/color-keys";
import { colors } from "@/lib/theme";

// Hidden dev route (/dev/kit): shows every kit component for review.
export default function Kit() {
  const [tab, setTab] = useState("today");
  const [gait, setGait] = useState("trot");
  const [days, setDays] = useState<string[]>(["mon"]);
  const [filter, setFilter] = useState("all");
  const [on, setOn] = useState(true);
  const [sheetOpen, setSheetOpen] = useState(false);

  return (
    <Screen back>
      <Hero
        eyebrow="Entwicklung"
        title="Komponenten"
        description="Alle Bausteine des Design-Systems auf einen Blick."
      >
        <View className="flex-row gap-4">
          {(
            [
              ["Schritt", colors.gait.walk],
              ["Trab", colors.gait.trot],
              ["Galopp", colors.gait.canter],
            ] as const
          ).map(([label, color]) => (
            <View key={label} className="flex-row items-center gap-2">
              <View className="h-3 w-3 rounded-pill" style={{ backgroundColor: color }} />
              <Text variant="secondary">{label}</Text>
            </View>
          ))}
        </View>
      </Hero>

      <SectionLabel>Text</SectionLabel>
      <Card className="gap-2">
        <Text variant="titleLg">Titel 28</Text>
        <Text variant="title">Titel 26</Text>
        <Text variant="heroNumberSm">44</Text>
        <Text variant="body">Fließtext 15 px</Text>
        <Text variant="bodySm">Fließtext 14 px</Text>
        <Text variant="secondary">Sekundärtext 13 px</Text>
        <Text variant="caption">Beschriftung 12 px</Text>
      </Card>

      <SectionLabel>Buttons</SectionLabel>
      <Card className="gap-3">
        <Button label="Primär" icon={Check} />
        <Button label="Sekundär" variant="secondary" />
        <Button label="Umrandet" variant="outline" />
        <Button label="Ghost" variant="ghost" />
        <Button label="Löschen" variant="danger" />
        <Button label="Lädt" loading />
        <Button label="Deaktiviert" disabled />
        <View className="flex-row items-center gap-3">
          <Button label="Klein" size="sm" />
          <Button label="Groß" size="lg" />
          <Button size="icon" icon={Plus} variant="outline" accessibilityLabel="Hinzufügen" />
        </View>
      </Card>

      <SectionLabel>Badges und Pills</SectionLabel>
      <Card className="gap-3">
        <View className="flex-row flex-wrap gap-2">
          <Badge label="Neutral" />
          <Badge variant="primary" label="Erledigt" />
          <Badge variant="accent" label="Dringend" />
          <Badge variant="info" label="Info" />
          <Badge variant="danger" label="Abgelehnt" />
        </View>
        <View className="flex-row flex-wrap gap-2">
          {[
            ["all", "Alle"],
            ["open", "Offen"],
            ["done", "Erledigt"],
          ].map(([value, label]) => (
            <Pill key={value} label={label} selected={filter === value} onPress={() => setFilter(value)} />
          ))}
        </View>
      </Card>

      <SectionLabel>Avatare</SectionLabel>
      <Card className="gap-4">
        <View className="flex-row flex-wrap gap-3">
          {COLOR_KEYS.map((key) => (
            <Avatar key={key} name={key} colorKey={key} />
          ))}
        </View>
        <View className="flex-row items-end gap-3">
          <Avatar name="Luna" colorKey="green" size="sm" />
          <Avatar name="Fanta" colorKey="amber" size="md" />
          <Avatar name="Balu" colorKey="blue" size="lg" />
          <Avatar name="Cookie" colorKey="rose" size="xl" />
        </View>
      </Card>

      <SectionLabel>Tabs, Schalter, Gruppen</SectionLabel>
      <Card className="gap-4">
        <Tabs value={tab} onValueChange={setTab}>
          <TabsList>
            <TabsTrigger value="today" label="Heute" />
            <TabsTrigger value="week" label="Woche" />
          </TabsList>
          <TabsContent value="today">
            <Text variant="secondary">Inhalt für „Heute“.</Text>
          </TabsContent>
          <TabsContent value="week">
            <Text variant="secondary">Inhalt für „Woche“.</Text>
          </TabsContent>
        </Tabs>
        <Divider />
        <Switch label="Erinnerungen" description="Push am Morgen" value={on} onValueChange={setOn} />
        <Divider />
        <ToggleGroup type="single" value={gait} onValueChange={setGait}>
          <ToggleGroupItem value="walk" label="Schritt" />
          <ToggleGroupItem value="trot" label="Trab" />
          <ToggleGroupItem value="canter" label="Galopp" />
        </ToggleGroup>
        <ToggleGroup type="multiple" value={days} onValueChange={setDays}>
          <ToggleGroupItem value="mon" label="Mo" />
          <ToggleGroupItem value="tue" label="Di" />
          <ToggleGroupItem value="wed" label="Mi" />
        </ToggleGroup>
      </Card>

      <SectionLabel>Karten und Sheet</SectionLabel>
      <Card shape="tile" className="flex-row items-center gap-3">
        <Icon as={Check} className="text-primary" />
        <Text variant="body">Kachel mit Radius 16</Text>
      </Card>
      <Button label="Sheet öffnen" variant="outline" onPress={() => setSheetOpen(true)} />
      <Sheet open={sheetOpen} onOpenChange={setSheetOpen} title="Neue Anfrage" description="Beispiel für ein Bottom-Sheet.">
        <Button label="Senden" fullWidth onPress={() => setSheetOpen(false)} />
      </Sheet>
    </Screen>
  );
}
