package auth

// AuthMethod defines the authentication strategy used by the authenticator.
type AuthMethod string

const (
	// MethodKubeToken validates bearer tokens via the Kubernetes TokenReview API.
	MethodKubeToken AuthMethod = "kube-token"

	// MethodKubeconfig authenticates using a kubeconfig file.
	MethodKubeconfig AuthMethod = "kubeconfig"

	// MethodAnonymous allows unauthenticated requests (no token required).
	MethodAnonymous AuthMethod = "anonymous"
)

// UserInfo holds the identity of an authenticated user.
type UserInfo struct {
	// Username is the unique identifier of the authenticated user.
	Username string `json:"username"`

	// Groups lists the groups the user belongs to.
	Groups []string `json:"groups"`

	// UID is the unique identifier assigned by the authentication provider.
	UID string `json:"uid"`
}

// AuthConfig configures the authentication middleware behaviour.
type AuthConfig struct {
	// Enabled controls whether authentication is active.
	Enabled bool `json:"enabled"`

	// TokenHeader is the HTTP header used to extract bearer tokens.
	// Defaults to "Authorization" when left empty.
	TokenHeader string `json:"tokenHeader,omitempty"`

	// KubeconfigPath is the filesystem path to a kubeconfig file.
	// When empty the in-cluster config is used.
	KubeconfigPath string `json:"kubeconfigPath,omitempty"`

	// AllowAnonymous permits requests that carry no authentication token.
	// When false, missing tokens result in a 401 response.
	AllowAnonymous bool `json:"allowAnonymous,omitempty"`
}

// DefaultAuthConfig returns an AuthConfig with sensible defaults.
func DefaultAuthConfig() AuthConfig {
	return AuthConfig{
		Enabled:     true,
		TokenHeader: "Authorization",
	}
}
