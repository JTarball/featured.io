package app

import (
	"flag"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/homedir"
)

// CMDFlags are the flags used by the cmd
type CMDFlags struct {
	LogLevel   string `yaml:"loglevel"`
	DevMode    bool   `yaml:"devmode"`
	KubeConfig string `yaml:"kubeconfig"`
	Namespace  string `yaml:"namespace"`
	// minResyncPeriod is the resync period in reflectors;
	//will be random between minResyncPeriod and 2*minResyncPeriod.
	MinResyncPeriod metav1.Duration `yaml:"minResyncPeriod"`

	//Development bool
	MetricsListenAddr string `yaml:"metricslistenaddr"`
	MetricsPath       string `yaml:"metricspath"`
}

// Init initializes and parse the flags
func (c *CMDFlags) Init() {
	flag.StringVar(&c.LogLevel, "loglevel", "INFO", "The log level")
	flag.BoolVar(&c.DevMode, "dev", false, "A development flag that will allow to run the operator outside a kubernetes cluster")

	kubehome := filepath.Join(homedir.HomeDir(), ".kube", "config")
	flag.StringVar(&c.KubeConfig, "kubeconfig", kubehome, "The kubernetes configuration path, only used when development mode enabled")

	flag.StringVar(&c.Namespace, "namespace", metav1.NamespaceAll, "The namespace for which the featured.io operator manages feature flags. Defaults to all.")

	// TODO: Add a minimum resync period & a EnsureDefaults() like https://github.com/oracle/mysql-operator/blob/9aebcc37a080283bdc199d7150d91f2e9bfa4e2c/pkg/controllers/cluster/controller.go#L322
	//fs.DurationVar(&c.MinResyncPeriod.Duration, "min-resync-period", c.MinResyncPeriod.Duration, "The resync period in reflectors will be random between MinResyncPeriod and 2*MinResyncPeriod.")

	flag.StringVar(&c.MetricsListenAddr, "metrics-address", ":9710", "Address to listen on for metrics.")
	flag.StringVar(&c.MetricsPath, "metrics-path", "/metrics", "Path to serve the metrics.")

	// Parse flags
	flag.Parse()
}
