provider "vsphere" {
  user                 = var.vsphere_username
  password             = var.vsphere_password
  vsphere_server       = var.vsphere_url
  allow_unverified_ssl = false
}

data "vsphere_datacenter" "datacenter" {
  name = var.vsphere_datacenter
}

data "vsphere_compute_cluster" "cluster" {
  name          = var.vsphere_cluster
  datacenter_id = data.vsphere_datacenter.datacenter.id
}

data "vsphere_datastore" "datastore" {
  name          = var.vsphere_datastore
  datacenter_id = data.vsphere_datacenter.datacenter.id
}

data "vsphere_network" "network" {
  name          = var.vsphere_network
  datacenter_id = data.vsphere_datacenter.datacenter.id
}

data "vsphere_virtual_machine" "template" {
  name          = var.vsphere_template
  datacenter_id = data.vsphere_datacenter.datacenter.id
}

resource "vsphere_tag_category" "category" {
  name        = "openshift-${var.cluster_id}"
  description = "Added by openshift-install do not remove"
  cardinality = "SINGLE"

  associable_types = [
    "VirtualMachine",
    "ResourcePool",
    "Folder"
  ]
}

resource "vsphere_tag" "tag" {
  name        = var.cluster_id
  category_id = vsphere_tag_category.category.id
  description = "Added by openshift-install do not remove"
}

resource "vsphere_folder" "folder" {
  path          = var.vsphere_folder
  type          = "vm"
  datacenter_id = data.vsphere_datacenter.datacenter.id
  tags          = [vsphere_tag.tag.id]
}


module "bootstrap" {
  source = "./bootstrap"

  ignition      = var.ignition_bootstrap
  resource_pool = data.vsphere_compute_cluster.cluster.resource_pool_id
  datastore     = data.vsphere_datastore.datastore.id
  folder        = vsphere_folder.folder.path
  network       = data.vsphere_network.network.id
  datacenter    = data.vsphere_datacenter.datacenter.id
  template      = data.vsphere_virtual_machine.template.id
  guest_id      = data.vsphere_virtual_machine.template.guest_id
  thin_disk     = data.vsphere_virtual_machine.template.disks.0.thin_provisioned
  scrub_disk    = data.vsphere_virtual_machine.template.disks.0.eagerly_scrub

  cluster_id = var.cluster_id
  tags       = [vsphere_tag.tag.id]
}

module "master" {
  source = "./master"

  // limitation of baremetal-runtimecfg.  The hostname must be master
  name           = "master"
  instance_count = var.master_count
  ignition       = var.ignition_master

  resource_pool = data.vsphere_compute_cluster.cluster.resource_pool_id
  datastore     = data.vsphere_datastore.datastore.id
  folder        = vsphere_folder.folder.path
  network       = data.vsphere_network.network.id
  datacenter    = data.vsphere_datacenter.datacenter.id
  template      = data.vsphere_virtual_machine.template.id
  guest_id      = data.vsphere_virtual_machine.template.guest_id
  thin_disk     = data.vsphere_virtual_machine.template.disks.0.thin_provisioned
  scrub_disk    = data.vsphere_virtual_machine.template.disks.0.eagerly_scrub
  tags          = [vsphere_tag.tag.id]

  cluster_domain   = var.cluster_domain
  cluster_id       = var.cluster_id
  memory           = var.vsphere_control_plane_memory_mib
  num_cpus         = var.vsphere_control_plane_num_cpus
  cores_per_socket = var.vsphere_control_plane_cores_per_socket
  disk_size        = var.vsphere_control_plane_disk_gib
}

