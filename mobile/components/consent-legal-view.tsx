import { View } from "react-native";

import { Card, Hero, Screen, Text } from "@/components/ui";
import { parseMarkdown, type Block, type Span } from "@/lib/consent-markdown";

function Spans({ spans, variant = "body" }: { spans: Span[]; variant?: "body" | "bodySm" }) {
  return (
    <>
      {spans.map((s, i) => (
        <Text key={i} variant={s.bold ? "bodyStrong" : variant}>
          {s.text}
        </Text>
      ))}
    </>
  );
}

function BlockView({ block }: { block: Block }) {
  switch (block.type) {
    case "heading":
      return (
        <Text variant={block.level === 3 ? "bodyStrong" : "title"} accessibilityRole="header" className={block.level === 2 ? "mt-4" : undefined}>
          {block.text}
        </Text>
      );
    case "paragraph":
      return (
        <Text variant="body">
          <Spans spans={block.spans} />
        </Text>
      );
    case "quote":
      return (
        <View className="rounded-tile border border-accent-soft bg-accent-soft p-4">
          <Text variant="bodySm">
            <Spans spans={block.spans} variant="bodySm" />
          </Text>
        </View>
      );
    case "list":
      return (
        <View className="gap-2">
          {block.items.map((item, i) => (
            <View key={i} className="flex-row gap-2">
              <Text variant="body">•</Text>
              <Text variant="body" className="flex-1">
                <Spans spans={item} />
              </Text>
            </View>
          ))}
        </View>
      );
  }
}

/**
 * A legal text (docs/legal/*.md, bundled in consent-legal-texts.ts) as a sub page: the first
 * heading becomes the hero title, the rest is rendered in one card.
 */
export function LegalScreen({ markdown, eyebrow, version }: { markdown: string; eyebrow: string; version: string }) {
  const blocks = parseMarkdown(markdown);
  const first = blocks[0];
  const title = first?.type === "heading" ? first.text : eyebrow;
  const rest = first?.type === "heading" ? blocks.slice(1) : blocks;
  return (
    <Screen back>
      <Hero eyebrow={eyebrow} title={title} description={`Textversion ${version}`} />
      <Card className="gap-3">
        {rest.map((block, i) => (
          <BlockView key={i} block={block} />
        ))}
      </Card>
    </Screen>
  );
}
