package imageref

import (
	"testing"
)

func TestParseImageReference(t *testing.T) {
	tests := []struct {
		name     string
		ref      string
		wantReg  string
		wantRepo string
		wantTag  string
		wantErr  bool
	}{
		{"empty", "", "", "", "", true},
		{"simple", "nginx", "docker.io", "library/nginx", "", false},
		{"with tag", "nginx:latest", "docker.io", "library/nginx", "latest", false},
		{"docker hub user", "myuser/myimage:v1", "docker.io", "myuser/myimage", "v1", false},
		{"full registry", "quay.io/ambient_code/runner:v2", "quay.io", "ambient_code/runner", "v2", false},
		{"digest", "quay.io/ambient_code/runner@sha256:abc123", "quay.io", "ambient_code/runner", "@sha256:abc123", false},
		{"registry with port", "localhost:5000/myimage:dev", "localhost:5000", "myimage", "dev", false},
		{"deep path", "registry.example.com/org/team/image:latest", "registry.example.com", "org/team/image", "latest", false},
		{"no tag", "quay.io/ambient_code/runner", "quay.io", "ambient_code/runner", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg, repo, tag, err := ParseImageReference(tt.ref)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseImageReference(%q) error = %v, wantErr %v", tt.ref, err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if reg != tt.wantReg {
				t.Errorf("registry = %q, want %q", reg, tt.wantReg)
			}
			if repo != tt.wantRepo {
				t.Errorf("repository = %q, want %q", repo, tt.wantRepo)
			}
			if tag != tt.wantTag {
				t.Errorf("tagOrDigest = %q, want %q", tag, tt.wantTag)
			}
		})
	}
}

func TestValidateRegistryAllowlist(t *testing.T) {
	tests := []struct {
		name      string
		ref       string
		allowlist []string
		wantErr   bool
	}{
		{"empty allowlist allows all", "quay.io/img:v1", nil, false},
		{"allowed registry", "quay.io/img:v1", []string{"quay.io", "docker.io"}, false},
		{"denied registry", "evil.io/img:v1", []string{"quay.io", "docker.io"}, true},
		{"case insensitive", "Quay.IO/img:v1", []string{"quay.io"}, false},
		{"docker hub implicit", "nginx:latest", []string{"docker.io"}, false},
		{"docker hub implicit denied", "nginx:latest", []string{"quay.io"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRegistryAllowlist(tt.ref, tt.allowlist)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRegistryAllowlist(%q, %v) error = %v, wantErr %v", tt.ref, tt.allowlist, err, tt.wantErr)
			}
		})
	}
}

func TestDetermineImagePullPolicy(t *testing.T) {
	tests := []struct {
		ref    string
		policy string
	}{
		{"quay.io/img@sha256:abc123", "IfNotPresent"},
		{"localhost/myimage:dev", "IfNotPresent"},
		{"quay.io/img:latest", "Always"},
		{"quay.io/img:v1.2.3", "Always"},
		{"nginx", "Always"},
	}

	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			got := DetermineImagePullPolicy(tt.ref)
			if got != tt.policy {
				t.Errorf("DetermineImagePullPolicy(%q) = %q, want %q", tt.ref, got, tt.policy)
			}
		})
	}
}

func TestParseAllowlist(t *testing.T) {
	tests := []struct {
		csv  string
		want int
	}{
		{"", 0},
		{"quay.io", 1},
		{"quay.io,docker.io", 2},
		{"quay.io, docker.io , gcr.io ", 3},
		{",,,", 0},
	}

	for _, tt := range tests {
		t.Run(tt.csv, func(t *testing.T) {
			got := ParseAllowlist(tt.csv)
			if len(got) != tt.want {
				t.Errorf("ParseAllowlist(%q) returned %d items, want %d", tt.csv, len(got), tt.want)
			}
		})
	}
}
