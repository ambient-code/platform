# Markdown Rendering in Session Messages

## Purpose

Bot messages and tool results display markdown content rendered with clear typography, readable structure, and theme-aware styling. All block-level elements stack vertically in normal document flow. The rendering is consistent across bot messages and tool result views.

## Requirements

### Requirement: Prose Typography

Bot message prose SHALL render in the application's proportional sans-serif font. Inline code SHALL be the only monospace element within a message.

#### Scenario: Body text uses sans-serif

- GIVEN a bot message containing paragraph text
- WHEN rendered
- THEN the message text uses the application's sans-serif font family
- AND no monospace font is applied to prose content

#### Scenario: Inline code is monospace

- GIVEN a bot message containing inline code (backtick-delimited)
- WHEN rendered
- THEN the inline code element uses a monospace font
- AND surrounding text remains in sans-serif

### Requirement: Block Layout

All block-level markdown elements SHALL stack vertically and occupy the full width of the message container.

#### Scenario: Multiple block elements

- GIVEN a bot message containing paragraphs, a table, and a list
- WHEN rendered
- THEN each element stacks vertically in document order
- AND each element fills the container width

#### Scenario: Streaming cursor position

- GIVEN a bot message is actively streaming
- WHEN the cursor is displayed
- THEN it appears at the text baseline of the last rendered character on the same line

### Requirement: Paragraph Spacing

Paragraphs in bot messages SHALL have visually distinct vertical separation.

#### Scenario: Multi-paragraph message

- GIVEN a bot message with three or more paragraphs
- WHEN rendered
- THEN each paragraph has clear bottom margin separating it from the next
- AND the spacing is at least 8px

#### Scenario: No scroll jump on new messages

- GIVEN a scrolled message list with existing messages
- WHEN a new bot message with multiple paragraphs arrives
- THEN the viewport position of previously rendered messages is unaffected

### Requirement: Inline Formatting

Bold, italic, and strikethrough text SHALL render with correct visual treatment and theme-aware color.

#### Scenario: Bold text

- GIVEN `**bold text**` in a bot message
- WHEN rendered
- THEN the text displays with semibold or heavier weight
- AND uses the foreground theme color

#### Scenario: Italic text

- GIVEN `*italic text*` in a bot message
- WHEN rendered
- THEN the text displays in italic style
- AND uses the foreground theme color

#### Scenario: Strikethrough text

- GIVEN `~~strikethrough~~` in a bot message
- WHEN rendered
- THEN the text displays with a line-through decoration
- AND appears at reduced opacity

#### Scenario: Nested formatting

- GIVEN `**bold with _italic_ inside**` in a bot message
- WHEN rendered
- THEN both bold and italic styles compose correctly

### Requirement: List Spacing

Ordered and unordered list items SHALL have vertical spacing harmonized with paragraph spacing for consistent vertical rhythm.

#### Scenario: List item gap

- GIVEN a bot message with a bulleted list
- WHEN rendered
- THEN list items have consistent vertical spacing comparable to paragraph spacing

### Requirement: GFM Table Rendering

GFM tables in bot messages and tool results SHALL render with visible borders, padded cells, a distinct header row, and horizontal overflow protection.

#### Scenario: Basic table

- GIVEN a bot message containing a GFM table with columns and rows
- WHEN rendered
- THEN every cell has a visible border
- AND cells have horizontal and vertical padding
- AND header cells are visually distinct from data cells (different background)

#### Scenario: Theme adaptation

- GIVEN a rendered table
- WHEN the user switches between light and dark mode
- THEN borders, header backgrounds, and text colors adapt to the active theme
- AND no hardcoded colors are visible

#### Scenario: Wide table overflow

- GIVEN a table wider than the message container
- WHEN rendered
- THEN the table scrolls horizontally within its own container
- AND the page body does not scroll horizontally

#### Scenario: Mobile viewport

- GIVEN a wide table on a 375px viewport
- WHEN rendered
- THEN the table is independently scrollable
- AND no content is clipped

#### Scenario: Table cell with inline elements

- GIVEN a table cell containing inline code, bold text, or a link
- WHEN rendered
- THEN the inline elements render correctly within the cell

#### Scenario: Single-column table

- GIVEN a table with only one column
- WHEN rendered
- THEN the column has borders on both sides

### Requirement: Blockquote Styling

Blockquotes SHALL render with a visible left border and indented text, visually distinct from surrounding prose.

#### Scenario: Basic blockquote

- GIVEN a `> quoted text` in a bot message
- WHEN rendered
- THEN the blockquote has a left border, left padding, and italic muted text
- AND it is visually distinct from regular prose

#### Scenario: Multi-line blockquote

- GIVEN a multi-line blockquote
- WHEN rendered
- THEN all lines are contained within a single styled blockquote block

#### Scenario: Theme adaptation

- GIVEN a blockquote
- WHEN the user switches themes
- THEN the border color adapts to the active theme

#### Scenario: Nested blockquote

- GIVEN a nested blockquote (`>> nested`)
- WHEN rendered
- THEN the inner blockquote renders with its own border and indentation inside the outer one

### Requirement: Horizontal Rule

Horizontal rules SHALL render as a full-width line in the theme's border color with vertical margin.

#### Scenario: Section separator

- GIVEN a `---` in a bot message
- WHEN rendered
- THEN a horizontal line appears spanning the message width
- AND the line uses a theme-aware border color

### Requirement: Image Containment

Images in bot messages SHALL be constrained to the message container width.

#### Scenario: Large image

- GIVEN a bot message with an image wider than the container
- WHEN rendered
- THEN the image scales to fit within the container width
- AND does not cause horizontal page overflow

#### Scenario: Broken image

- GIVEN a bot message with a broken image URL
- WHEN rendered
- THEN the alt text is visible
- AND the element does not stretch beyond the container

### Requirement: Extended Heading Levels

H4, H5, and H6 headings SHOULD follow a progressively smaller scale below H3.

#### Scenario: Heading hierarchy

- GIVEN a bot message with H3, H4, H5, and H6 headings
- WHEN rendered
- THEN each lower level appears visually smaller than the one above
- AND all are larger than or equal to body text size

### Requirement: Consistent Rendering Across Views

Markdown rendering SHALL be visually consistent between bot messages and tool result views.

#### Scenario: Table in tool result

- GIVEN a GFM table appearing in a tool result
- WHEN rendered
- THEN borders, padding, and header styling match the same table in a bot message

#### Scenario: Blockquote in tool result

- GIVEN a blockquote appearing in a tool result
- WHEN rendered
- THEN the border, indentation, and text styling match the same blockquote in a bot message

### Requirement: Theme-Aware Colors

All markdown element colors SHALL use theme-aware values that adapt to light and dark mode. No hardcoded color values SHALL be used.

#### Scenario: Dark mode

- GIVEN any styled markdown element (table, blockquote, heading, rule)
- WHEN the user is in dark mode
- THEN all borders, backgrounds, and text colors resolve to their dark-mode theme values
