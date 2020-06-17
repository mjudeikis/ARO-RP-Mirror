package etcd

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/davecgh/go-spew/spew"

	"github.com/coreos/etcd/clientv3"
	assetstls "github.com/openshift/installer/pkg/asset/tls"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/graph"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
)

var (
	dialTimeout    = 2 * time.Second
	requestTimeout = 10 * time.Second
)

type Interface interface {
	Get(ctx context.Context, oc *api.OpenShiftCluster) ([]byte, error)
}

type etcdactions struct {
	log *logrus.Entry
	env env.Interface

	kv clientv3.KV
}

func New(log *logrus.Entry, env env.Interface) Interface {
	return &etcdactions{
		log: log,
		env: env,
	}
}

func (ea *etcdactions) Get(ctx context.Context, oc *api.OpenShiftCluster) ([]byte, error) {
	err := ea.init(ctx, oc)
	if err != nil {
		return nil, err
	}

	ea.log.Info("get object")
	gr, err := ea.kv.Get(ctx, "/")
	if err != nil {
		return nil, err
	}
	fmt.Println("Value: ", string(gr.Kvs[0].Value), "Revision: ", gr.Header.Revision)
	os.Exit(1)

	return nil, nil
}

func (ea *etcdactions) init(ctx context.Context, oc *api.OpenShiftCluster) error {
	gCli, err := graph.New(ctx, ea.log, ea.env, oc)
	if err != nil {
		return err
	}

	g, err := gCli.Load(ctx)
	if err != nil {
		return err
	}

	certKey := g[reflect.TypeOf(&assetstls.EtcdSignerCertKey{})].(*assetstls.EtcdSignerCertKey)

	data, _ := pem.Decode(certKey.Key())

	key, err := x509.ParsePKCS1PrivateKey(data.Bytes)
	if err != nil {
		ea.log.Info("key")
		spew.Dump(err)
		return err
	}

	data, _ = pem.Decode(certKey.Cert())
	certs, err := x509.ParseCertificates(data.Bytes)
	if err != nil {
		spew.Dump(err)
		return err
	}

	_, _, err = utiltls.GenerateKeyAndCertificate("rp-etcd", key, certs[0], false, true)
	if err != nil {
		spew.Dump(err)
		return err
	}

	cli, err := clientv3.New(clientv3.Config{

		DialTimeout: dialTimeout,
		Endpoints:   []string{oc.Properties.NetworkProfile.PrivateEndpointIP + ":2379"},
	})
	if err != nil {
		return err
	}
	ea.kv = clientv3.NewKV(cli)

	return nil
}
