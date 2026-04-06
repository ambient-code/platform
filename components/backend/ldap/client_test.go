package ldap

import (
	"testing"
	"time"
)

func TestParseGitHubUsername(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid github url",
			input:    "Github->https://github.com/jdoe",
			expected: "jdoe",
		},
		{
			name:     "valid github url with trailing slash",
			input:    "Github->https://github.com/jdoe/",
			expected: "jdoe",
		},
		{
			name:     "valid github url with extra path",
			input:    "Github->https://github.com/jdoe/repos",
			expected: "jdoe",
		},
		{
			name:     "non-github social url",
			input:    "Twitter->https://twitter.com/jdoe",
			expected: "",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "partial prefix",
			input:    "Github->https://github.com/",
			expected: "",
		},
		{
			name:     "different case prefix",
			input:    "github->https://github.com/jdoe",
			expected: "",
		},
		{
			name:     "linkedin url",
			input:    "LinkedIn->https://www.linkedin.com/in/someone",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseGitHubUsername(tt.input)
			if result != tt.expected {
				t.Errorf("ParseGitHubUsername(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractCNFromDN(t *testing.T) {
	tests := []struct {
		name     string
		dn       string
		expected string
	}{
		{
			name:     "standard group DN",
			dn:       "cn=aipcc-eng-all,ou=managedGroups,dc=redhat,dc=com",
			expected: "aipcc-eng-all",
		},
		{
			name:     "CN with spaces",
			dn:       "cn=My Group, ou=groups, dc=example, dc=com",
			expected: "My Group",
		},
		{
			name:     "no CN",
			dn:       "ou=managedGroups,dc=redhat,dc=com",
			expected: "",
		},
		{
			name:     "empty string",
			dn:       "",
			expected: "",
		},
		{
			name:     "uppercase CN",
			dn:       "CN=TestGroup,ou=groups,dc=example,dc=com",
			expected: "TestGroup",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractCNFromDN(tt.dn)
			if result != tt.expected {
				t.Errorf("extractCNFromDN(%q) = %q, want %q", tt.dn, result, tt.expected)
			}
		})
	}
}

func TestSanitizeQuery(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal query",
			input:    "jdo",
			expected: "jdo",
		},
		{
			name:     "leading/trailing spaces",
			input:    "  jdo  ",
			expected: "jdo",
		},
		{
			name:     "long query truncated",
			input:    "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz1234567890",
			expected: "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwx",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "embedded newline stripped",
			input:    "foo\nbar",
			expected: "foobar",
		},
		{
			name:     "embedded carriage return stripped",
			input:    "foo\rbar",
			expected: "foobar",
		},
		{
			name:     "CRLF stripped",
			input:    "foo\r\nbar",
			expected: "foobar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeQuery(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeQuery(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestUserSearchFilter(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "simple query",
			query:    "jdoe",
			expected: "(|(uid=*jdoe*)(givenName=*jdoe*)(sn=*jdoe*))",
		},
		{
			name:     "first name query",
			query:    "Jane",
			expected: "(|(uid=*Jane*)(givenName=*Jane*)(sn=*Jane*))",
		},
		{
			name:     "query with special chars",
			query:    "user(test)",
			expected: `(|(uid=*user\28test\29*)(givenName=*user\28test\29*)(sn=*user\28test\29*))`,
		},
		{
			name:     "query with asterisk",
			query:    "user*",
			expected: `(|(uid=*user\2a*)(givenName=*user\2a*)(sn=*user\2a*))`,
		},
		{
			name:     "multi-word query",
			query:    "Jane Do",
			expected: "(&(|(uid=*Jane*)(givenName=*Jane*)(sn=*Jane*))(|(uid=*Do*)(givenName=*Do*)(sn=*Do*)))",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UserSearchFilter(tt.query)
			if result != tt.expected {
				t.Errorf("UserSearchFilter(%q) = %q, want %q", tt.query, result, tt.expected)
			}
		})
	}
}

func TestGroupSearchFilter(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "simple query",
			query:    "aipcc",
			expected: "(cn=aipcc*)",
		},
		{
			name:     "query with special chars",
			query:    "group(test)",
			expected: `(cn=group\28test\29*)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GroupSearchFilter(tt.query)
			if result != tt.expected {
				t.Errorf("GroupSearchFilter(%q) = %q, want %q", tt.query, result, tt.expected)
			}
		})
	}
}

func TestCacheTTL(t *testing.T) {
	client := &Client{
		cacheTTL: 50 * time.Millisecond,
	}

	client.cacheSet("test-key", "test-value")

	// Should be in cache
	val, ok := client.cacheGet("test-key")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if val.(string) != "test-value" {
		t.Errorf("expected 'test-value', got %v", val)
	}

	// Wait for expiry
	time.Sleep(60 * time.Millisecond)

	// Should be expired
	_, ok = client.cacheGet("test-key")
	if ok {
		t.Fatal("expected cache miss after TTL")
	}
}

func TestCacheMiss(t *testing.T) {
	client := &Client{
		cacheTTL: defaultCacheTTL,
	}

	_, ok := client.cacheGet("nonexistent")
	if ok {
		t.Fatal("expected cache miss for nonexistent key")
	}
}

func TestNewClient(t *testing.T) {
	client, err := NewClient("ldaps://ldap.example.com", "", "cn=users,cn=accounts,dc=ipa,dc=redhat,dc=com", "", "uid=svc,cn=users,cn=accounts,dc=ipa,dc=redhat,dc=com", "pass", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if client.url != "ldaps://ldap.example.com" {
		t.Errorf("expected url 'ldaps://ldap.example.com', got %q", client.url)
	}
	if client.baseDN != "cn=users,cn=accounts,dc=ipa,dc=redhat,dc=com" {
		t.Errorf("expected baseDN 'cn=users,cn=accounts,dc=ipa,dc=redhat,dc=com', got %q", client.baseDN)
	}
	if client.groupBaseDN != "cn=groups,cn=accounts,dc=ipa,dc=redhat,dc=com" {
		t.Errorf("expected groupBaseDN 'cn=groups,cn=accounts,dc=ipa,dc=redhat,dc=com', got %q", client.groupBaseDN)
	}
	if client.bindDN != "uid=svc,cn=users,cn=accounts,dc=ipa,dc=redhat,dc=com" {
		t.Errorf("expected bindDN 'uid=svc,cn=users,cn=accounts,dc=ipa,dc=redhat,dc=com', got %q", client.bindDN)
	}
	if client.cacheTTL != defaultCacheTTL {
		t.Errorf("expected cacheTTL %v, got %v", defaultCacheTTL, client.cacheTTL)
	}
}

func TestNewClientExplicitGroupBaseDN(t *testing.T) {
	client, err := NewClient("ldaps://ldap.example.com", "", "cn=users,cn=accounts,dc=example,dc=com", "cn=groups,cn=accounts,dc=example,dc=com", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if client.groupBaseDN != "cn=groups,cn=accounts,dc=example,dc=com" {
		t.Errorf("expected explicit groupBaseDN 'cn=groups,cn=accounts,dc=example,dc=com', got %q", client.groupBaseDN)
	}
}

func TestSearchUsersShortQuery(t *testing.T) {
	client, err := NewClient("ldaps://ldap.example.com", "", "cn=users,cn=accounts,dc=ipa,dc=redhat,dc=com", "", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Query too short should return nil without connecting
	users, err := client.SearchUsers("m")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if users != nil {
		t.Errorf("expected nil for short query, got %v", users)
	}
}

func TestSearchGroupsShortQuery(t *testing.T) {
	client, err := NewClient("ldaps://ldap.example.com", "", "cn=users,cn=accounts,dc=ipa,dc=redhat,dc=com", "", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	groups, err := client.SearchGroups("a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if groups != nil {
		t.Errorf("expected nil for short query, got %v", groups)
	}
}

func TestGetUserEmptyUID(t *testing.T) {
	client, err := NewClient("ldaps://ldap.example.com", "", "cn=users,cn=accounts,dc=ipa,dc=redhat,dc=com", "", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	user, err := client.GetUser("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user != nil {
		t.Errorf("expected nil for empty uid, got %v", user)
	}
}
