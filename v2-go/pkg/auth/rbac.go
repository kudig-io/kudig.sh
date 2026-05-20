package auth

import (
	"context"
	"fmt"
	"net/http"

	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// RBACChecker evaluates Kubernetes RBAC policies via the SubjectAccessReview API.
type RBACChecker struct {
	clientset *kubernetes.Clientset
}

// NewRBACChecker creates an RBACChecker backed by the given clientset.
func NewRBACChecker(clientset *kubernetes.Clientset) *RBACChecker {
	return &RBACChecker{clientset: clientset}
}

// CheckAccess determines whether user is authorised to perform verb on resource
// in the given namespace. The namespace may be empty for cluster-scoped resources;
// use CheckClusterAccess for explicitly cluster-scoped checks.
func (rc *RBACChecker) CheckAccess(ctx context.Context, user *UserInfo, verb, resource, namespace string) (bool, error) {
	if user == nil {
		return false, fmt.Errorf("rbac: user must not be nil")
	}

	sar := &authorizationv1.SubjectAccessReview{
		Spec: authorizationv1.SubjectAccessReviewSpec{
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Namespace: namespace,
				Verb:      verb,
				Resource:  resource,
			},
			User:   user.Username,
			UID:    user.UID,
			Groups: user.Groups,
		},
	}

	result, err := rc.clientset.AuthorizationV1().SubjectAccessReviews().Create(
		ctx, sar, metav1.CreateOptions{},
	)
	if err != nil {
		return false, fmt.Errorf("rbac: subject access review failed: %w", err)
	}

	return result.Status.Allowed, nil
}

// CheckClusterAccess determines whether user is authorised to perform verb on
// a cluster-scoped resource (no namespace).
func (rc *RBACChecker) CheckClusterAccess(ctx context.Context, user *UserInfo, verb, resource string) (bool, error) {
	if user == nil {
		return false, fmt.Errorf("rbac: user must not be nil")
	}

	sar := &authorizationv1.SubjectAccessReview{
		Spec: authorizationv1.SubjectAccessReviewSpec{
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Verb:     verb,
				Resource: resource,
			},
			User:   user.Username,
			UID:    user.UID,
			Groups: user.Groups,
		},
	}

	result, err := rc.clientset.AuthorizationV1().SubjectAccessReviews().Create(
		ctx, sar, metav1.CreateOptions{},
	)
	if err != nil {
		return false, fmt.Errorf("rbac: subject access review failed: %w", err)
	}

	return result.Status.Allowed, nil
}

// RequireAccess returns HTTP middleware that enforces RBAC access for the
// authenticated user. It reads the user from the request context (set by
// Authenticator.Middleware) and checks whether the user can perform verb on
// resource in the request's namespace path parameter.
//
// Returns 403 Forbidden when access is denied or the user is not authenticated.
func (rc *RBACChecker) RequireAccess(next http.Handler, verb, resource string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := UserFromContext(r.Context())
		if user == nil {
			http.Error(w, `{"error":"authentication required"}`, http.StatusForbidden)
			return
		}

		// Attempt to extract namespace from the URL path. Gorilla mux stores
		// this under the key "namespace"; fall back to the query string.
		namespace := r.PathValue("namespace")
		if namespace == "" {
			namespace = r.URL.Query().Get("namespace")
		}

		allowed, err := rc.CheckAccess(r.Context(), user, verb, resource, namespace)
		if err != nil {
			http.Error(w, `{"error":"authorization check failed"}`, http.StatusInternalServerError)
			return
		}

		if !allowed {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequireClusterAccess returns HTTP middleware that enforces cluster-scoped RBAC
// access for the authenticated user.
func (rc *RBACChecker) RequireClusterAccess(next http.Handler, verb, resource string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := UserFromContext(r.Context())
		if user == nil {
			http.Error(w, `{"error":"authentication required"}`, http.StatusForbidden)
			return
		}

		allowed, err := rc.CheckClusterAccess(r.Context(), user, verb, resource)
		if err != nil {
			http.Error(w, `{"error":"authorization check failed"}`, http.StatusInternalServerError)
			return
		}

		if !allowed {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
