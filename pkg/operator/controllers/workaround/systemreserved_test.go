package workaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	machinev1beta1 "github.com/openshift/cluster-api/pkg/apis/machine/v1beta1"
	clusterapi "github.com/openshift/cluster-api/pkg/client/clientset_generated/clientset"
	fakeclusterclient "github.com/openshift/cluster-api/pkg/client/clientset_generated/clientset/fake"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	fakemcoclient "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ktesting "k8s.io/client-go/testing"

	"github.com/Azure/ARO-RP/pkg/util/cmp"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	mock_dynamichelper "github.com/Azure/ARO-RP/pkg/util/mocks/dynamichelper"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
)

func machineset(vmSize string) *machinev1beta1.MachineSet {
	return &machinev1beta1.MachineSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-worker-profile-1",
			Namespace: "openshift-machine-api",
		},
		Spec: machinev1beta1.MachineSetSpec{
			Template: machinev1beta1.MachineTemplateSpec{
				Spec: machinev1beta1.MachineSpec{
					ProviderSpec: machinev1beta1.ProviderSpec{
						Value: &runtime.RawExtension{
							Raw: []byte(fmt.Sprintf(`{
"apiVersion": "azureproviderconfig.openshift.io/v1beta1",
"kind": "AzureMachineProviderSpec",
"osDisk": {
"diskSizeGB": 512
},
"vmSize": "%s"
}`, vmSize)),
						},
					},
				},
			},
		},
	}
}

func TestSystemreservedEnsure(t *testing.T) {
	tests := []struct {
		name                         string
		mcocli                       *fakemcoclient.Clientset
		clustercli                   clusterapi.Interface
		mocker                       func(mdh *mock_dynamichelper.MockDynamicHelper)
		machineConfigPoolNeedsUpdate bool
		wantErr                      bool
	}{
		{
			name: "first time create",
			mcocli: fakemcoclient.NewSimpleClientset(&mcv1.MachineConfigPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "worker",
				},
			}),
			machineConfigPoolNeedsUpdate: true,
			clustercli:                   fakeclusterclient.NewSimpleClientset(machineset("Standard_D8as_v4")),
			mocker: func(mdh *mock_dynamichelper.MockDynamicHelper) {
				mdh.EXPECT().Ensure(gomock.Any()).Return(nil)
			},
		},
		{
			name: "nothing to be done",
			mcocli: fakemcoclient.NewSimpleClientset(&mcv1.MachineConfigPool{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "worker",
					Labels: map[string]string{labelName: labelValue},
				},
			}),
			clustercli: fakeclusterclient.NewSimpleClientset(machineset("Standard_D8as_v4")),
			mocker: func(mdh *mock_dynamichelper.MockDynamicHelper) {
				mdh.EXPECT().Ensure(gomock.Any()).Return(nil)
			},
		},
	}
	err := mcv1.AddToScheme(scheme.Scheme)
	if err != nil {
		t.Error(err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mdh := mock_dynamichelper.NewMockDynamicHelper(controller)
			sr := &systemreserved{
				mcocli:     tt.mcocli,
				clustercli: tt.clustercli,
				dh:         mdh,
				log:        utillog.GetLogger(),
			}

			var updated bool
			tt.mcocli.PrependReactor("update", "machineconfigpools", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
				updated = true
				return false, nil, nil
			})

			tt.mocker(mdh)
			if err := sr.Ensure(); (err != nil) != tt.wantErr {
				t.Errorf("systemreserved.Ensure() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.machineConfigPoolNeedsUpdate != updated {
				t.Errorf("systemreserved.Ensure() updated %v, machineConfigPoolNeedsUpdate = %v", updated, tt.machineConfigPoolNeedsUpdate)
			}
		})
	}
}

func TestSystemreservedKubeletConfig(t *testing.T) {
	tests := []struct {
		vmSize  string
		wantMem string
		wantCPU string
		wantErr bool
	}{
		{
			vmSize:  "Standard_D8as_v4",
			wantMem: "3560Mi",
			wantCPU: "500m",
		},
		{
			vmSize:  "Standard_D16s_v3",
			wantMem: "5480Mi",
			wantCPU: "500m",
		},
		{
			vmSize:  "Standard_E32s_v3",
			wantMem: "11880Mi",
			wantCPU: "500m",
		},
	}

	err := mcv1.AddToScheme(scheme.Scheme)
	if err != nil {
		t.Error(err)
	}
	for _, tt := range tests {
		t.Run(tt.vmSize, func(t *testing.T) {
			sr := &systemreserved{}
			got, err := sr.kubeletConfig(tt.vmSize)
			if (err != nil) != tt.wantErr {
				t.Errorf("systemreserved.kubeletConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			want := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "machineconfiguration.openshift.io/v1",
					"kind":       "KubeletConfig",
					"metadata": map[string]interface{}{
						"creationTimestamp": nil,
						"labels": map[string]interface{}{
							"aro.openshift.io/limits": "",
						},
						"name": kubeletConfigName,
					},
					"spec": map[string]interface{}{
						"kubeletConfig": map[string]interface{}{
							"systemReserved": map[string]interface{}{
								"cpu":    tt.wantCPU,
								"memory": tt.wantMem,
							},
						},
						"machineConfigPoolSelector": map[string]interface{}{
							"matchLabels": map[string]interface{}{
								"aro.openshift.io/limits": "",
							},
						},
					},
					"status": map[string]interface{}{
						"conditions": nil,
					},
				},
			}

			if !reflect.DeepEqual(got, want) {
				t.Errorf("systemreserved.kubeletConfig() = %v, want %v", got, want)
				t.Error(cmp.Diff(got, want))
			}
		})
	}
}
