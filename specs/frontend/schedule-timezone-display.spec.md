# Schedule Timezone Display Specification

## Purpose

Defines how the frontend displays cron-based schedule times to users. All schedules are stored and configured in UTC. The frontend annotates displayed times with the UTC label and, when the user's browser is not in UTC, appends the local timezone equivalent so users can understand when a schedule will fire in their own timezone without performing mental offset calculations.

## Requirements

### Requirement: UTC Label on Daily-or-Longer Schedules

When displaying a human-readable cron description for a schedule that fires once per day or less frequently (daily, weekly, monthly), the system SHALL insert "UTC" immediately after the time portion in the description.

#### Scenario: Daily schedule at 09:00

- GIVEN a cron expression `0 9 * * *`
- WHEN the frontend renders the schedule description
- THEN the description SHALL contain "09:00 UTC" or "9:00 AM UTC"

#### Scenario: Weekday schedule at 14:30

- GIVEN a cron expression `30 14 * * 1-5`
- WHEN the frontend renders the schedule description
- THEN the description SHALL contain "14:30 UTC" or "02:30 PM UTC"

#### Scenario: Monthly schedule

- GIVEN a cron expression `0 12 1 * *`
- WHEN the frontend renders the schedule description
- THEN the description SHALL contain "UTC" after the time portion

### Requirement: Local Timezone Parenthetical

When the user's browser timezone differs from UTC, the system SHALL append the equivalent local time in parentheses after the UTC-labeled description.

#### Scenario: User in UTC+1 viewing a daily schedule

- GIVEN a cron expression `0 9 * * *`
- AND the user's browser is in a UTC+1 timezone
- WHEN the frontend renders the schedule description
- THEN the output SHALL match the pattern `<description> UTC (<local time> <timezone>)`
- AND the local time SHALL reflect the browser's offset (e.g., "10:00 AM GMT+1")

#### Scenario: User in UTC viewing a daily schedule

- GIVEN a cron expression `0 9 * * *`
- AND the user's browser is in UTC
- WHEN the frontend renders the schedule description
- THEN the output SHALL contain "UTC"
- AND the output SHALL NOT contain a parenthesized local time (since local equals UTC)

### Requirement: Sub-Daily Schedules Omit Timezone Annotation

Schedules that fire more frequently than once per day (every minute, every N minutes, hourly, every N hours) SHALL display the plain cron description without any UTC label or local timezone parenthetical.

#### Scenario: Every 5 minutes

- GIVEN a cron expression `*/5 * * * *`
- WHEN the frontend renders the schedule description
- THEN the output SHALL be the plain human-readable description (e.g., "Every 5 minutes")
- AND the output SHALL NOT contain "UTC"

#### Scenario: Every 2 hours

- GIVEN a cron expression `0 */2 * * *`
- WHEN the frontend renders the schedule description
- THEN the output SHALL be the plain human-readable description
- AND the output SHALL NOT contain "UTC"

#### Scenario: Hourly

- GIVEN a cron expression `0 * * * *`
- WHEN the frontend renders the schedule description
- THEN the output SHALL be the plain human-readable description
- AND the output SHALL NOT contain "UTC"

### Requirement: Sub-Daily Detection

The system SHALL determine whether a schedule is sub-daily by comparing the interval between consecutive future run times. If the interval between the first two next runs is less than 24 hours, the schedule is sub-daily.

#### Scenario: Every 12 hours classified as sub-daily

- GIVEN a cron expression `0 */12 * * *`
- WHEN the system evaluates whether the schedule is sub-daily
- THEN it SHALL classify the schedule as sub-daily regardless of the current time of day

### Requirement: Graceful Fallback

The system SHALL gracefully handle invalid or unparseable cron expressions by returning the raw expression string without modification.

#### Scenario: Invalid cron expression

- GIVEN an invalid cron expression `not-a-cron`
- WHEN the frontend renders the schedule description
- THEN the output SHALL be the raw string `not-a-cron`

#### Scenario: Cron expression with no recognizable time portion

- GIVEN a valid cron expression whose human-readable description does not contain a recognizable time pattern
- WHEN the frontend renders the schedule description
- THEN the output SHALL be the plain description without UTC annotation

### Requirement: Both 12-Hour and 24-Hour Format Support

The system SHALL correctly identify and annotate time portions in both 12-hour format (e.g., "02:30 PM") and 24-hour format (e.g., "14:30"), depending on the user's locale settings.

#### Scenario: 24-hour locale

- GIVEN a cron expression `30 14 * * *`
- AND the user's locale produces 24-hour time format "14:30"
- WHEN the frontend renders the schedule description
- THEN the output SHALL contain "14:30 UTC"

#### Scenario: 12-hour locale

- GIVEN a cron expression `30 14 * * *`
- AND the user's locale produces 12-hour time format "02:30 PM"
- WHEN the frontend renders the schedule description
- THEN the output SHALL contain "02:30 PM UTC"
