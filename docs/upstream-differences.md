# Upstream differences

This file catalogues the differences of install approach between ARO and
upstream OCP.


## Installer carry patches

* CARRY: HACK: remove dependency on github.com/openshift/installer/pkg/terraform
  from pkg/asset/cluster

  This enables to use installer as a library.

* CARRY: PARTIAL: allow platform credentials to be prepopulated in
  installconfig.PlatformCreds

  This avoids the installer going to disk to fetch the credentials, enabling one
  installer to handle multiple cluster installations simultaneously.

* CARRY: allow end user to specify Azure resource group on cluster creation

  This allows the RP to specify the cluster's resource group.  TODO: reduce the
  scope of this patch to get this upstream; don't allow end-users to choose
  their cluster resource group.

* CARRY: HACK: don't use managed identity on ARO

  At the moment OCP on Azure uses MSI for kubelets and controllers and one or
  more service principals for operators.  For now on ARO, simplify to all
  components using the user-provided SP.  Later, we'll reinstate a separate
  managed identity at least for worker kubelets.

* CARRY: HACK: don't set public DNS zone on DNS CRD in ARO

  In ARO, the public DNS zone is maintained by the service and the cluster
  operator does not have permissions to modify it.

* CARRY: allow VM image to be overriden

  This commit enables ARO to use platform-published VM images.

* CARRY: HACK: don't require BaseDomainResourceGroupName on ARO

  This prevents a validating failure when omitting a configurable which is never
  used in ARO.

* CARRY: HACK: Vendor kube-1.18.3. Update MCO dependencies and fixup code

  This enables ARO code to be on kube-1.18.3 libraries, instead of 1.17.

* CARRY: HACK: Commit /data/assets in the installer so it can be used as a library

## Installation differences

* ARO persists the install graph in the cluster storage account in a new "aro"
  container / "graph" blob.

* No managed identity (for now).

* No IPv6 support (for now).

* Upstream installer closely binds the installConfig (cluster) name, cluster
  domain name, infra ID and Azure resource name prefix.  ARO separates these out
  a little.  The installConfig (cluster) name and the domain name remain bound;
  the infra ID and Azure resource name prefix are taken from the ARO resource
  name.

* API server public IP domain name label is not set.

* ARO uses first party RHCOS OS images published by Microsoft.

* ARO never creates xxxxx-bootstrap-pip for bootstrap VM, or the corresponding
  NSG rule.

* ARO API server LB uses Azure outboundRule rather than port 27627 inbound rule.

* `xxxxx` LB uses Azure outboundRule and outbound IP `xxxxx-outbound-pip-v4`.

* ARO deploys a private link service in order for the RP to be able to
  communicate with the cluster.
