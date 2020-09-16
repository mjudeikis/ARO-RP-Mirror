terraform {
  required_version = ">= 0.12"
}

variable "machine_v4_cidrs" {
  type = list(string)

  description = <<EOF
The list of IPv4 address spaces from which to assign machine IPs.
EOF

}

variable "machine_v6_cidrs" {
  type = list(string)

  description = <<EOF
The list of IPv6 address spaces from which to assign machine IPs.
EOF

}

variable "master_count" {
  type = string

  default = "1"

  description = <<EOF
The number of master nodes to be created.
This applies only to cloud platforms.
EOF

}

variable "base_domain" {
  type = string

  description = <<EOF
The base DNS domain of the cluster. It must NOT contain a trailing period. Some
DNS providers will automatically add this if necessary.

Example: `openshift.example.com`.

Note: This field MUST be set manually prior to creating the cluster.
This applies only to cloud platforms.
EOF

}

variable "cluster_domain" {
  type = string

  description = <<EOF
The domain of the cluster. It must NOT contain a trailing period. Some
DNS providers will automatically add this if necessary.

All the records for the cluster are created under this domain.

Note: This field MUST be set manually prior to creating the cluster.
EOF

}

variable "ignition_master" {
  type = string

  default = ""

  description = <<EOF
(internal) Ignition config file contents. This is automatically generated by the installer.
EOF

}

variable "ignition_bootstrap" {
  type = string

  default = ""

  description = <<EOF
(internal) Ignition config file contents. This is automatically generated by the installer.
EOF

}

// This variable is generated by OpenShift internally. Do not modify
variable "cluster_id" {
  type = string

  description = <<EOF
(internal) The OpenShift cluster id.

This cluster id must be of max length 27 and must have only alphanumeric or hyphen characters.
EOF

}

variable "use_ipv4" {
  type = bool

  description = <<EOF
Should the cluster be created with ipv4 networking.
EOF

}

variable "use_ipv6" {
  type = bool

  description = <<EOF
Should the cluster be created with ipv6 networking.
EOF

}
