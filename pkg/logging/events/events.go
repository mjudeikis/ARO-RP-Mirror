package events

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/env"
)

const (
	kubeNamespace      = "openshift-azure-logging"
	kubeServiceAccount = "system:serviceaccount:" + kubeNamespace + ":eventrouter"

	eventRouterImageFormat = "%s.azurecr.io/ose-logging-eventrouter:latest"
)

type EventLogging interface {
	CreateOrUpdate() error
}

type eventLogging struct {
	log *logrus.Entry
	env env.Interface

	cli kubernetes.Interface
}

func New(log *logrus.Entry, e env.Interface, cli kubernetes.Interface) EventLogging {
	return &eventLogging{
		log: log,
		env: e,

		cli: cli,
	}
}

func (g *eventLogging) eventRouterImage() string {
	return fmt.Sprintf(eventRouterImageFormat, g.env.ACRName())
}

func (g *eventLogging) ensureNamespace(ns string) error {
	_, err := g.cli.CoreV1().Namespaces().Create(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ns,
		},
	})
	if !errors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

func (g *eventLogging) ensureServiceAccount() error {
	_, err := g.cli.CoreV1().ServiceAccounts(kubeNamespace).Create(&v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: "eventrouter",
		},
	})
	if !errors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

func (g *eventLogging) ensureClusterRole() error {
	name := "event-reader"
	r := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"events"},
				Verbs:     []string{"get", "watch", "list"},
			},
		},
	}
	_, err := g.cli.RbacV1().ClusterRoles().Create(r)
	if !errors.IsAlreadyExists(err) {
		return err
	}
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_r, err := g.cli.RbacV1().ClusterRoles().Get(name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		r.ResourceVersion = _r.ResourceVersion
		_, err = g.cli.RbacV1().ClusterRoles().Update(r)
		return err
	})
}

func (g *eventLogging) ensureClusterRoleBinding() error {
	name := "event-reader-binding"
	rb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      kubeServiceAccount[strings.LastIndex(kubeServiceAccount, ":"):],
				Namespace: kubeNamespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind: "ClusterRole",
			Name: "event-reader",
		},
	}
	_, err := g.cli.RbacV1().ClusterRoleBindings().Create(rb)
	if !errors.IsAlreadyExists(err) {
		return err
	}
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_rb, err := g.cli.RbacV1().ClusterRoleBindings().Get(name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		rb.ResourceVersion = _rb.ResourceVersion
		_, err = g.cli.RbacV1().ClusterRoleBindings().Update(rb)
		return err
	})
}

func (g *eventLogging) ensureConfigMap() error {
	name := "eventrouter"
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: kubeNamespace,
		},
		Data: map[string]string{
			"config.json": "{\"sink\": \"stdout\"}",
		},
	}
	_, err := g.cli.CoreV1().ConfigMaps(kubeNamespace).Create(cm)
	if !errors.IsAlreadyExists(err) {
		return err
	}
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_cm, err := g.cli.CoreV1().ConfigMaps(kubeNamespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		cm.ResourceVersion = _cm.ResourceVersion
		_, err = g.cli.CoreV1().ConfigMaps(kubeNamespace).Update(cm)
		return err
	})
}

func (g *eventLogging) ensureDeployment() error {
	name := "eventrouter"
	d := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: kubeNamespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"component":     name,
					"logging-infra": name,
					"provider":      "openshift",
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"component":     name,
						"logging-infra": name,
						"provider":      "openshift",
					},
					Annotations: map[string]string{"scheduler.alpha.kubernetes.io/critical-pod": ""},
				},
				Spec: v1.PodSpec{
					ServiceAccountName: "eventrouter",
					Volumes: []v1.Volume{
						{
							Name: "config-volume",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: "eventrouter",
									},
								},
							},
						},
					},
					Containers: []v1.Container{
						{
							Name:  "eventrouter",
							Image: g.eventRouterImage(),
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "config-volume",
									MountPath: "/etc/eventrouter",
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := g.cli.AppsV1().Deployments(kubeNamespace).Create(d)
	if !errors.IsAlreadyExists(err) {
		return err
	}
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_d, err := g.cli.AppsV1().Deployments(kubeNamespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		d.ResourceVersion = _d.ResourceVersion
		_, err = g.cli.AppsV1().Deployments(kubeNamespace).Update(d)
		return err
	})

}

func (g *eventLogging) CreateOrUpdate() error {
	err := g.ensureNamespace(kubeNamespace)
	if err != nil {
		return err
	}

	err = g.ensureServiceAccount()
	if err != nil {
		return err
	}

	err = g.ensureClusterRole()
	if err != nil {
		return err
	}

	err = g.ensureClusterRoleBinding()
	if err != nil {
		return err
	}

	err = g.ensureClusterRoleBinding()
	if err != nil {
		return err
	}

	err = g.ensureConfigMap()
	if err != nil {
		return err
	}

	err = g.ensureDeployment()
	if err != nil {
		return err
	}

	return nil
}
