// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package multi

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/ServiceWeaver/weaver/internal/files"
	"github.com/ServiceWeaver/weaver/internal/status"
	"github.com/ServiceWeaver/weaver/runtime"
	"github.com/ServiceWeaver/weaver/runtime/codegen"
	"github.com/ServiceWeaver/weaver/runtime/colors"
	"github.com/ServiceWeaver/weaver/runtime/logging"
	"github.com/ServiceWeaver/weaver/runtime/retry"
	"github.com/ServiceWeaver/weaver/runtime/tool"
	"github.com/google/uuid"
)

var deployCmd = tool.Command{
	Name:        "deploy",
	Description: "Deploy a Service Weaver app",
	Help:        "Usage:\n  weaver multi deploy <configfile>",
	Flags:       flag.NewFlagSet("deploy", flag.ContinueOnError),
	Fn:          deploy,
}

// deploy deploys an application on the local machine using a multiprocess
// deployer. Note that each component is deployed as a separate OS process.
func deploy(ctx context.Context, args []string) error {
	// Validate command line arguments.
	if len(args) == 0 {
		return fmt.Errorf("no config file provided")
	}
	if len(args) > 1 {
		return fmt.Errorf("too many arguments")
	}

	// Load the config file.
	configFile := args[0]
	bytes, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("load config file %q: %w\n", configFile, err)
	}
	config, err := runtime.ParseConfig(configFile, string(bytes), codegen.ComponentConfigValidator)
	if err != nil {
		return fmt.Errorf("load config file %q: %w\n", configFile, err)
	}

	// Sanity check the config.
	if _, err := os.Stat(config.Binary); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("binary %q doesn't exist", config.Binary)
	}

	// Create the deployer.
	deploymentId := uuid.New().String()
	d, err := newDeployer(ctx, deploymentId, config)
	if err != nil {
		return fmt.Errorf("create deployer: %w", err)
	}

	// Run a status server.
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	mux := http.NewServeMux()
	status.RegisterServer(mux, d, d.logger)
	go func() {
		if err := serveHTTP(ctx, lis, mux); err != nil {
			fmt.Fprintf(os.Stderr, "status server: %v\n", err)
		}
	}()

	// Deploy main.
	if err := d.startMain(); err != nil {
		return fmt.Errorf("start main process: %w", err)
	}

	// Wait for the status server to become active.
	client := status.NewClient(lis.Addr().String())
	for r := retry.Begin(); r.Continue(ctx); {
		_, err := client.Status(ctx)
		if err == nil {
			break
		}
		fmt.Fprintf(os.Stderr, "status server %q unavailable: %#v\n", lis.Addr(), err)
	}

	// Register the deployment.
	registry, err := defaultRegistry(ctx)
	if err != nil {
		return fmt.Errorf("create registry: %w", err)
	}
	reg := status.Registration{
		DeploymentId: deploymentId,
		App:          config.Name,
		Addr:         lis.Addr().String(),
	}
	fmt.Fprint(os.Stderr, reg.Rolodex())
	if err := registry.Register(ctx, reg); err != nil {
		return fmt.Errorf("register deployment: %w", err)
	}

	userDone := make(chan os.Signal, 1)
	deployerDone := make(chan error, 1)
	go func() {
		err := d.wait()
		deployerDone <- err
	}()
	signal.Notify(userDone, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		// Wait for the user to kill the app or the app to return an error.
		select {
		case <-userDone:
			fmt.Fprintf(os.Stderr, "Application %s terminated by the user\n", config.Name)
		case err := <-deployerDone:
			fmt.Fprintf(os.Stderr, "Application %s error: %v\n", config.Name, err)
		}
		if err := registry.Unregister(ctx, deploymentId); err != nil {
			fmt.Fprintf(os.Stderr, "unregister deployment: %v\n", err)
		}
		os.Exit(1)
	}()

	// Follow the logs.
	source := logging.FileSource(logdir)
	query := fmt.Sprintf(`full_version == %q && !("serviceweaver/system" in attrs)`, deploymentId)
	r, err := source.Query(ctx, query, true)
	if err != nil {
		return err
	}
	pp := logging.NewPrettyPrinter(colors.Enabled())
	for {
		entry, err := r.Read(ctx)
		if errors.Is(err, io.EOF) {
			return nil
		} else if err != nil {
			return err
		}
		fmt.Println(pp.Format(entry))
	}
}

// defaultRegistryDir() returns $XDG_DATA_HOME/serviceweaver/multi_registry, or
// ~/.local/share/serviceweaver/multi_registry if XDG_DATA_HOME is not set.
func defaultRegistryDir() (string, error) {
	dir, err := files.DefaultDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "multi_registry"), nil
}

// defaultRegistry returns a registry in defaultRegistryDir().
func defaultRegistry(ctx context.Context) (*status.Registry, error) {
	dir, err := defaultRegistryDir()
	if err != nil {
		return nil, err
	}
	return status.NewRegistry(ctx, dir)
}
