package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// contextKey is an unexported type for context keys defined in this package.
type contextKey struct {
	name string
}

// userContextKey is the key under which UserInfo is stored in a request context.
var userContextKey = contextKey{name: "user"}

// Authenticator validates Kubernetes bearer tokens and injects the resulting
// UserInfo into the HTTP request context.
type Authenticator struct {
	config    AuthConfig
	clientset *kubernetes.Clientset
}

// NewAuthenticator builds an Authenticator from the provided config.
// It resolves the Kubernetes clientset from KubeconfigPath or in-cluster config.
func NewAuthenticator(config AuthConfig) (*Authenticator, error) {
	tokenHeader := config.TokenHeader
	if tokenHeader == "" {
		tokenHeader = "Authorization"
	}
	config.TokenHeader = tokenHeader

	var restConfig *rest.Config
	var err error

	if config.KubeconfigPath != "" {
		restConfig, err = clientcmd.BuildConfigFromFlags("", config.KubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("auth: failed to build config from kubeconfig %q: %w", config.KubeconfigPath, err)
		}
	} else {
		restConfig, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("auth: failed to build in-cluster config: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("auth: failed to create kubernetes clientset: %w", err)
	}

	return &Authenticator{
		config:    config,
		clientset: clientset,
	}, nil
}

// NewAuthenticatorFromClientset creates an Authenticator using an already-initialised clientset.
func NewAuthenticatorFromClientset(clientset *kubernetes.Clientset, config AuthConfig) *Authenticator {
	tokenHeader := config.TokenHeader
	if tokenHeader == "" {
		tokenHeader = "Authorization"
	}
	config.TokenHeader = tokenHeader

	return &Authenticator{
		config:    config,
		clientset: clientset,
	}
}

// Middleware returns an http.Handler that authenticates every request.
//
//   - Bearer tokens are extracted from the header named by config.TokenHeader.
//   - Valid tokens produce a UserInfo stored in the request context (retrieve
//     with UserFromContext).
//   - If AllowAnonymous is true, requests without a token proceed with a nil user.
//   - Invalid or expired tokens always receive a 401 response.
func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.config.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		token := extractBearerToken(r.Header.Get(a.config.TokenHeader))

		if token == "" {
			if a.config.AllowAnonymous {
				next.ServeHTTP(w, r)
				return
			}
			http.Error(w, `{"error":"missing authentication token"}`, http.StatusUnauthorized)
			return
		}

		userInfo, err := a.validateToken(r.Context(), token)
		if err != nil {
			http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		ctx := WithUser(r.Context(), userInfo)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// validateToken submits a TokenReview to the Kubernetes API and returns the
// authenticated user information.
func (a *Authenticator) validateToken(ctx context.Context, token string) (*UserInfo, error) {
	tokenReview := &authenticationv1.TokenReview{
		Spec: authenticationv1.TokenReviewSpec{
			Token: token,
		},
	}

	result, err := a.clientset.AuthenticationV1().TokenReviews().Create(
		ctx, tokenReview, metav1.CreateOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("auth: token review request failed: %w", err)
	}

	if !result.Status.Authenticated {
		if result.Status.Error != "" {
			return nil, fmt.Errorf("auth: token not authenticated: %s", result.Status.Error)
		}
		return nil, fmt.Errorf("auth: token not authenticated")
	}

	return &UserInfo{
		Username: result.Status.User.Username,
		UID:      result.Status.User.UID,
		Groups:   result.Status.User.Groups,
	}, nil
}

// extractBearerToken strips the "Bearer " prefix (case-insensitive) from the
// raw authorization header value and trims whitespace.
func extractBearerToken(header string) string {
	if len(header) > 7 && strings.EqualFold(header[:7], "Bearer ") {
		return strings.TrimSpace(header[7:])
	}
	return ""
}

// WithUser returns a copy of ctx carrying the supplied UserInfo.
func WithUser(ctx context.Context, user *UserInfo) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// UserFromContext retrieves the UserInfo stored by the authentication middleware.
// Returns nil when no user has been set (e.g. anonymous requests).
func UserFromContext(ctx context.Context) *UserInfo {
	user, _ := ctx.Value(userContextKey).(*UserInfo)
	return user
}
