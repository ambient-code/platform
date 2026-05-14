package imageref

import (
	"fmt"
	"strings"
)

func ParseImageReference(ref string) (registry, repository, tagOrDigest string, err error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", "", "", fmt.Errorf("empty image reference")
	}

	if idx := strings.Index(ref, "@sha256:"); idx >= 0 {
		tagOrDigest = ref[idx:]
		ref = ref[:idx]
	} else if idx := strings.LastIndex(ref, ":"); idx >= 0 {
		candidate := ref[idx+1:]
		if !strings.Contains(candidate, "/") {
			tagOrDigest = candidate
			ref = ref[:idx]
		}
	}

	parts := strings.Split(ref, "/")
	switch {
	case len(parts) == 1:
		registry = "docker.io"
		repository = "library/" + parts[0]
	case len(parts) == 2 && !strings.Contains(parts[0], ".") && !strings.Contains(parts[0], ":"):
		registry = "docker.io"
		repository = ref
	default:
		registry = parts[0]
		repository = strings.Join(parts[1:], "/")
	}

	if repository == "" {
		return "", "", "", fmt.Errorf("invalid image reference: missing repository in %q", ref)
	}

	return registry, repository, tagOrDigest, nil
}

func ValidateRegistryAllowlist(ref string, allowlist []string) error {
	if len(allowlist) == 0 {
		return nil
	}

	registry, _, _, err := ParseImageReference(ref)
	if err != nil {
		return err
	}

	for _, allowed := range allowlist {
		if strings.EqualFold(registry, strings.TrimSpace(allowed)) {
			return nil
		}
	}

	return fmt.Errorf("registry %q is not in the allowed list %v", registry, allowlist)
}

func DetermineImagePullPolicy(ref string) string {
	if strings.Contains(ref, "@sha256:") {
		return "IfNotPresent"
	}
	if strings.HasPrefix(ref, "localhost/") {
		return "IfNotPresent"
	}
	return "Always"
}

func ParseAllowlist(csv string) []string {
	csv = strings.TrimSpace(csv)
	if csv == "" {
		return nil
	}
	parts := strings.Split(csv, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
