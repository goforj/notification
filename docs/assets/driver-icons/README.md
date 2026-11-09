# Driver artwork

`presentation.json` owns driver labels, icon paths and monochrome brand colors. `sources.json` records the original source, license and SHA-256 of each vendored mark. Brand marks identify their corresponding services; trademarks remain with their owners. Feishu uses the shared Feishu/Lark mark. Mail, Log and Webhook use original domain symbols.

The generator embeds vector artwork and the three upstream PNG marks directly in compact SVG badges. Generation needs no external image service. Provider marks retain their original colors. Each badge uses a subtle brand tint, a fine matching border and dark text, with no separate icon tile. Empty Illustrator foreignObject metadata is omitted when embedding Gotify's SVG; visible artwork is preserved.

To update a mark, replace its source asset, refresh its provenance hash, and run `make generate` and `make check-generated`. Do not edit generated `docs/assets/badges/driver-*.svg` files directly.
