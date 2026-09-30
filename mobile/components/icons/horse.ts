import { createLucideIcon } from "lucide-react-native";

/**
 * Horse, standing, facing right. Lucide has no horse, so this one is drawn in the Lucide style
 * (24 x 24 grid, 2 px padding, outline only, stroke 2, round caps and joins) and built with the
 * same factory: it takes `size`, `color`, `strokeWidth` and, through `Icon`, `className`
 * like every other icon.
 *
 * @example <Icon as={Horse} size={20} className="text-muted" />
 */
export const Horse = createLucideIcon("horse", [
  [
    "path",
    {
      d: "M4.5 10.2Q4.5 8 6.8 8H12.2L14.4 3.8L15.4 2.3L16.6 4.1L21.2 7.6L21.5 9.4H18.6L17.4 8.2L16.9 12Q16.6 15 13.6 15H8Q4.5 15 4.5 10.2Z",
      key: "body",
    },
  ],
  ["path", { d: "M14.2 15V21", key: "front-leg" }],
  ["path", { d: "M7.8 15C7.8 17.2 6.6 18.2 6.6 21", key: "hind-leg" }],
  ["path", { d: "M4.7 9.6C2.7 10.2 2.2 12.8 3.6 15.4", key: "tail" }],
]);
