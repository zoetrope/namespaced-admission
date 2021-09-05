package main

import (
	"context"
	"net/http"
	"os"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
}

func main() {
	opts := zap.Options{
		Development: true,
	}
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     ":8080",
		Port:                   9443,
		HealthProbeBindAddress: ":8081",
		LeaderElection:         false,
		CertDir:                "/certs",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	wh := mgr.GetWebhookServer()
	wh.Register("/validate", &webhook.Admission{
		Handler: &dummyValidator{},
	})
	wh.Register("/mutate", &webhook.Admission{
		Handler: &dummyMutator{},
	})

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting dummy-webhook")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

type dummyValidator struct {
}

func (m *dummyValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	logger := ctrl.Log.WithName("dummy-validator")
	dec, _ := admission.NewDecoder(scheme)

	logger.Info("validate", "namespace", req.Namespace, "name", req.Namespace)
	obj := unstructured.Unstructured{}
	err := dec.Decode(req, &obj)
	if err != nil {
		logger.Error(err, "failed to decode")
		return admission.Errored(http.StatusInternalServerError, err)
	}
	logger.Info("validate", "namespace", req.Namespace, "name", req.Namespace, "kind", obj.GetKind())
	return admission.Allowed("ok")
}

type dummyMutator struct {
}

func (m *dummyMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	logger := ctrl.Log.WithName("dummy-mutator")
	dec, _ := admission.NewDecoder(scheme)

	logger.Info("mutate", "namespace", req.Namespace, "name", req.Namespace)
	obj := unstructured.Unstructured{}
	err := dec.Decode(req, &obj)
	if err != nil {
		logger.Error(err, "failed to decode")
		return admission.Errored(http.StatusInternalServerError, err)
	}
	logger.Info("mutate", "namespace", req.Namespace, "name", req.Namespace, "kind", obj.GetKind())

	return admission.Allowed("ok")
}
