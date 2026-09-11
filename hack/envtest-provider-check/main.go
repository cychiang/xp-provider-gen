// Command envtest-provider-check starts a throwaway envtest Kubernetes API
// server, points a given upjet-generated provider binary at it, and fails if
// the provider panics or never gets its controllers running. It is what lets
// scripts/e2e-upjet.sh prove a generated provider RUNS, not just builds.
//
// It is its own Go module so that this check's dependencies (controller-runtime,
// client-go) never touch the generator's own go.mod/go.sum.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

// envtestContextName is the (arbitrary) kubeconfig cluster/user/context name
// written for the throwaway envtest API server.
const envtestContextName = "envtest"

func main() {
	os.Exit(mainErr())
}

func mainErr() int {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "envtest-provider-check:", err)
		return 1
	}
	return 0
}

func run(ctx context.Context) error {
	provider := flag.String("provider", "", "path to the provider binary to exercise")
	crdDir := flag.String("crd-dir", "", "directory of CRDs to load into the test API server")
	settle := flag.Duration("settle", 10*time.Second, "how long the provider must stay up before it is considered started")
	flag.Parse()

	if *provider == "" || *crdDir == "" {
		return fmt.Errorf("-provider and -crd-dir are required")
	}

	env := &envtest.Environment{
		CRDDirectoryPaths:     []string{*crdDir},
		ErrorIfCRDPathMissing: true,
	}
	cfg, err := env.Start()
	if err != nil {
		return fmt.Errorf("starting envtest API server: %w", err)
	}
	defer func() {
		if stopErr := env.Stop(); stopErr != nil {
			fmt.Fprintln(os.Stderr, "envtest-provider-check: stopping envtest:", stopErr)
		}
	}()

	kubeconfig, err := os.CreateTemp("", "envtest-provider-check-*.kubeconfig")
	if err != nil {
		return fmt.Errorf("creating kubeconfig file: %w", err)
	}
	defer os.Remove(kubeconfig.Name())
	if err := writeKubeconfig(cfg, kubeconfig.Name()); err != nil {
		return err
	}

	return runProvider(ctx, *provider, kubeconfig.Name(), *settle)
}

func writeKubeconfig(cfg *rest.Config, path string) error {
	kc := clientcmdapi.NewConfig()
	kc.Clusters[envtestContextName] = &clientcmdapi.Cluster{
		Server:                   cfg.Host,
		CertificateAuthorityData: cfg.CAData,
	}
	kc.AuthInfos[envtestContextName] = &clientcmdapi.AuthInfo{
		ClientCertificateData: cfg.CertData,
		ClientKeyData:         cfg.KeyData,
	}
	kc.Contexts[envtestContextName] = &clientcmdapi.Context{Cluster: envtestContextName, AuthInfo: envtestContextName}
	kc.CurrentContext = envtestContextName
	return clientcmd.WriteToFile(*kc, path)
}

// runProvider execs the provider binary against the envtest cluster and lets
// it run for settle. Reaching settle (or the caller's ctx being canceled) is
// the expected way for the process to end: it is asked to stop with SIGTERM
// and force-killed if it ignores that. Exiting on its own before then means
// it crashed or never really started, which is a failure. Either way, a
// panic signature in its output, or never reporting a started controller, is
// also a failure.
func runProvider(ctx context.Context, provider, kubeconfig string, settle time.Duration) error {
	settleCtx, cancel := context.WithTimeout(ctx, settle)
	defer cancel()

	cmd := exec.CommandContext(settleCtx, provider,
		"--terraform-version=1.5.7",
		"--terraform-provider-source=hashicorp/kubernetes",
		"--terraform-provider-version=2.38.0",
		"--no-leader-election",
		"--debug",
	)
	cmd.Cancel = func() error {
		return cmd.Process.Signal(syscall.SIGTERM)
	}
	cmd.WaitDelay = 5 * time.Second
	cmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfig)

	var out bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &out)
	cmd.Stderr = cmd.Stdout

	waitErr := cmd.Run()
	if settleCtx.Err() == nil {
		// The process ended on its own, before we ever asked it to stop.
		return fmt.Errorf("provider exited before startup completed: %w; output:\n%s", waitErr, out.String())
	}

	output := out.String()
	if strings.Contains(output, "panic:") || strings.Contains(output, "runtime error") {
		return fmt.Errorf("provider panicked during startup; output:\n%s", output)
	}
	if !strings.Contains(output, "Starting Controller") {
		return fmt.Errorf("provider never reported starting a controller; output:\n%s", output)
	}

	fmt.Println("envtest-provider-check: provider started controllers and shut down cleanly")
	return nil
}
