export const INACTIVITY_TIMEOUT_TOOLTIP =
  "The session is stopped when no activity (user messages) is detected for this duration. The countdown starts from the last activity time, or from session start if there is no interaction. When set to 0, auto-stop is disabled entirely.";

// Kubernetes TokenRequest does not support non-expiring tokens — the API server
// silently caps ExpirationSeconds. Max is 1 year; "No expiration" is not offered
// because K8s will expire the token regardless.
export const EXPIRATION_OPTIONS = [
  { value: '86400', label: '1 day' },
  { value: '604800', label: '7 days' },
  { value: '2592000', label: '30 days' },
  { value: '7776000', label: '90 days' },
  { value: '31536000', label: '1 year' },
] as const;

export const DEFAULT_EXPIRATION = '7776000'; // 90 days
