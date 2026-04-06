// Package ldap provides LDAP search and caching for user and group lookups.
package ldap

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	goldap "github.com/go-ldap/ldap/v3"
)

const (
	defaultConnTimeout     = 5 * time.Second
	defaultQueryTimeoutSec = 3 // LDAP search time limit in seconds
	defaultMaxResults      = 10
	defaultCacheTTL        = 5 * time.Minute
	MinQueryLength         = 2
	maxQueryLength         = 50

	cacheKeyUserSearch  = "users:"
	cacheKeyGroupSearch = "groups:"
	cacheKeyUser        = "user:"
)

// userAttributes is the list of LDAP attributes to fetch for user entries.
var userAttributes = []string{"uid", "cn", "mail", "title", "rhatSocialURL", "memberOf"}

// LDAPUser represents a user entry from LDAP.
type LDAPUser struct {
	UID            string   `json:"uid"`
	FullName       string   `json:"fullName"`
	Email          string   `json:"email"`
	Title          string   `json:"title"`
	GitHubUsername string   `json:"githubUsername"`
	Groups         []string `json:"groups"`
}

// LDAPGroup represents a group entry from LDAP.
type LDAPGroup struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// cacheEntry holds a cached result with expiry.
type cacheEntry struct {
	value     any
	expiresAt time.Time
}

// Client provides LDAP search functionality with in-memory caching.
type Client struct {
	url          string // explicit LDAP URL (fallback if srvDomain is empty)
	srvDomain    string // domain for DNS SRV lookup (e.g. "ipa.redhat.com")
	baseDN       string
	groupBaseDN  string
	bindDN       string
	bindPassword string
	tlsConfig    *tls.Config
	cache        sync.Map
	cacheTTL     time.Duration
}

// NewClient creates a new LDAP client.
// url is the LDAP server URL, used as a fallback when srvDomain is empty.
// srvDomain enables DNS SRV discovery (e.g. "ipa.redhat.com" queries _ldap._tcp.ipa.redhat.com).
// baseDN is the base DN for user searches (e.g. "cn=users,cn=accounts,dc=ipa,dc=redhat,dc=com").
// groupBaseDN is the base DN for group searches. If empty, it is derived from
// baseDN by replacing the first component with "cn=groups".
// bindDN and bindPassword are used for authenticated binds (required for IPA).
// caCertPath is an optional path to a PEM CA certificate file to trust.
func NewClient(url, srvDomain, baseDN, groupBaseDN, bindDN, bindPassword, caCertPath string) (*Client, error) {
	if groupBaseDN == "" {
		if parts := strings.SplitN(baseDN, ",", 2); len(parts) == 2 {
			groupBaseDN = "cn=groups," + parts[1]
		}
	}

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	if caCertPath != "" {
		rootCAs, err := x509.SystemCertPool()
		if err != nil {
			rootCAs = x509.NewCertPool()
		}
		caCert, err := os.ReadFile(caCertPath)
		if err != nil {
			return nil, fmt.Errorf("ldap read CA cert %s: %w", caCertPath, err)
		}
		if !rootCAs.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("ldap CA cert %s: no valid PEM certificates found", caCertPath)
		}
		tlsConfig.RootCAs = rootCAs
	}

	return &Client{
		url:          url,
		srvDomain:    srvDomain,
		baseDN:       baseDN,
		groupBaseDN:  groupBaseDN,
		bindDN:       bindDN,
		bindPassword: bindPassword,
		tlsConfig:    tlsConfig,
		cacheTTL:     defaultCacheTTL,
	}, nil
}

// connect dials an LDAP server (discovered via SRV or explicit URL) and performs an authenticated bind.
func (c *Client) connect() (*goldap.Conn, error) {
	conn, err := c.dial()
	if err != nil {
		return nil, err
	}

	if c.bindDN != "" {
		if err := conn.Bind(c.bindDN, c.bindPassword); err != nil {
			conn.Close()
			return nil, fmt.Errorf("ldap bind: %w", err)
		}
	}

	return conn, nil
}

// dial connects to an LDAP server. If srvDomain is configured, it discovers
// servers via DNS SRV records (_ldap._tcp.<domain>) and tries each in
// priority/weight order using LDAPS (port 636). Falls back to the explicit URL.
func (c *Client) dial() (*goldap.Conn, error) {
	dialOpts := []goldap.DialOpt{
		goldap.DialWithTLSConfig(c.tlsConfig),
		goldap.DialWithDialer(&net.Dialer{Timeout: defaultConnTimeout}),
	}

	if c.srvDomain == "" {
		conn, err := goldap.DialURL(c.url, dialOpts...)
		if err != nil {
			return nil, fmt.Errorf("ldap dial %s: %w", c.url, err)
		}
		return conn, nil
	}

	_, addrs, err := net.LookupSRV("ldap", "tcp", c.srvDomain)
	if err != nil || len(addrs) == 0 {
		if c.url != "" {
			conn, dialErr := goldap.DialURL(c.url, dialOpts...)
			if dialErr != nil {
				return nil, fmt.Errorf("ldap SRV lookup failed (%v) and fallback dial %s failed: %w", err, c.url, dialErr)
			}
			return conn, nil
		}
		return nil, fmt.Errorf("ldap SRV lookup _ldap._tcp.%s: %w", c.srvDomain, err)
	}

	// SRV records return port 389 (plain LDAP); connect on 636 for LDAPS.
	var lastErr error
	for _, addr := range addrs {
		host := strings.TrimSuffix(addr.Target, ".")
		url := fmt.Sprintf("ldaps://%s:636", host)
		conn, dialErr := goldap.DialURL(url, dialOpts...)
		if dialErr != nil {
			lastErr = fmt.Errorf("ldap dial %s: %w", url, dialErr)
			continue
		}
		return conn, nil
	}
	return nil, fmt.Errorf("ldap all SRV targets for %s failed, last error: %w", c.srvDomain, lastErr)
}

// cacheGet returns a cached value if it exists and hasn't expired.
func (c *Client) cacheGet(key string) (any, bool) {
	val, ok := c.cache.Load(key)
	if !ok {
		return nil, false
	}
	entry, ok := val.(cacheEntry)
	if !ok {
		c.cache.Delete(key)
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.cache.Delete(key)
		return nil, false
	}
	return entry.value, true
}

// cacheSet stores a value in the cache with TTL.
func (c *Client) cacheSet(key string, value any) {
	c.cache.Store(key, cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(c.cacheTTL),
	})
}

// entryToUser converts an LDAP entry into an LDAPUser struct.
func entryToUser(entry *goldap.Entry) LDAPUser {
	user := LDAPUser{
		UID:      entry.GetAttributeValue("uid"),
		FullName: entry.GetAttributeValue("cn"),
		Email:    entry.GetAttributeValue("mail"),
		Title:    entry.GetAttributeValue("title"),
	}
	for _, socialURL := range entry.GetAttributeValues("rhatSocialURL") {
		if gh := ParseGitHubUsername(socialURL); gh != "" {
			user.GitHubUsername = gh
			break
		}
	}
	seen := make(map[string]struct{})
	for _, dn := range entry.GetAttributeValues("memberOf") {
		if cn := extractCNFromDN(dn); cn != "" {
			if _, dup := seen[cn]; !dup {
				seen[cn] = struct{}{}
				user.Groups = append(user.Groups, cn)
			}
		}
	}
	return user
}

// SearchUsers searches for users matching the query string.
func (c *Client) SearchUsers(query string) ([]LDAPUser, error) {
	query = sanitizeQuery(query)
	if len(query) < MinQueryLength {
		return nil, nil
	}

	cacheKey := cacheKeyUserSearch + query
	if cached, ok := c.cacheGet(cacheKey); ok {
		if users, ok := cached.([]LDAPUser); ok {
			return users, nil
		}
	}

	conn, err := c.connect()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	searchReq := goldap.NewSearchRequest(
		c.baseDN,
		goldap.ScopeWholeSubtree,
		goldap.NeverDerefAliases,
		defaultMaxResults,
		defaultQueryTimeoutSec,
		false,
		UserSearchFilter(query),
		userAttributes,
		nil,
	)

	result, err := conn.Search(searchReq)
	if err != nil && !goldap.IsErrorWithCode(err, goldap.LDAPResultSizeLimitExceeded) {
		return nil, fmt.Errorf("ldap user search: %w", err)
	}

	users := make([]LDAPUser, 0, len(result.Entries))
	for _, entry := range result.Entries {
		users = append(users, entryToUser(entry))
	}

	c.cacheSet(cacheKey, users)
	return users, nil
}

// SearchGroups searches for groups matching the query string.
// Searches the cn attribute with a prefix match in the groups base DN.
func (c *Client) SearchGroups(query string) ([]LDAPGroup, error) {
	query = sanitizeQuery(query)
	if len(query) < MinQueryLength {
		return nil, nil
	}

	cacheKey := cacheKeyGroupSearch + query
	if cached, ok := c.cacheGet(cacheKey); ok {
		if groups, ok := cached.([]LDAPGroup); ok {
			return groups, nil
		}
	}

	conn, err := c.connect()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	searchReq := goldap.NewSearchRequest(
		c.groupBaseDN,
		goldap.ScopeWholeSubtree,
		goldap.NeverDerefAliases,
		defaultMaxResults,
		defaultQueryTimeoutSec,
		false,
		GroupSearchFilter(query),
		[]string{"cn", "description"},
		nil,
	)

	result, err := conn.Search(searchReq)
	if err != nil && !goldap.IsErrorWithCode(err, goldap.LDAPResultSizeLimitExceeded) {
		return nil, fmt.Errorf("ldap group search: %w", err)
	}

	groups := make([]LDAPGroup, 0, len(result.Entries))
	for _, entry := range result.Entries {
		groups = append(groups, LDAPGroup{
			Name:        entry.GetAttributeValue("cn"),
			Description: entry.GetAttributeValue("description"),
		})
	}

	c.cacheSet(cacheKey, groups)
	return groups, nil
}

// GetUser retrieves a single user by exact UID match.
func (c *Client) GetUser(uid string) (*LDAPUser, error) {
	uid = sanitizeQuery(uid)
	if uid == "" {
		return nil, nil
	}

	cacheKey := cacheKeyUser + uid
	if cached, ok := c.cacheGet(cacheKey); ok {
		if user, ok := cached.(*LDAPUser); ok {
			return user, nil
		}
	}

	conn, err := c.connect()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	escaped := goldap.EscapeFilter(uid)
	filter := fmt.Sprintf("(uid=%s)", escaped)

	searchReq := goldap.NewSearchRequest(
		c.baseDN,
		goldap.ScopeWholeSubtree,
		goldap.NeverDerefAliases,
		1,
		defaultQueryTimeoutSec,
		false,
		filter,
		userAttributes,
		nil,
	)

	result, err := conn.Search(searchReq)
	if err != nil {
		return nil, fmt.Errorf("ldap user get: %w", err)
	}

	if len(result.Entries) == 0 {
		return nil, nil
	}

	user := entryToUser(result.Entries[0])
	c.cacheSet(cacheKey, &user)
	return &user, nil
}

// ParseGitHubUsername extracts a GitHub username from an rhatSocialURL value.
// Expected format: "Github->https://github.com/<username>"
func ParseGitHubUsername(socialURL string) string {
	prefix := "Github->https://github.com/"
	if !strings.HasPrefix(socialURL, prefix) {
		return ""
	}
	username := strings.TrimPrefix(socialURL, prefix)
	// Remove trailing slashes and any path segments
	username = strings.TrimRight(username, "/")
	if idx := strings.Index(username, "/"); idx >= 0 {
		username = username[:idx]
	}
	return username
}

// extractCNFromDN extracts the CN value from a distinguished name.
// e.g. "cn=mygroup,ou=managedGroups,dc=redhat,dc=com" -> "mygroup"
func extractCNFromDN(dn string) string {
	parts := strings.Split(dn, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(strings.ToLower(part), "cn=") {
			return part[3:]
		}
	}
	return ""
}

// sanitizeQuery cleans and truncates a search query.
func sanitizeQuery(q string) string {
	q = strings.TrimSpace(q)
	q = strings.NewReplacer("\n", "", "\r", "").Replace(q)
	if len(q) > maxQueryLength {
		q = q[:maxQueryLength]
	}
	return q
}

// UserSearchFilter builds the LDAP filter string for user searches.
// Searches uid, givenName (first name), and sn (last name) with substring matching.
// Multi-word queries are split so each word must match at least one field (AND).
// Exported for testing.
func UserSearchFilter(query string) string {
	words := strings.Fields(query)
	if len(words) == 0 {
		return "(uid=*)"
	}
	if len(words) == 1 {
		escaped := goldap.EscapeFilter(words[0])
		return fmt.Sprintf("(|(uid=*%s*)(givenName=*%s*)(sn=*%s*))", escaped, escaped, escaped)
	}
	// Multiple words: each word must match at least one field
	parts := make([]string, 0, len(words))
	for _, w := range words {
		escaped := goldap.EscapeFilter(w)
		parts = append(parts, fmt.Sprintf("(|(uid=*%s*)(givenName=*%s*)(sn=*%s*))", escaped, escaped, escaped))
	}
	return "(&" + strings.Join(parts, "") + ")"
}

// GroupSearchFilter builds the LDAP filter string for group searches.
// Exported for testing.
func GroupSearchFilter(query string) string {
	escaped := goldap.EscapeFilter(query)
	return fmt.Sprintf("(cn=%s*)", escaped)
}
