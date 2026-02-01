package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	routingv1alpha1 "github.com/seu-user/header-route-controller/api/v1alpha1"
	"github.com/seu-user/header-route-controller/internal/envoy"
)

// HeaderRouteReconciler reconciles a HeaderRoute object
type HeaderRouteReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Configuration from environment
type ControllerConfig struct {
	// ConfigMapName is the name of the ConfigMap to update with Envoy config
	ConfigMapName string
	// ConfigMapNamespace is the namespace of the ConfigMap
	ConfigMapNamespace string
	// ConfigMapKey is the key in the ConfigMap that holds Envoy config
	ConfigMapKey string
	// WatchNamespace limits which namespace to watch (empty = all)
	WatchNamespace string
	// DefaultBackendService is the fallback service name
	DefaultBackendService string
	// DefaultBackendNamespace is the fallback service namespace
	DefaultBackendNamespace string
	// DefaultBackendPort is the fallback service port
	DefaultBackendPort int32
}

var config ControllerConfig

func init() {
	config = ControllerConfig{
		ConfigMapName:           getEnv("CONFIGMAP_NAME", "envoy-config"),
		ConfigMapNamespace:      getEnv("CONFIGMAP_NAMESPACE", "poc"),
		ConfigMapKey:            getEnv("CONFIGMAP_KEY", "envoy.json"),
		WatchNamespace:          getEnv("WATCH_NAMESPACE", ""),
		DefaultBackendService:   getEnv("DEFAULT_BACKEND_SERVICE", ""),
		DefaultBackendNamespace: getEnv("DEFAULT_BACKEND_NAMESPACE", "poc"),
		DefaultBackendPort:      getEnvInt32("DEFAULT_BACKEND_PORT", 80),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt32(key string, defaultValue int32) int32 {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.ParseInt(value, 10, 32); err == nil {
			return int32(i)
		}
	}
	return defaultValue
}

// +kubebuilder:rbac:groups=routing.example.com,resources=headerroutes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=routing.example.com,resources=headerroutes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch

// Reconcile handles HeaderRoute changes
func (r *HeaderRouteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling HeaderRoute", "name", req.Name, "namespace", req.Namespace)

	// List all HeaderRoutes
	var headerRoutes routingv1alpha1.HeaderRouteList
	listOpts := []client.ListOption{}
	
	if config.WatchNamespace != "" {
		listOpts = append(listOpts, client.InNamespace(config.WatchNamespace))
	}

	if err := r.List(ctx, &headerRoutes, listOpts...); err != nil {
		logger.Error(err, "Failed to list HeaderRoutes")
		return ctrl.Result{}, err
	}

	logger.Info("Found HeaderRoutes", "count", len(headerRoutes.Items))

	// Prepare default backend if configured
	var defaultBackend *envoy.DefaultBackend
	if config.DefaultBackendService != "" {
		defaultBackend = &envoy.DefaultBackend{
			ServiceName: config.DefaultBackendService,
			Namespace:   config.DefaultBackendNamespace,
			Port:        config.DefaultBackendPort,
		}
		logger.Info("Using default backend", 
			"service", defaultBackend.ServiceName,
			"namespace", defaultBackend.Namespace,
			"port", defaultBackend.Port)
	}

	// Generate Envoy configuration
	envoyConfig := envoy.GenerateEnvoyConfig(headerRoutes.Items, defaultBackend)

	// Convert to JSON
	configJSON, err := json.MarshalIndent(envoyConfig, "", "  ")
	if err != nil {
		logger.Error(err, "Failed to marshal Envoy config")
		return ctrl.Result{}, err
	}

	// Update or create ConfigMap
	if err := r.updateConfigMap(ctx, configJSON); err != nil {
		logger.Error(err, "Failed to update ConfigMap")
		return ctrl.Result{}, err
	}

	// Update status for the specific HeaderRoute that triggered reconciliation
	var headerRoute routingv1alpha1.HeaderRoute
	if err := r.Get(ctx, req.NamespacedName, &headerRoute); err != nil {
		if errors.IsNotFound(err) {
			// HeaderRoute was deleted, that's okay
			logger.Info("HeaderRoute not found, likely deleted")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Update status
	headerRoute.Status.Ready = true
	headerRoute.Status.Message = "Route configured successfully"
	headerRoute.Status.LastUpdated = metav1.Time{Time: time.Now()}

	if err := r.Status().Update(ctx, &headerRoute); err != nil {
		logger.Error(err, "Failed to update HeaderRoute status")
		// Don't return error, config is already updated
	}

	logger.Info("Successfully reconciled HeaderRoute", "name", req.Name)
	return ctrl.Result{}, nil
}

func (r *HeaderRouteReconciler) updateConfigMap(ctx context.Context, configData []byte) error {
	logger := log.FromContext(ctx)
	
	configMapKey := types.NamespacedName{
		Name:      config.ConfigMapName,
		Namespace: config.ConfigMapNamespace,
	}

	var existingCM corev1.ConfigMap
	err := r.Get(ctx, configMapKey, &existingCM)

	if errors.IsNotFound(err) {
		// Create new ConfigMap
		newCM := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      config.ConfigMapName,
				Namespace: config.ConfigMapNamespace,
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "header-route-controller",
					"app.kubernetes.io/name":       "envoy-config",
				},
			},
			Data: map[string]string{
				config.ConfigMapKey: string(configData),
			},
		}

		if err := r.Create(ctx, newCM); err != nil {
			return fmt.Errorf("failed to create ConfigMap: %w", err)
		}
		logger.Info("Created ConfigMap", "name", config.ConfigMapName, "namespace", config.ConfigMapNamespace)
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to get ConfigMap: %w", err)
	}

	// Update existing ConfigMap
	if existingCM.Data == nil {
		existingCM.Data = make(map[string]string)
	}
	existingCM.Data[config.ConfigMapKey] = string(configData)

	if err := r.Update(ctx, &existingCM); err != nil {
		return fmt.Errorf("failed to update ConfigMap: %w", err)
	}
	
	logger.Info("Updated ConfigMap", "name", config.ConfigMapName, "namespace", config.ConfigMapNamespace)
	return nil
}

// SetupWithManager sets up the controller with the Manager
func (r *HeaderRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&routingv1alpha1.HeaderRoute{}).
		Complete(r)
}
