package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	consoleapi "github.com/openshift/console-operator/pkg/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GatherFailureLogs will log failed cluster components data out into RP logs for
// easier debugging
func (i *Installer) GatherFailureLogs(ctx context.Context) error {
	err := i.initializeKubernetesClients(ctx)
	if err != nil {
		return err
	}

	var statusCode int
	err = i.kubernetescli.Discovery().RESTClient().
		Get().
		AbsPath("/healthz").
		Do().
		StatusCode(&statusCode).
		Error()

	if statusCode != http.StatusOK {
		return fmt.Errorf("cluster never reached healthy API server state")
	}

	err = i.logClusterOperatorStatus(ctx)
	if err != nil {
		return err
	}
	return i.logConsoleOperatorStatus(ctx)
}

func (i *Installer) logClusterOperatorStatus(ctx context.Context) error {
	cos, err := i.configcli.ConfigV1().ClusterOperators().List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	data, err := json.Marshal(cos)
	if err != nil {
		return err
	}
	i.log.Printf("cluster operators state: %s", string(data))
	return nil
}

func (i *Installer) logConsoleOperatorStatus(ctx context.Context) error {
	i.log.Print("gathering console-operator logs")
	operatorConfig, err := i.operatorcli.OperatorV1().Consoles().Get(consoleapi.ConfigResourceName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	data, err := json.Marshal(operatorConfig)
	if err != nil {
		return err
	}
	i.log.Printf("console operator state: %s", string(data))
	return nil
}
