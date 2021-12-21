/*
Copyright 2021 The Ceph-CSI Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"google.golang.org/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	endpoint    = "unix:///tmp/csi-addons.sock"
	stagingPath = "/var/lib/kubelet/plugins/kubernetes.io/csi/pv/"
)

type command struct {
	endpoint         string
	stagingPath      string
	operation        string
	persistentVolume string
}

var cmd = &command{}

func init() {
	flag.StringVar(&cmd.endpoint, "endpoint", endpoint, "CSI-Addons endpoint")
	flag.StringVar(&cmd.stagingPath, "stagingpath", stagingPath, "staging path")
	flag.StringVar(&cmd.operation, "operation", "", "csi-addons operation")
	flag.StringVar(&cmd.persistentVolume, "persistentvolume", "", "name of the PersistentVolume")

	flag.Usage = func() {
		flag.PrintDefaults()
		fmt.Fprintln(flag.CommandLine.Output())
		fmt.Fprintln(flag.CommandLine.Output(), "The following operations are supported:")
		for op, _ := range operations {
			fmt.Fprintln(flag.CommandLine.Output(), " - " + op)
		}
		os.Exit(0)
	}

	flag.Parse()
}

func main() {
	op, found := operations[cmd.operation]
	if !found {
		fmt.Printf("ERROR: operation %q not found\n", cmd.operation)
		os.Exit(1)
	}

	op.Connect()
	defer op.Close()

	err := op.Init(cmd)
	if err != nil {
		fmt.Printf("ERROR: failed to initialize %q: %v\n", cmd.operation, err)
		os.Exit(1)
	}

	err = op.Execute()
	if err != nil {
		fmt.Printf("ERROR: failed to execute %q: %v\n", cmd.operation, err)
		os.Exit(1)
	}
}

func getKubernetesClient() *kubernetes.Clientset {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	return clientset
}

// getSecret get the secret details by name.
func getSecret(c *kubernetes.Clientset, ns, name string) (map[string]string, error) {
	secrets := make(map[string]string)

	secret, err := c.CoreV1().Secrets(ns).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	for k, v := range secret.Data {
		secrets[k] = string(v)
	}

	return secrets, nil
}

var operations = make(map[string]operation)

type operation interface {
	Connect()
	Close()

	Init(c *command) error
	Execute() error
}

type grpcClient struct {
	Client *grpc.ClientConn
}

func (g *grpcClient) Connect() {
	conn, err := grpc.Dial(cmd.endpoint, grpc.WithInsecure())
	if err != nil {
		panic(fmt.Sprintf("failed to connect to %q: %v", cmd.endpoint, err))
	}

	g.Client = conn
}

func (g *grpcClient) Close() {
	g.Client.Close()
}

func registerOperation(name string, op operation) error {
	operations[name] = op

	return nil
}
