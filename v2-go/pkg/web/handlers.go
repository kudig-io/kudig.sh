package web

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ScaleDeploymentRequest is the POST body for scaling a deployment.
type ScaleDeploymentRequest struct {
	Replicas int32 `json:"replicas"`
}

// ClusterInfo is a simple cluster representation.
type ClusterInfo struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// --- Cluster Handlers ---

func (s *Server) handleListClusters(w http.ResponseWriter, r *http.Request) {
	ns, err := NewResourcesFromClientset(s.clientset).ListNamespaces()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	clusters := make([]ClusterInfo, 0, len(ns))
	for _, n := range ns {
		status := "Active"
		if n.Status.Phase != "" {
			status = string(n.Status.Phase)
		}
		clusters = append(clusters, ClusterInfo{Name: n.Name, Status: status})
	}
	respondJSON(w, http.StatusOK, clusters)
}

func (s *Server) handleGetCluster(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["cluster"]
	ns, err := s.clientset.CoreV1().Namespaces().Get(
		r.Context(), name, metav1.GetOptions{})
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, ns)
}

func (s *Server) handleGetClusterStatus(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["cluster"]
	ns, err := s.clientset.CoreV1().Namespaces().Get(
		r.Context(), name, metav1.GetOptions{})
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"name": ns.Name, "status": string(ns.Status.Phase)})
}

// --- Pod Handlers ---

func (s *Server) handleListPods(w http.ResponseWriter, r *http.Request) {
	ns := mux.Vars(r)["namespace"]
	pods, err := NewResourcesFromClientset(s.clientset).ListPods(ns)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, pods)
}

func (s *Server) handleListAllPods(w http.ResponseWriter, r *http.Request) {
	pods, err := NewResourcesFromClientset(s.clientset).ListPods("")
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, pods)
}

func (s *Server) handleGetPod(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pod, err := NewResourcesFromClientset(s.clientset).GetPod(vars["namespace"], vars["pod"])
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, pod)
}

func (s *Server) handleGetPodLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tailLines := int64(100)
	if tl := r.URL.Query().Get("tailLines"); tl != "" {
		if v, err := strconv.ParseInt(tl, 10, 64); err == nil && v > 0 {
			tailLines = v
		}
	}
	logs, err := NewResourcesFromClientset(s.clientset).GetPodLogs(
		vars["namespace"], vars["pod"], tailLines)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"logs": logs})
}

func (s *Server) handleDeletePod(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if err := NewResourcesFromClientset(s.clientset).DeletePod(vars["namespace"], vars["pod"]); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// --- Node Handlers ---

func (s *Server) handleListNodes(w http.ResponseWriter, r *http.Request) {
	nodes, err := NewResourcesFromClientset(s.clientset).ListNodes()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, nodes)
}

func (s *Server) handleGetNode(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["node"]
	node, err := NewResourcesFromClientset(s.clientset).GetNode(name)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, node)
}

func (s *Server) handleGetNodeMetrics(w http.ResponseWriter, r *http.Request) {
	metrics, err := NewResourcesFromClientset(s.clientset).GetNodeMetrics()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, metrics)
}

// --- Deployment Handlers ---

func (s *Server) handleListDeployments(w http.ResponseWriter, r *http.Request) {
	ns := mux.Vars(r)["namespace"]
	deps, err := NewResourcesFromClientset(s.clientset).ListDeployments(ns)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, deps)
}

func (s *Server) handleListAllDeployments(w http.ResponseWriter, r *http.Request) {
	deps, err := NewResourcesFromClientset(s.clientset).ListDeployments("")
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, deps)
}

func (s *Server) handleGetDeployment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dep, err := NewResourcesFromClientset(s.clientset).GetDeployment(vars["namespace"], vars["deployment"])
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, dep)
}

func (s *Server) handleScaleDeployment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var req ScaleDeploymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := NewResourcesFromClientset(s.clientset).ScaleDeployment(
		vars["namespace"], vars["deployment"], req.Replicas); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "scaled"})
}

func (s *Server) handleRestartDeployment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if err := NewResourcesFromClientset(s.clientset).RestartDeployment(
		vars["namespace"], vars["deployment"]); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "restarting"})
}

func (s *Server) handleGetDeploymentPods(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pods, err := NewResourcesFromClientset(s.clientset).GetDeploymentPods(
		vars["namespace"], vars["deployment"])
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, pods)
}

func (s *Server) handleGetDeploymentStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	status, err := NewResourcesFromClientset(s.clientset).GetDeploymentStatus(
		vars["namespace"], vars["deployment"])
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, status)
}

// --- Service Handlers ---

func (s *Server) handleListServices(w http.ResponseWriter, r *http.Request) {
	ns := mux.Vars(r)["namespace"]
	svcs, err := NewResourcesFromClientset(s.clientset).ListServices(ns)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, svcs)
}

func (s *Server) handleListAllServices(w http.ResponseWriter, r *http.Request) {
	svcs, err := NewResourcesFromClientset(s.clientset).ListServices("")
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, svcs)
}

func (s *Server) handleGetService(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	svc, err := NewResourcesFromClientset(s.clientset).GetService(vars["namespace"], vars["service"])
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, svc)
}

func (s *Server) handleGetServiceEndpoints(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ep, err := NewResourcesFromClientset(s.clientset).GetServiceEndpoints(vars["namespace"], vars["service"])
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, ep)
}

// --- Event Handlers ---

func (s *Server) handleGetEvents(w http.ResponseWriter, r *http.Request) {
	ns := ""
	if v, ok := mux.Vars(r)["namespace"]; ok {
		ns = v
	}
	events, err := NewResourcesFromClientset(s.clientset).ListEvents(ns)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, events)
}

// --- Monitoring Handlers ---

func (s *Server) handleGetMonitorStatus(w http.ResponseWriter, r *http.Request) {
	nodes, err := NewResourcesFromClientset(s.clientset).ListNodes()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	total := len(nodes)
	ready := 0
	for _, n := range nodes {
		for _, c := range n.Status.Conditions {
			if c.Type == "Ready" && c.Status == "True" {
				ready++
			}
		}
	}
	respondJSON(w, http.StatusOK, map[string]int{
		"total_nodes":     total,
		"ready_nodes":     ready,
		"not_ready_nodes": total - ready,
	})
}

func (s *Server) handleGetMonitorAlerts(w http.ResponseWriter, r *http.Request) {
	events, err := NewResourcesFromClientset(s.clientset).ListEvents("")
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	alerts := make([]interface{}, 0)
	for _, e := range events {
		if e.Type == "Warning" {
			alerts = append(alerts, e)
		}
	}
	respondJSON(w, http.StatusOK, alerts)
}

func (s *Server) handleGetMetricsHistory(w http.ResponseWriter, r *http.Request) {
	metrics, err := NewResourcesFromClientset(s.clientset).GetNodeMetrics()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, metrics)
}
