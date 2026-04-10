package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"ambient-code-backend/server"
	"ambient-code-backend/types"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	authzv1 "k8s.io/api/authorization/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// K8sClientScheduled is the backend service account client used for CronJob operations.
var K8sClientScheduled kubernetes.Interface

const (
	labelScheduledSession      = "ambient-code.io/scheduled-session"
	labelScheduledSessionName  = "ambient-code.io/scheduled-session-name"
	labelCreatedBy             = "ambient-code.io/created-by"
	annotationDisplayName      = "ambient-code.io/display-name"
	annotationReuseLastSession = "ambient-code.io/reuse-last-session"

	// flagReuseLastSession is the Unleash feature flag for the reuse last session option.
	flagReuseLastSession = "scheduled-session.reuse.enabled"
)

// checkScheduledSessionAccess verifies the user token and checks RBAC permission
// for the given verb on agenticsessions in the project namespace.
// CronJobs are platform-managed resources, so we gate access using agenticsessions
// permissions as a proxy — the same gate used by the main session handlers.
func checkScheduledSessionAccess(c *gin.Context, verb string) bool {
	k8s, _ := GetK8sClientsForRequest(c)
	if k8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return false
	}

	project := c.GetString("project")
	ssar := &authzv1.SelfSubjectAccessReview{
		Spec: authzv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authzv1.ResourceAttributes{
				Group:     "vteam.ambient-code",
				Resource:  "agenticsessions",
				Verb:      verb,
				Namespace: project,
			},
		},
	}
	res, err := k8s.AuthorizationV1().SelfSubjectAccessReviews().Create(c.Request.Context(), ssar, metav1.CreateOptions{})
	if err != nil {
		log.Printf("RBAC check failed for scheduled session %s in project %s: %v", verb, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify permissions"})
		return false
	}
	if !res.Status.Allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized to manage scheduled sessions in this project"})
		return false
	}
	return true
}

// ListScheduledSessions lists all CronJobs labeled as scheduled sessions.
func ListScheduledSessions(c *gin.Context) {
	project := c.GetString("project")

	if !checkScheduledSessionAccess(c, "list") {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	list, err := K8sClientScheduled.BatchV1().CronJobs(project).List(ctx, metav1.ListOptions{
		LabelSelector: labelScheduledSession + "=true",
	})
	if err != nil {
		log.Printf("Failed to list scheduled sessions in project %s: %v", project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list scheduled sessions"})
		return
	}

	sessions := make([]types.ScheduledSession, 0, len(list.Items))
	for i := range list.Items {
		sessions = append(sessions, cronJobToScheduledSession(&list.Items[i]))
	}

	c.JSON(http.StatusOK, gin.H{"items": sessions})
}

// CreateScheduledSession creates a new CronJob that triggers agentic sessions on a schedule.
func CreateScheduledSession(c *gin.Context) {
	project := c.GetString("project")

	if !checkScheduledSessionAccess(c, "create") {
		return
	}

	var req types.CreateScheduledSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Invalid request body for scheduled session in project %s: %v", project, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if !isValidCronExpression(req.Schedule) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid cron schedule format"})
		return
	}

	// Gate reuseLastSession behind feature flag
	if req.ReuseLastSession && !isReuseEnabled(c) {
		req.ReuseLastSession = false
	}

	userID := c.GetString("userID")

	// Inject userContext into the session template so triggered sessions can
	// resolve credentials (GitHub, GitLab, etc.) via the backend API.
	// The trigger creates the CR directly, bypassing CreateAgenticSession's
	// server-side userContext injection, so we must embed it here.
	if req.SessionTemplate.UserContext == nil && userID != "" {
		displayName := ""
		if v, ok := c.Get("userName"); ok {
			if s, ok2 := v.(string); ok2 {
				displayName = s
			}
		}
		groups := []string{}
		if v, ok := c.Get("userGroups"); ok {
			if gg, ok2 := v.([]string); ok2 {
				groups = gg
			}
		}
		req.SessionTemplate.UserContext = &types.UserContext{
			UserID:      userID,
			DisplayName: displayName,
			Groups:      groups,
		}
	}

	name := fmt.Sprintf("schedule-%s", uuid.New().String())

	// Inject display name into session template so the trigger can use it for naming
	if req.DisplayName != "" && req.SessionTemplate.DisplayName == "" {
		req.SessionTemplate.DisplayName = req.DisplayName
	}

	templateJSON, err := json.Marshal(req.SessionTemplate)
	if err != nil {
		log.Printf("Failed to marshal session template: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encode session template"})
		return
	}

	successLimit := int32(5)
	failedLimit := int32(3)
	startDeadline := int64(300)

	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: project,
			Labels: map[string]string{
				labelScheduledSession: "true",
				labelCreatedBy:        sanitizeLabelValue(userID),
			},
			Annotations: map[string]string{
				annotationDisplayName:      req.DisplayName,
				annotationReuseLastSession: fmt.Sprintf("%t", req.ReuseLastSession),
			},
		},
		Spec: batchv1.CronJobSpec{
			Schedule:                   req.Schedule,
			Suspend:                    &req.Suspend,
			ConcurrencyPolicy:          batchv1.ForbidConcurrent,
			SuccessfulJobsHistoryLimit: &successLimit,
			FailedJobsHistoryLimit:     &failedLimit,
			StartingDeadlineSeconds:    &startDeadline,
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							ServiceAccountName: "ambient-session-trigger",
							RestartPolicy:      corev1.RestartPolicyNever,
							Containers: []corev1.Container{
								{
									Name:            "trigger",
									Image:           server.OperatorImage,
									ImagePullPolicy: corev1.PullPolicy(server.ImagePullPolicy),
									Command:         []string{"./operator", "session-trigger"},
									Env: []corev1.EnvVar{
										{Name: "SESSION_TEMPLATE", Value: string(templateJSON)},
										{Name: "PROJECT_NAMESPACE", Value: project},
										{Name: "SCHEDULED_SESSION_NAME", Value: name},
										{Name: "REUSE_LAST_SESSION", Value: fmt.Sprintf("%t", req.ReuseLastSession)},
									},
									SecurityContext: &corev1.SecurityContext{
										AllowPrivilegeEscalation: types.BoolPtr(false),
										Capabilities: &corev1.Capabilities{
											Drop: []corev1.Capability{"ALL"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	created, err := K8sClientScheduled.BatchV1().CronJobs(project).Create(c.Request.Context(), cronJob, metav1.CreateOptions{})
	if err != nil {
		log.Printf("Failed to create scheduled session in project %s: %v", project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create scheduled session"})
		return
	}

	c.JSON(http.StatusCreated, cronJobToScheduledSession(created))
}

// GetScheduledSession gets a single scheduled session by name.
func GetScheduledSession(c *gin.Context) {
	project := c.GetString("project")
	name := c.Param("scheduledSessionName")

	if !checkScheduledSessionAccess(c, "get") {
		return
	}

	cj, err := K8sClientScheduled.BatchV1().CronJobs(project).Get(c.Request.Context(), name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Scheduled session not found"})
			return
		}
		log.Printf("Failed to get scheduled session %s in project %s: %v", name, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get scheduled session"})
		return
	}

	c.JSON(http.StatusOK, cronJobToScheduledSession(cj))
}

// UpdateScheduledSession applies partial updates to a scheduled session.
func UpdateScheduledSession(c *gin.Context) {
	project := c.GetString("project")
	name := c.Param("scheduledSessionName")

	if !checkScheduledSessionAccess(c, "update") {
		return
	}

	var req types.UpdateScheduledSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Invalid request body for scheduled session update %s in project %s: %v", name, project, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Gate reuseLastSession behind feature flag
	if req.ReuseLastSession != nil && *req.ReuseLastSession && !isReuseEnabled(c) {
		req.ReuseLastSession = nil
	}

	cj, err := K8sClientScheduled.BatchV1().CronJobs(project).Get(c.Request.Context(), name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Scheduled session not found"})
			return
		}
		log.Printf("Failed to get scheduled session %s for update: %v", name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get scheduled session"})
		return
	}

	if req.Schedule != nil {
		if !isValidCronExpression(*req.Schedule) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid cron schedule format"})
			return
		}
		cj.Spec.Schedule = *req.Schedule
	}
	if req.Suspend != nil {
		cj.Spec.Suspend = req.Suspend
	}
	if req.DisplayName != nil {
		if cj.Annotations == nil {
			cj.Annotations = map[string]string{}
		}
		cj.Annotations[annotationDisplayName] = *req.DisplayName
	}
	if req.ReuseLastSession != nil {
		if cj.Annotations == nil {
			cj.Annotations = map[string]string{}
		}
		cj.Annotations[annotationReuseLastSession] = fmt.Sprintf("%t", *req.ReuseLastSession)
		upsertTriggerEnvVar(cj, "REUSE_LAST_SESSION", fmt.Sprintf("%t", *req.ReuseLastSession))
	}
	if req.SessionTemplate != nil {
		// Inject userContext so updated templates retain credential resolution
		userID := c.GetString("userID")
		if req.SessionTemplate.UserContext == nil && userID != "" {
			displayName := ""
			if v, ok := c.Get("userName"); ok {
				if s, ok2 := v.(string); ok2 {
					displayName = s
				}
			}
			groups := []string{}
			if v, ok := c.Get("userGroups"); ok {
				if gg, ok2 := v.([]string); ok2 {
					groups = gg
				}
			}
			req.SessionTemplate.UserContext = &types.UserContext{
				UserID:      userID,
				DisplayName: displayName,
				Groups:      groups,
			}
		}
		templateJSON, err := json.Marshal(req.SessionTemplate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encode session template"})
			return
		}
		upsertTriggerEnvVar(cj, "SESSION_TEMPLATE", string(templateJSON))
	}

	updated, err := K8sClientScheduled.BatchV1().CronJobs(project).Update(c.Request.Context(), cj, metav1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to update scheduled session %s in project %s: %v", name, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update scheduled session"})
		return
	}

	c.JSON(http.StatusOK, cronJobToScheduledSession(updated))
}

// DeleteScheduledSession deletes a scheduled session and its child jobs.
func DeleteScheduledSession(c *gin.Context) {
	project := c.GetString("project")
	name := c.Param("scheduledSessionName")

	if !checkScheduledSessionAccess(c, "delete") {
		return
	}

	propagation := metav1.DeletePropagationBackground
	err := K8sClientScheduled.BatchV1().CronJobs(project).Delete(c.Request.Context(), name, metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Scheduled session not found"})
			return
		}
		log.Printf("Failed to delete scheduled session %s in project %s: %v", name, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete scheduled session"})
		return
	}

	c.Status(http.StatusNoContent)
}

// SuspendScheduledSession sets suspend=true on a scheduled session.
func SuspendScheduledSession(c *gin.Context) {
	patchSuspend(c, true)
}

// ResumeScheduledSession sets suspend=false on a scheduled session.
func ResumeScheduledSession(c *gin.Context) {
	patchSuspend(c, false)
}

func patchSuspend(c *gin.Context, suspend bool) {
	project := c.GetString("project")
	name := c.Param("scheduledSessionName")

	if !checkScheduledSessionAccess(c, "update") {
		return
	}

	cj, err := K8sClientScheduled.BatchV1().CronJobs(project).Get(c.Request.Context(), name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Scheduled session not found"})
			return
		}
		log.Printf("Failed to get scheduled session %s for suspend/resume: %v", name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get scheduled session"})
		return
	}

	cj.Spec.Suspend = &suspend
	updated, err := K8sClientScheduled.BatchV1().CronJobs(project).Update(c.Request.Context(), cj, metav1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to update suspend for scheduled session %s: %v", name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update scheduled session"})
		return
	}

	c.JSON(http.StatusOK, cronJobToScheduledSession(updated))
}

// TriggerScheduledSession creates a one-off Job from the CronJob's jobTemplate.
func TriggerScheduledSession(c *gin.Context) {
	project := c.GetString("project")
	name := c.Param("scheduledSessionName")

	if !checkScheduledSessionAccess(c, "create") {
		return
	}

	cj, err := K8sClientScheduled.BatchV1().CronJobs(project).Get(c.Request.Context(), name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Scheduled session not found"})
			return
		}
		log.Printf("Failed to get scheduled session %s for trigger: %v", name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get scheduled session"})
		return
	}

	jobName := fmt.Sprintf("%s-manual-%s", name, uuid.New().String()[:8])
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: project,
			Labels: map[string]string{
				labelScheduledSession:     "true",
				labelScheduledSessionName: name,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "batch/v1",
					Kind:       "CronJob",
					Name:       cj.Name,
					UID:        cj.UID,
					Controller: types.BoolPtr(true),
				},
			},
		},
		Spec: *cj.Spec.JobTemplate.Spec.DeepCopy(),
	}

	created, err := K8sClientScheduled.BatchV1().Jobs(project).Create(c.Request.Context(), job, metav1.CreateOptions{})
	if err != nil {
		log.Printf("Failed to trigger scheduled session %s: %v", name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to trigger scheduled session"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"name":      created.Name,
		"namespace": created.Namespace,
	})
}

// ListScheduledSessionRuns lists AgenticSessions created by a scheduled session.
func ListScheduledSessionRuns(c *gin.Context) {
	project := c.GetString("project")
	name := c.Param("scheduledSessionName")

	_, k8sDyn := GetK8sClientsForRequest(c)
	if k8sDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	gvr := GetAgenticSessionV1Alpha1Resource()
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	list, err := k8sDyn.Resource(gvr).Namespace(project).List(ctx, metav1.ListOptions{
		LabelSelector: labelScheduledSessionName + "=" + name,
	})
	if err != nil {
		log.Printf("Failed to list runs for scheduled session %s: %v", name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list scheduled session runs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": list.Items})
}

// cronJobToScheduledSession converts a CronJob to a ScheduledSession response.
func cronJobToScheduledSession(cj *batchv1.CronJob) types.ScheduledSession {
	ss := types.ScheduledSession{
		Name:              cj.Name,
		Namespace:         cj.Namespace,
		CreationTimestamp: cj.CreationTimestamp.Format(time.RFC3339),
		Schedule:          cj.Spec.Schedule,
		ActiveCount:       len(cj.Status.Active),
	}

	if cj.Spec.Suspend != nil {
		ss.Suspend = *cj.Spec.Suspend
	}

	if cj.Status.LastScheduleTime != nil {
		t := cj.Status.LastScheduleTime.Format(time.RFC3339)
		ss.LastScheduleTime = &t
	}

	if cj.Labels != nil {
		ss.Labels = cj.Labels
	}
	if cj.Annotations != nil {
		ss.Annotations = cj.Annotations
		if dn, ok := cj.Annotations[annotationDisplayName]; ok {
			ss.DisplayName = dn
		}
		if reuse, ok := cj.Annotations[annotationReuseLastSession]; ok {
			ss.ReuseLastSession = reuse == "true"
		}
	}

	// Extract session template from the trigger container's SESSION_TEMPLATE env var
	for _, container := range cj.Spec.JobTemplate.Spec.Template.Spec.Containers {
		if container.Name == "trigger" {
			for _, env := range container.Env {
				if env.Name == "SESSION_TEMPLATE" {
					var tmpl types.CreateAgenticSessionRequest
					if err := json.Unmarshal([]byte(env.Value), &tmpl); err == nil {
						ss.SessionTemplate = tmpl
					}
				}
			}
		}
	}

	return ss
}

// isReuseEnabled checks if the reuse last session feature flag is enabled,
// respecting workspace-scoped overrides.
func isReuseEnabled(c *gin.Context) bool {
	project := c.GetString("project")
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s != nil {
		overrides, err := getWorkspaceOverrides(c.Request.Context(), reqK8s, project)
		if err == nil && overrides != nil {
			if val, exists := overrides[flagReuseLastSession]; exists {
				return val == "true"
			}
		}
	}
	return FeatureEnabled(flagReuseLastSession)
}

// upsertTriggerEnvVar updates or appends an environment variable in the trigger container.
func upsertTriggerEnvVar(cj *batchv1.CronJob, name, value string) {
	for i := range cj.Spec.JobTemplate.Spec.Template.Spec.Containers {
		container := &cj.Spec.JobTemplate.Spec.Template.Spec.Containers[i]
		if container.Name == "trigger" {
			for j := range container.Env {
				if container.Env[j].Name == name {
					container.Env[j].Value = value
					return
				}
			}
			container.Env = append(container.Env, corev1.EnvVar{Name: name, Value: value})
			return
		}
	}
	log.Printf("Warning: trigger container not found in CronJob %s/%s while setting %s", cj.Namespace, cj.Name, name)
}

// sanitizeLabelValue ensures a string is safe for use as a Kubernetes label value.
// Label values must be 63 characters or less, start and end with alphanumeric,
// and contain only alphanumeric, '-', '_', or '.'.
func sanitizeLabelValue(s string) string {
	if s == "" {
		return "unknown"
	}
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' || ch == '.' {
			result = append(result, ch)
		} else {
			result = append(result, '_')
		}
	}
	if len(result) > 63 {
		result = result[:63]
	}
	// Ensure starts and ends with alphanumeric
	for len(result) > 0 && !isAlphanumeric(result[0]) {
		result = result[1:]
	}
	for len(result) > 0 && !isAlphanumeric(result[len(result)-1]) {
		result = result[:len(result)-1]
	}
	if len(result) == 0 {
		return "unknown"
	}
	return string(result)
}

func isAlphanumeric(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

// isValidCronExpression validates that a string looks like a valid 5-field cron expression.
// It checks structure (5 space-separated fields) and that each field contains only valid
// cron characters. Full semantic validation is left to Kubernetes.
func isValidCronExpression(expr string) bool {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return false
	}
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return false
	}
	for _, field := range fields {
		if len(field) == 0 || len(field) > 20 {
			return false
		}
		for _, ch := range field {
			if !strings.ContainsRune("0123456789*,-/JANFEBMARAPRMAYJUNJULAUGSEPOCTNOVDECMONTUEWEDTHUFRISATSUN", ch) &&
				(ch < 'a' || ch > 'z') {
				return false
			}
		}
	}
	return true
}
