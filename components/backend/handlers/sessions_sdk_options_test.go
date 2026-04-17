//go:build test

package handlers

import (
	test_constants "ambient-code-backend/tests/constants"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SDK Options", Label(test_constants.LabelUnit, test_constants.LabelHandlers, test_constants.LabelSessions), func() {

	Describe("filterSdkOptions", func() {
		It("should pass through valid keys unchanged", func() {
			input := map[string]interface{}{
				"system_prompt":  "You are helpful",
				"max_turns":      float64(10),
				"max_budget_usd": float64(5.0),
			}
			result, err := filterSdkOptions(input)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(3))
			Expect(result["system_prompt"]).To(Equal("You are helpful"))
			Expect(result["max_turns"]).To(Equal(float64(10)))
			Expect(result["max_budget_usd"]).To(Equal(float64(5.0)))
		})

		It("should silently drop unknown keys", func() {
			input := map[string]interface{}{
				"system_prompt":   "valid",
				"unknown_key":     "dropped",
				"another_unknown": 42,
			}
			result, err := filterSdkOptions(input)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result).To(HaveKey("system_prompt"))
			Expect(result).NotTo(HaveKey("unknown_key"))
			Expect(result).NotTo(HaveKey("another_unknown"))
		})

		It("should drop platform-internal keys (cwd, resume, mcp_servers, etc.)", func() {
			input := map[string]interface{}{
				"cwd":                         "/some/path",
				"resume":                      true,
				"mcp_servers":                 []interface{}{},
				"setting_sources":             "something",
				"continue_conversation":       true,
				"add_dirs":                    []interface{}{"/a"},
				"cli_path":                    "/usr/bin/claude",
				"settings":                    map[string]interface{}{},
				"permission_prompt_tool_name": "tool",
				"fork_session":                true,
				"api_key":                     "sk-secret",
				"stderr":                      "pipe",
				"system_prompt":               "valid key",
			}
			result, err := filterSdkOptions(input)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result).To(HaveKey("system_prompt"))
		})

		It("should return nil for empty map", func() {
			result, err := filterSdkOptions(map[string]interface{}{})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeNil())
		})

		It("should return nil for nil input", func() {
			result, err := filterSdkOptions(nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeNil())
		})

		It("should return nil when all keys are filtered out", func() {
			input := map[string]interface{}{
				"unknown_key": "value",
				"api_key":     "secret",
			}
			result, err := filterSdkOptions(input)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeNil())
		})

		It("should return error when a valid key has wrong type", func() {
			input := map[string]interface{}{
				"max_turns": "not a number",
			}
			_, err := filterSdkOptions(input)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("max_turns"))
		})
	})

	Describe("validateSdkOptionValue", func() {

		// --- String keys ---

		Context("string keys (system_prompt, permission_mode, effort, user)", func() {
			stringKeys := []string{"permission_mode", "effort", "user"}

			It("should accept string values", func() {
				for _, key := range stringKeys {
					err := validateSdkOptionValue(key, "valid string")
					Expect(err).NotTo(HaveOccurred(), "key=%s should accept string", key)
				}
			})

			It("should reject numeric values for string keys", func() {
				for _, key := range stringKeys {
					err := validateSdkOptionValue(key, float64(42))
					Expect(err).To(HaveOccurred(), "key=%s should reject number", key)
				}
			})
		})

		Context("system_prompt (string or map)", func() {
			It("should accept a string value", func() {
				err := validateSdkOptionValue("system_prompt", "You are helpful")
				Expect(err).NotTo(HaveOccurred())
			})

			It("should accept a map value (preset format)", func() {
				preset := map[string]interface{}{
					"type": "preset",
					"name": "my-preset",
				}
				err := validateSdkOptionValue("system_prompt", preset)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should reject a numeric value", func() {
				err := validateSdkOptionValue("system_prompt", float64(42))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("system_prompt"))
			})
		})

		// --- Numeric keys ---

		Context("numeric keys (max_turns, max_budget_usd, max_buffer_size)", func() {
			numericKeys := []string{"max_turns", "max_budget_usd", "max_buffer_size"}

			It("should accept float64 values", func() {
				for _, key := range numericKeys {
					err := validateSdkOptionValue(key, float64(10))
					Expect(err).NotTo(HaveOccurred(), "key=%s should accept float64", key)
				}
			})

			It("should accept int values", func() {
				for _, key := range numericKeys {
					err := validateSdkOptionValue(key, 10)
					Expect(err).NotTo(HaveOccurred(), "key=%s should accept int", key)
				}
			})

			It("should reject string values for numeric keys", func() {
				for _, key := range numericKeys {
					err := validateSdkOptionValue(key, "not a number")
					Expect(err).To(HaveOccurred(), "key=%s should reject string", key)
				}
			})

			It("should reject bool values for numeric keys", func() {
				for _, key := range numericKeys {
					err := validateSdkOptionValue(key, true)
					Expect(err).To(HaveOccurred(), "key=%s should reject bool", key)
				}
			})
		})

		// --- Bool keys ---

		Context("bool keys (include_partial_messages, enable_file_checkpointing)", func() {
			boolKeys := []string{"include_partial_messages", "enable_file_checkpointing"}

			It("should accept bool values", func() {
				for _, key := range boolKeys {
					err := validateSdkOptionValue(key, true)
					Expect(err).NotTo(HaveOccurred(), "key=%s should accept true", key)
					err = validateSdkOptionValue(key, false)
					Expect(err).NotTo(HaveOccurred(), "key=%s should accept false", key)
				}
			})

			It("should reject string values for bool keys", func() {
				for _, key := range boolKeys {
					err := validateSdkOptionValue(key, "true")
					Expect(err).To(HaveOccurred(), "key=%s should reject string", key)
				}
			})

			It("should reject numeric values for bool keys", func() {
				for _, key := range boolKeys {
					err := validateSdkOptionValue(key, float64(1))
					Expect(err).To(HaveOccurred(), "key=%s should reject number", key)
				}
			})
		})

		// --- Slice keys ---

		Context("slice keys (allowed_tools, disallowed_tools, betas, plugins)", func() {
			sliceKeys := []string{"allowed_tools", "disallowed_tools", "betas", "plugins"}

			It("should accept []interface{} values", func() {
				for _, key := range sliceKeys {
					err := validateSdkOptionValue(key, []interface{}{"item1", "item2"})
					Expect(err).NotTo(HaveOccurred(), "key=%s should accept slice", key)
				}
			})

			It("should accept empty []interface{}", func() {
				for _, key := range sliceKeys {
					err := validateSdkOptionValue(key, []interface{}{})
					Expect(err).NotTo(HaveOccurred(), "key=%s should accept empty slice", key)
				}
			})

			It("should reject string values for slice keys", func() {
				for _, key := range sliceKeys {
					err := validateSdkOptionValue(key, "not a slice")
					Expect(err).To(HaveOccurred(), "key=%s should reject string", key)
				}
			})
		})

		// --- Complex object keys ---

		Context("complex object keys (thinking, sandbox, output_format, hooks, agents, env, extra_args, tools)", func() {
			complexKeys := []string{"thinking", "sandbox", "output_format", "hooks", "agents", "env", "extra_args", "tools"}

			It("should accept map[string]interface{} values", func() {
				for _, key := range complexKeys {
					val := map[string]interface{}{"nested": "value"}
					err := validateSdkOptionValue(key, val)
					Expect(err).NotTo(HaveOccurred(), "key=%s should accept map", key)
				}
			})

			It("should pass through non-map values for complex keys (JSON handles them)", func() {
				for _, key := range complexKeys {
					// Complex keys accept anything that JSON can serialize
					err := validateSdkOptionValue(key, "string-value")
					Expect(err).NotTo(HaveOccurred(), "key=%s should pass through string", key)
					err = validateSdkOptionValue(key, float64(42))
					Expect(err).NotTo(HaveOccurred(), "key=%s should pass through number", key)
					err = validateSdkOptionValue(key, true)
					Expect(err).NotTo(HaveOccurred(), "key=%s should pass through bool", key)
				}
			})
		})

		// --- nil values ---

		Context("nil values", func() {
			It("should always pass validation for any key", func() {
				keys := []string{
					"system_prompt", "permission_mode", "max_turns",
					"max_budget_usd", "include_partial_messages",
					"allowed_tools", "thinking", "effort", "user",
				}
				for _, key := range keys {
					err := validateSdkOptionValue(key, nil)
					Expect(err).NotTo(HaveOccurred(), "key=%s should accept nil", key)
				}
			})
		})
	})
})
