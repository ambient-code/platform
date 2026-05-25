package trigger

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"ambient-code-operator/internal/types"
)

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal string is lowercased and cleaned",
			input:    "DailyReport",
			expected: "dailyreport",
		},
		{
			name:     "special characters replaced with hyphens",
			input:    "my task!here",
			expected: "my-task-here",
		},
		{
			name:     "consecutive special chars produce single hyphen",
			input:    "hello!!!world",
			expected: "hello-world",
		},
		{
			name:     "trailing hyphens trimmed",
			input:    "hello!",
			expected: "hello",
		},
		{
			name:     "string over 40 chars truncated",
			input:    "abcdefghijklmnopqrstuvwxyz1234567890abcdefghijklmnop",
			expected: "abcdefghijklmnopqrstuvwxyz1234567890abcd",
		},
		{
			name:     "empty string returns run",
			input:    "",
			expected: "run",
		},
		{
			name:     "all special chars returns run",
			input:    "!!!@@@###",
			expected: "run",
		},
		{
			name:     "mixed case lowercased",
			input:    "MyDailyTask",
			expected: "mydailytask",
		},
		{
			name:     "spaces replaced with hyphens",
			input:    "daily jira summary",
			expected: "daily-jira-summary",
		},
		{
			name:     "leading special chars omitted",
			input:    "  hello",
			expected: "hello",
		},
		{
			name:     "digits preserved",
			input:    "task123",
			expected: "task123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeName(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeName_ScheduledSessionUUID(t *testing.T) {
	// Scheduled session names contain UUIDs (e.g. "schedule-cb9522da-ce4d-4bc8-8cc8-5647407288f5").
	// Without sanitization, "session-" + name + "-" + timestamp = 64 chars, exceeding
	// the 63-char K8s Service name limit.  sanitizeName must cap at 40 chars.
	input := "schedule-cb9522da-ce4d-4bc8-8cc8-5647407288f5"
	result := sanitizeName(input)
	if len(result) > 40 {
		t.Errorf("sanitizeName(%q) length = %d, want <= 40", input, len(result))
	}
	// session-{sanitized}-{10-digit-ts} must fit in 63 chars
	svcName := "session-" + result + "-1775052000"
	if len(svcName) > 63 {
		t.Errorf("derived Service name %q length = %d, exceeds 63-char K8s limit", svcName, len(svcName))
	}
}

func TestSanitizeName_TruncationPreservesValidSuffix(t *testing.T) {
	// Verify that truncation to 40 chars does not leave a trailing hyphen
	input := "abcdefghijklmnopqrstuvwxyz1234567890abcd!"
	result := sanitizeName(input)
	if len(result) > 40 {
		t.Errorf("sanitizeName(%q) length = %d, want <= 40", input, len(result))
	}
	if result[len(result)-1] == '-' {
		t.Errorf("sanitizeName(%q) ends with hyphen: %q", input, result)
	}
}

func TestApplyFeatureFlagOverrides(t *testing.T) {
	tests := []struct {
		name            string
		configMapData   map[string]string
		existingEnvVars map[string]interface{}
		expectedEnvVars map[string]interface{}
	}{
		{
			name:            "jira-write enabled sets JIRA_READ_ONLY_MODE to false",
			configMapData:   map[string]string{"jira-write": "true"},
			existingEnvVars: nil,
			expectedEnvVars: map[string]interface{}{"JIRA_READ_ONLY_MODE": "false"},
		},
		{
			name:            "jira-write disabled does not set env var",
			configMapData:   map[string]string{"jira-write": "false"},
			existingEnvVars: nil,
			expectedEnvVars: nil,
		},
		{
			name:            "no overrides ConfigMap does not set env var",
			configMapData:   nil,
			existingEnvVars: nil,
			expectedEnvVars: nil,
		},
		{
			name:            "preserves existing env vars",
			configMapData:   map[string]string{"jira-write": "true"},
			existingEnvVars: map[string]interface{}{"CUSTOM_VAR": "value"},
			expectedEnvVars: map[string]interface{}{
				"CUSTOM_VAR":          "value",
				"JIRA_READ_ONLY_MODE": "false",
			},
		},
		{
			name:            "other flags do not affect env vars",
			configMapData:   map[string]string{"other-flag": "true"},
			existingEnvVars: nil,
			expectedEnvVars: nil,
		},
		{
			name:            "jira-write with non-true value does not set env var",
			configMapData:   map[string]string{"jira-write": "yes"},
			existingEnvVars: nil,
			expectedEnvVars: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake K8s client
			var k8sClient *fake.Clientset
			if tt.configMapData != nil {
				configMap := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      types.FeatureFlagOverridesConfigMap,
						Namespace: "test-namespace",
					},
					Data: tt.configMapData,
				}
				k8sClient = fake.NewSimpleClientset(configMap)
			} else {
				k8sClient = fake.NewSimpleClientset()
			}

			template := map[string]interface{}{}
			if tt.existingEnvVars != nil {
				template["environmentVariables"] = tt.existingEnvVars
			}

			// Apply feature flag overrides
			ctx := context.Background()
			err := applyFeatureFlagOverrides(ctx, k8sClient, "test-namespace", template)
			if err != nil {
				t.Fatalf("applyFeatureFlagOverrides() unexpected error: %v", err)
			}

			envVars, ok := template["environmentVariables"].(map[string]interface{})
			if tt.expectedEnvVars == nil {
				if envVars != nil && len(envVars) > 0 {
					t.Errorf("Expected no environmentVariables, got %v", envVars)
				}
				return
			}

			if !ok {
				t.Fatal("environmentVariables is not a map")
			}

			if len(envVars) != len(tt.expectedEnvVars) {
				t.Errorf("environmentVariables count = %d, want %d", len(envVars), len(tt.expectedEnvVars))
			}

			for key, expectedVal := range tt.expectedEnvVars {
				actualVal, exists := envVars[key]
				if !exists {
					t.Errorf("environmentVariables[%q] missing, want %q", key, expectedVal)
					continue
				}
				if actualVal != expectedVal {
					t.Errorf("environmentVariables[%q] = %q, want %q", key, actualVal, expectedVal)
				}
			}
		})
	}
}
