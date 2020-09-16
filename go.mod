module github.com/Azure/ARO-RP

go 1.13

exclude (
	github.com/etcd-io/bbolt v1.3.5
	github.com/terraform-providers/terraform-provider-aws v0.0.0
	github.com/terraform-providers/terraform-provider-azurerm v0.0.0
)

require (
	github.com/Azure/azure-sdk-for-go v46.0.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.4
	github.com/Azure/go-autorest/autorest/adal v0.9.2
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.1
	github.com/Azure/go-autorest/autorest/date v0.3.0
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/Azure/go-autorest/autorest/validation v0.2.1-0.20191028180845-3492b2aff503
	github.com/Azure/go-autorest/tracing v0.6.0
	github.com/BurntSushi/toml v0.3.1
	github.com/VividCortex/ewma v1.1.1 // indirect
	github.com/alvaroloes/enumer v1.1.2
	github.com/apparentlymart/go-cidr v1.1.0
	github.com/containers/image v3.0.2+incompatible
	github.com/coreos/go-systemd v0.0.0
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/etcd-io/bbolt v0.0.0-00010101000000-000000000000 // indirect
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-bindata/go-bindata v3.1.2+incompatible
	github.com/go-logr/logr v0.2.1
	github.com/go-test/deep v1.0.5
	github.com/golang/mock v1.3.1
	github.com/google/go-cmp v0.5.2
	github.com/googleapis/gnostic v0.4.1
	github.com/gorilla/mux v1.8.0
	github.com/jim-minter/go-cosmosdb v0.0.0-20200923160222-1528d2db09d6
	github.com/jmespath/go-jmespath v0.3.0
	github.com/onsi/ginkgo v1.12.1
	github.com/onsi/gomega v1.10.1
	github.com/openshift/api v3.9.1-0.20191111211345-a27ff30ebf09+incompatible
	github.com/openshift/client-go v0.0.0-20200729195840-c2b1adc6bed6
	github.com/openshift/cluster-api v0.0.0-20191129101638-b09907ac6668
	github.com/openshift/console-operator v0.0.0-20200904235146-182ff9dbe857
	github.com/openshift/installer v0.0.0-00010101000000-000000000000
	github.com/openshift/machine-api-operator v0.2.1-0.20200527204437-14e5e0c7d862
	github.com/openshift/machine-config-operator v4.2.0-alpha.0.0.20190917115525-033375cbe820+incompatible
	github.com/operator-framework/operator-sdk v0.19.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/common v0.13.0
	github.com/satori/go.uuid v1.2.0
	github.com/shurcooL/vfsgen v0.0.0-20200824052919-0d455de96546
	github.com/sirupsen/logrus v1.6.0
	github.com/ugorji/go/codec v1.1.7
	github.com/vbauerster/mpb v3.4.0+incompatible // indirect
	golang.org/x/crypto v0.0.0-20200820211705-5c72a883971a
	golang.org/x/sync v0.0.0-20200625203802-6e8e738ad208
	golang.org/x/tools v0.0.0-20200430192856-2840dafb9ee1
	k8s.io/api v0.19.0-rc.2
	k8s.io/apiextensions-apiserver v0.19.0-rc.2
	k8s.io/apimachinery v0.19.0-rc.2
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/code-generator v0.18.3
	sigs.k8s.io/cluster-api-provider-azure v0.0.0
	sigs.k8s.io/controller-runtime v0.6.2
	sigs.k8s.io/controller-tools v0.3.0
)

replace (
	github.com/coreos/go-systemd => github.com/coreos/go-systemd/v22 v22.0.0 // Pin non-versioned import to v22.0.0
	github.com/coreos/ignition => github.com/coreos/ignition v0.35.0
	github.com/etcd-io/bbolt => go.etcd.io/bbolt v1.3.5 // indirect from containers/images
	github.com/go-log/log => github.com/go-log/log v0.1.1-0.20181211034820-a514cf01a3eb
	github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.4.0
	github.com/metal3-io/baremetal-operator => github.com/openshift/baremetal-operator v0.0.0-20200611190251-d997d9c06ba8
	github.com/metal3-io/cluster-api-provider-baremetal => github.com/openshift/cluster-api-provider-baremetal v0.0.0-20190821174549-a2a477909c1d
	github.com/openshift/api => github.com/openshift/api v0.0.0-20200413201024-c6e8c9b6eb9a
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20200521150516-05eb9880269c
	github.com/openshift/installer => github.com/jim-minter/installer v0.0.0-20200915125021-47a8a781f957
	github.com/openshift/machine-api-operator => github.com/openshift/machine-api-operator v0.0.0-20200529045911-d19e8d007f7c
	github.com/openshift/machine-config-operator => github.com/openshift/machine-config-operator v0.0.0-20200904000724-41d29dde06d6 // ARO Carry-on: to avoid k8s version miss-align
	github.com/vmware/govmomi => github.com/vmware/govmomi v0.22.2-0.20200420222347-5fceac570f29
	google.golang.org/api => google.golang.org/api v0.13.0
	k8s.io/api => k8s.io/api v0.18.3
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.18.3
	k8s.io/apimachinery => k8s.io/apimachinery v0.18.3
	k8s.io/apiserver => k8s.io/apiserver v0.18.3
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.18.3
	k8s.io/client-go => k8s.io/client-go v0.18.3
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.18.3
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.18.3
	k8s.io/code-generator => k8s.io/code-generator v0.18.3
	k8s.io/component-base => k8s.io/component-base v0.18.3
	k8s.io/cri-api => k8s.io/cri-api v0.18.3
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.18.3
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.18.3
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.18.3
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.18.3
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.18.3
	k8s.io/kubectl => k8s.io/kubectl v0.18.3
	k8s.io/kubelet => k8s.io/kubelet v0.18.3
	k8s.io/kubernetes => k8s.io/kubernetes v1.18.3
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.18.3
	k8s.io/metrics => k8s.io/metrics v0.18.3
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.18.3
	sigs.k8s.io/cluster-api-provider-aws => github.com/openshift/cluster-api-provider-aws v0.2.1-0.20200506073438-9d49428ff837 // Pin OpenShift fork
	sigs.k8s.io/cluster-api-provider-azure => github.com/openshift/cluster-api-provider-azure v0.0.0-20200529030741-17d4edc5142f
	sigs.k8s.io/cluster-api-provider-openstack => github.com/openshift/cluster-api-provider-openstack v0.0.0-20200526112135-319a35b2e38e
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.6.0
)
