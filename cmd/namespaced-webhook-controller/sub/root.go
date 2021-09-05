package sub

import (
	"flag"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	namespaced_webhook "github.com/zoetrope/namespaced-webhook"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var options struct {
	targetLabelKey       string
	metricsAddr          string
	probeAddr            string
	enableLeaderElection bool
	leaderElectionID     string
	webhookAddr          string
	certDir              string
	zapOpts              zap.Options
}

var rootCmd = &cobra.Command{
	Use:     "namespaced-webhook-controller",
	Version: namespaced_webhook.Version,
	Short:   "namespaced-webhook controller",
	Long:    `namespaced-webhook controller`,

	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return subMain()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	fs := rootCmd.Flags()
	fs.StringVar(&options.targetLabelKey, "target-label-key", corev1.LabelMetadataName, "The label key of namespaces to select webhook target")
	fs.StringVar(&options.metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to")
	fs.StringVar(&options.probeAddr, "health-probe-bind-address", ":8081", "Listen address for health probes")
	fs.BoolVar(&options.enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	fs.StringVar(&options.leaderElectionID, "leader-election-id", "namespaced-webhook", "ID for leader election by controller-runtime")
	fs.StringVar(&options.webhookAddr, "webhook-bind-address", ":9443", "Listen address for the webhook endpoint")
	fs.StringVar(&options.certDir, "cert-dir", "", "webhook certificate directory")

	goflags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(goflags)
	options.zapOpts.BindFlags(goflags)

	fs.AddGoFlagSet(goflags)
}
