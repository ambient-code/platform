package handlers

import (
	"fmt"
	"log"
	"net/http"

	"ambient-code-backend/ldap"

	"github.com/gin-gonic/gin"
)

// LDAPClient is the shared LDAP client instance, initialized in main.go when LDAP_URL is set.
// Access is gated in the frontend by the "ldap.autocomplete.enabled" workspace feature flag.
var LDAPClient *ldap.Client

// SearchLDAPUsers handles GET /api/ldap/users?q={query}
func SearchLDAPUsers(c *gin.Context) {
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	query := c.Query("q")
	if len(query) < ldap.MinQueryLength {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("query must be at least %d characters", ldap.MinQueryLength)})
		return
	}

	if LDAPClient == nil {
		c.JSON(http.StatusOK, gin.H{"users": []ldap.LDAPUser{}})
		return
	}

	users, err := LDAPClient.SearchUsers(query)
	if err != nil {
		log.Printf("LDAP user search error for query %q: %v", query, err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "LDAP search unavailable"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

// SearchLDAPGroups handles GET /api/ldap/groups?q={query}
func SearchLDAPGroups(c *gin.Context) {
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	query := c.Query("q")
	if len(query) < ldap.MinQueryLength {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("query must be at least %d characters", ldap.MinQueryLength)})
		return
	}

	if LDAPClient == nil {
		c.JSON(http.StatusOK, gin.H{"groups": []ldap.LDAPGroup{}})
		return
	}

	groups, err := LDAPClient.SearchGroups(query)
	if err != nil {
		log.Printf("LDAP group search error for query %q: %v", query, err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "LDAP search unavailable"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"groups": groups})
}

// GetLDAPUser handles GET /api/ldap/users/:uid
func GetLDAPUser(c *gin.Context) {
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	uid := c.Param("uid")
	if uid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "uid is required"})
		return
	}

	if LDAPClient == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "LDAP not configured"})
		return
	}

	user, err := LDAPClient.GetUser(uid)
	if err != nil {
		log.Printf("LDAP user get error for uid %q: %v", uid, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to look up user"})
		return
	}

	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}
