package handlers

import (
	"log"
	"os"
	"strings"

	"ambient-code-backend/featureflags"
	"ambient-code-backend/jwtauth"
	"ambient-code-backend/server"

	"github.com/gin-gonic/gin"
	authnv1 "k8s.io/api/authentication/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const ssoFeatureFlag = "sso-authentication"

func SSOEnabled() bool {
	if os.Getenv("SSO_ENABLED") == "true" {
		return true
	}
	return featureflags.IsEnabled(ssoFeatureFlag)
}

func buildImpersonatingClients(claims *jwtauth.Claims) (kubernetes.Interface, dynamic.Interface) {
	if BaseKubeConfig == nil {
		log.Printf("SSO: cannot build impersonating clients: BaseKubeConfig is nil")
		return nil, nil
	}

	impersonateUser := claims.PreferredUsername
	if impersonateUser == "" {
		impersonateUser = claims.Email
	}
	if impersonateUser == "" {
		impersonateUser = claims.Sub
	}
	if impersonateUser == "" {
		log.Printf("SSO: JWT has no usable identity claim (email, preferred_username, sub)")
		return nil, nil
	}

	groups := claims.Groups
	if len(groups) == 0 {
		groups = []string{"system:authenticated"}
	}

	cfg := rest.CopyConfig(BaseKubeConfig)
	cfg.Impersonate = rest.ImpersonationConfig{
		UserName: impersonateUser,
		Groups:   groups,
	}

	kc, err1 := kubernetes.NewForConfig(cfg)
	dc, err2 := dynamic.NewForConfig(cfg)
	if err1 != nil || err2 != nil {
		log.Printf("SSO: failed to build impersonating clients for %s: typed=%v dynamic=%v", impersonateUser, err1, err2)
		return nil, nil
	}

	return kc, dc
}

func tokenReviewIdentity(c *gin.Context, token string) (userName string, groups []string, ok bool) {
	if K8sClientMw == nil {
		return "", nil, false
	}

	tr := &authnv1.TokenReview{Spec: authnv1.TokenReviewSpec{Token: token}}
	rv, err := K8sClientMw.AuthenticationV1().TokenReviews().Create(c.Request.Context(), tr, v1.CreateOptions{})
	if err != nil || !rv.Status.Authenticated || rv.Status.Error != "" {
		return "", nil, false
	}

	username := strings.TrimSpace(rv.Status.User.Username)
	if username == "" {
		return "", nil, false
	}

	// For service accounts, resolve the creating user's identity from annotations
	const saPrefix = "system:serviceaccount:"
	if strings.HasPrefix(username, saPrefix) {
		rest := strings.TrimPrefix(username, saPrefix)
		parts := strings.SplitN(rest, ":", 2)
		if len(parts) == 2 {
			sa, err := K8sClientMw.CoreV1().ServiceAccounts(parts[0]).Get(c.Request.Context(), parts[1], v1.GetOptions{})
			if err == nil && sa.Annotations != nil {
				if uid := sa.Annotations["ambient-code.io/created-by-user-id"]; uid != "" {
					username = uid
				}
			}
		}
	}

	return username, rv.Status.User.Groups, true
}

func buildImpersonatingClientsFromIdentity(userName string, groups []string) (kubernetes.Interface, dynamic.Interface) {
	if BaseKubeConfig == nil || userName == "" {
		return nil, nil
	}

	cfg := rest.CopyConfig(BaseKubeConfig)
	cfg.Impersonate = rest.ImpersonationConfig{
		UserName: userName,
		Groups:   groups,
	}

	kc, err1 := kubernetes.NewForConfig(cfg)
	dc, err2 := dynamic.NewForConfig(cfg)
	if err1 != nil || err2 != nil {
		log.Printf("SSO: failed to build impersonating clients for identity %s: typed=%v dynamic=%v", userName, err1, err2)
		return nil, nil
	}

	return kc, dc
}

func setIdentityFromTokenReview(c *gin.Context, userName string, groups []string) {
	c.Set("userID", server.SanitizeUserID(userName))
	c.Set("userIDOriginal", userName)
	c.Set("userName", userName)

	if len(groups) > 0 {
		c.Set("userGroups", groups)
	}

	c.Set("authIdentity", userName)
}
