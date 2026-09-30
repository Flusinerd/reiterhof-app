import assert from "node:assert/strict";
import { test } from "node:test";

import { parseInline, parseMarkdown } from "./consent-markdown.ts";

test("parseInline splits bold text", () => {
  assert.deepEqual(parseInline("a **b** c"), [
    { text: "a ", bold: false },
    { text: "b", bold: true },
    { text: " c", bold: false },
  ]);
  assert.deepEqual(parseInline("**nur fett**"), [{ text: "nur fett", bold: true }]);
  assert.deepEqual(parseInline(""), []);
});

test("parseInline keeps an unmatched marker literal", () => {
  assert.deepEqual(parseInline("a ** b"), [{ text: "a ** b", bold: false }]);
});

test("parseMarkdown handles headings, paragraphs, lists and quotes", () => {
  const blocks = parseMarkdown(
    [
      "# Titel",
      "",
      "> **Entwurf** hier",
      "> zweite Zeile",
      "",
      "## Abschnitt",
      "Zeile eins",
      "Zeile zwei",
      "",
      "- **Alle:** sehen viel",
      "- Niemand",
      "",
      "### Unter",
      "Text",
    ].join("\n"),
  );
  assert.deepEqual(blocks, [
    { type: "heading", level: 1, text: "Titel" },
    {
      type: "quote",
      spans: [
        { text: "Entwurf", bold: true },
        { text: " hier zweite Zeile", bold: false },
      ],
    },
    { type: "heading", level: 2, text: "Abschnitt" },
    { type: "paragraph", spans: [{ text: "Zeile eins Zeile zwei", bold: false }] },
    {
      type: "list",
      items: [
        [
          { text: "Alle:", bold: true },
          { text: " sehen viel", bold: false },
        ],
        [{ text: "Niemand", bold: false }],
      ],
    },
    { type: "heading", level: 3, text: "Unter" },
    { type: "paragraph", spans: [{ text: "Text", bold: false }] },
  ]);
});

test("a plain line ends a list and CRLF is accepted", () => {
  const blocks = parseMarkdown("- a\r\n- b\r\nAbsatz\r\n");
  assert.equal(blocks.length, 2);
  assert.equal(blocks[0].type, "list");
  assert.equal(blocks[1].type, "paragraph");
});
