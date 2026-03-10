package types

// CreateScheduledSessionRequest is the request body for creating a scheduled session.
type CreateScheduledSessionRequest struct {
	Schedule        string                      `json:"schedule" binding:"required"`
	DisplayName     string                      `json:"displayName" binding:"required"`
	SessionTemplate CreateAgenticSessionRequest `json:"sessionTemplate" binding:"required"`
	Suspend         bool                        `json:"suspend,omitempty"`
}

// UpdateScheduledSessionRequest is the request body for updating a scheduled session (partial updates).
type UpdateScheduledSessionRequest struct {
	Schedule        *string                      `json:"schedule,omitempty"`
	DisplayName     *string                      `json:"displayName,omitempty"`
	SessionTemplate *CreateAgenticSessionRequest `json:"sessionTemplate,omitempty"`
	Suspend         *bool                        `json:"suspend,omitempty"`
}

// ScheduledSession is the response type for a scheduled session (backed by a CronJob).
type ScheduledSession struct {
	Name              string                      `json:"name"`
	Namespace         string                      `json:"namespace"`
	CreationTimestamp string                      `json:"creationTimestamp"`
	Schedule          string                      `json:"schedule"`
	Suspend           bool                        `json:"suspend"`
	DisplayName       string                      `json:"displayName"`
	SessionTemplate   CreateAgenticSessionRequest `json:"sessionTemplate"`
	LastScheduleTime  *string                     `json:"lastScheduleTime,omitempty"`
	ActiveCount       int                         `json:"activeCount"`
	Labels            map[string]string           `json:"labels,omitempty"`
	Annotations       map[string]string           `json:"annotations,omitempty"`
}
