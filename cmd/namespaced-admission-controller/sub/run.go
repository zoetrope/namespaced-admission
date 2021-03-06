package sub

import (
	"fmt"
	"net"
	"strconv"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	webhookv1 "github.com/zoetrope/namespaced-admission/api/v1"
	"github.com/zoetrope/namespaced-admission/controllers"
	"github.com/zoetrope/namespaced-admission/hooks"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func subMain() error {
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&options.zapOpts)))
	setupLogger := ctrl.Log.WithName("setup")

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return fmt.Errorf("unable to add client-go objects: %w", err)
	}
	if err := webhookv1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("unable to add client-go objects: %w", err)
	}

	host, p, err := net.SplitHostPort(options.webhookAddr)
	if err != nil {
		return fmt.Errorf("invalid webhook address: %s, %v", options.webhookAddr, err)
	}
	numPort, err := strconv.Atoi(p)
	if err != nil {
		return fmt.Errorf("invalid webhook address: %s, %v", options.webhookAddr, err)
	}

	config := ctrl.GetConfigOrDie()
	mgr, err := ctrl.NewManager(config, ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     options.metricsAddr,
		Host:                   host,
		Port:                   numPort,
		HealthProbeBindAddress: options.probeAddr,
		LeaderElection:         options.enableLeaderElection,
		LeaderElectionID:       options.leaderElectionID,
		CertDir:                options.certDir,
	})
	if err != nil {
		return fmt.Errorf("unable to start manager: %w", err)
	}

	dec, err := admission.NewDecoder(scheme)
	if err != nil {
		return fmt.Errorf("unable to create admission decoder: %w", err)
	}

	if err = (&controllers.NamespacedMutatingWebhookConfigurationReconciler{
		Client:         mgr.GetClient(),
		Scheme:         mgr.GetScheme(),
		TargetLabelKey: options.targetLabelKey,
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create NamespacedMutatingWebhookConfiguration controller: %w", err)
	}
	hooks.SetupNamespacedMutatingWebhookConfigurationWebhook(mgr, config, dec)

	if err = (&controllers.NamespacedValidatingWebhookConfigurationReconciler{
		Client:         mgr.GetClient(),
		Scheme:         mgr.GetScheme(),
		TargetLabelKey: options.targetLabelKey,
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create NamespacedValidatingWebhookConfiguration controller: %w", err)
	}
	hooks.SetupNamespacedValidatingWebhookConfigurationWebhook(mgr, config, dec)

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return fmt.Errorf("unable to set up health check: %w", err)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		return fmt.Errorf("unable to set up ready check: %w", err)
	}

	setupLogger.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		return fmt.Errorf("problem running manager: %s", err)
	}
	return nil
}
