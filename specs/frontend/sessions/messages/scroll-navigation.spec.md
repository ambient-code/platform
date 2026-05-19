# Session Message Scroll Navigation

## Purpose

The session message view SHALL provide contextual scroll navigation buttons that allow users to quickly navigate to the top or bottom of the message list without manual scrolling.

## Requirements

### Requirement: Scroll-to-Top Button Visibility

The message view SHALL display a scroll-to-top button when the user has scrolled past a threshold distance from the top of the message container.

#### Scenario: Button appears after scrolling down

- GIVEN a message list with content taller than the viewport
- WHEN the user scrolls past a threshold distance from the top
- THEN a scroll-to-top button becomes visible

#### Scenario: Button hidden near the top

- GIVEN the scroll-to-top button is visible
- WHEN the user scrolls back within the threshold distance of the top
- THEN the button becomes hidden

### Requirement: Scroll-to-Bottom Button Visibility

The message view SHALL display a scroll-to-bottom button when the user is not at the bottom of the message list.

#### Scenario: Button appears when scrolled up

- GIVEN a message list with content taller than the viewport
- WHEN the user scrolls up so the bottom of the content is not visible
- THEN a scroll-to-bottom button becomes visible

#### Scenario: Button hidden at bottom

- GIVEN the scroll-to-bottom button is visible
- WHEN the user scrolls to near the bottom of the content
- THEN the button becomes hidden

### Requirement: No Buttons on Short Content

Neither scroll button SHALL appear when the message content does not overflow the container.

#### Scenario: Few messages

- GIVEN a message list shorter than the viewport height
- WHEN the view renders
- THEN neither scroll button is visible

### Requirement: Smooth Animated Transitions

Both buttons SHALL animate in and out smoothly rather than appearing or disappearing abruptly. Hidden buttons SHALL NOT trigger hover effects or tooltips.

#### Scenario: Button fades in

- GIVEN a scroll button is hidden
- WHEN its visibility condition becomes true
- THEN the button transitions to visible smoothly

#### Scenario: Button fades out

- GIVEN a scroll button is visible
- WHEN its visibility condition becomes false
- THEN the button transitions to hidden smoothly
- AND the button does not receive pointer events while hidden
- AND the button's tooltip does not appear while hidden

### Requirement: Smooth User-Initiated Scrolling

Both buttons SHALL scroll smoothly when activated by user click.

#### Scenario: Scroll to top

- GIVEN the scroll-to-top button is visible
- WHEN the user clicks it
- THEN the message container scrolls smoothly to the top

#### Scenario: Scroll to bottom

- GIVEN the scroll-to-bottom button is visible
- WHEN the user clicks it
- THEN the message container scrolls smoothly to the bottom

### Requirement: Instant Auto-Scroll During Streaming

Auto-scroll that keeps the viewport pinned to new messages during streaming SHALL remain instant and not use smooth scrolling.

#### Scenario: New message during streaming

- GIVEN the user is at the bottom of the message list
- WHEN a new streaming message or token arrives
- THEN the container scrolls instantly to show the new content
- AND the scroll is not animated

### Requirement: Button Tooltips

Both buttons SHALL display a tooltip describing their action.

#### Scenario: Tooltip content

- GIVEN either scroll button is visible
- WHEN the user hovers over the button
- THEN a tooltip appears with the text "Scroll to top" or "Scroll to bottom" respectively

### Requirement: Button Layout

Both buttons SHALL be positioned in the bottom-right corner of the message container, stacked vertically with the scroll-to-top button above the scroll-to-bottom button.

#### Scenario: Both buttons visible

- GIVEN the user has scrolled past the top threshold and is not at the bottom
- WHEN both buttons are visible
- THEN they appear stacked vertically in the bottom-right corner of the message area
- AND the scroll-to-top button is above the scroll-to-bottom button
