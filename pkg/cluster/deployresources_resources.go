package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/base64"
	"fmt"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	mgmtprivatedns "github.com/Azure/azure-sdk-for-go/services/privatedns/mgmt/2018-09-01/privatedns"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/openshift/installer/pkg/asset/ignition/machine"
	"github.com/openshift/installer/pkg/asset/installconfig"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

func dnsPrivateZone(installConfig *installconfig.InstallConfig) *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtprivatedns.PrivateZone{
			Name:     to.StringPtr(installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain),
			Type:     to.StringPtr("Microsoft.Network/privateDnsZones"),
			Location: to.StringPtr("global"),
		},
		APIVersion: azureclient.APIVersions["Microsoft.Network/privateDnsZones"],
	}
}

func dnsPrivateRecordAPIINT(infraID string, installConfig *installconfig.InstallConfig) *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtprivatedns.RecordSet{
			Name: to.StringPtr(installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain + "/api-int"),
			Type: to.StringPtr("Microsoft.Network/privateDnsZones/A"),
			RecordSetProperties: &mgmtprivatedns.RecordSetProperties{
				TTL: to.Int64Ptr(300),
				ARecords: &[]mgmtprivatedns.ARecord{
					{
						Ipv4Address: to.StringPtr(fmt.Sprintf("[reference('Microsoft.Network/loadBalancers/%s-internal', '%s').frontendIpConfigurations[0].properties.privateIPAddress]", infraID, azureclient.APIVersions["Microsoft.Network"])),
					},
				},
			},
		},
		APIVersion: azureclient.APIVersions["Microsoft.Network/privateDnsZones"],
		DependsOn: []string{
			"Microsoft.Network/loadBalancers/" + infraID + "-internal",
			"Microsoft.Network/privateDnsZones/" + installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain,
		},
	}
}

func dnsPrivateRecordAPI(infraID string, installConfig *installconfig.InstallConfig) *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtprivatedns.RecordSet{
			Name: to.StringPtr(installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain + "/api"),
			Type: to.StringPtr("Microsoft.Network/privateDnsZones/A"),
			RecordSetProperties: &mgmtprivatedns.RecordSetProperties{
				TTL: to.Int64Ptr(300),
				ARecords: &[]mgmtprivatedns.ARecord{
					{
						Ipv4Address: to.StringPtr(fmt.Sprintf("[reference('Microsoft.Network/loadBalancers/%s-internal', '%s').frontendIpConfigurations[0].properties.privateIPAddress]", infraID, azureclient.APIVersions["Microsoft.Network"])),
					},
				},
			},
		},
		APIVersion: azureclient.APIVersions["Microsoft.Network/privateDnsZones"],
		DependsOn: []string{
			"Microsoft.Network/loadBalancers/" + infraID + "-internal",
			"Microsoft.Network/privateDnsZones/" + installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain,
		},
	}
}

func dnsVirtualNetworkLink(vnetID string, installConfig *installconfig.InstallConfig) *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtprivatedns.VirtualNetworkLink{
			VirtualNetworkLinkProperties: &mgmtprivatedns.VirtualNetworkLinkProperties{
				VirtualNetwork: &mgmtprivatedns.SubResource{
					ID: to.StringPtr(vnetID),
				},
				RegistrationEnabled: to.BoolPtr(false),
			},
			Name:     to.StringPtr(installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain + "/" + installConfig.Config.ObjectMeta.Name + "-network-link"),
			Type:     to.StringPtr("Microsoft.Network/privateDnsZones/virtualNetworkLinks"),
			Location: to.StringPtr("global"),
		},
		APIVersion: azureclient.APIVersions["Microsoft.Network/privateDnsZones"],
		DependsOn: []string{
			"Microsoft.Network/privateDnsZones/" + installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain,
		},
	}
}

func networkPrivateLinkService(infraID, subscriptionID string, oc *api.OpenShiftCluster, installConfig *installconfig.InstallConfig) *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.PrivateLinkService{
			PrivateLinkServiceProperties: &mgmtnetwork.PrivateLinkServiceProperties{
				LoadBalancerFrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
					{
						ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', '%s-internal', 'internal-lb-ip-v4')]", infraID)),
					},
				},
				IPConfigurations: &[]mgmtnetwork.PrivateLinkServiceIPConfiguration{
					{
						PrivateLinkServiceIPConfigurationProperties: &mgmtnetwork.PrivateLinkServiceIPConfigurationProperties{
							Subnet: &mgmtnetwork.Subnet{
								ID: to.StringPtr(oc.Properties.MasterProfile.SubnetID),
							},
						},
						Name: to.StringPtr(infraID + "-pls-nic"),
					},
				},
				Visibility: &mgmtnetwork.PrivateLinkServicePropertiesVisibility{
					Subscriptions: &[]string{
						subscriptionID,
					},
				},
				AutoApproval: &mgmtnetwork.PrivateLinkServicePropertiesAutoApproval{
					Subscriptions: &[]string{
						subscriptionID,
					},
				},
			},
			Name:     to.StringPtr(infraID + "-pls"),
			Type:     to.StringPtr("Microsoft.Network/privateLinkServices"),
			Location: &installConfig.Config.Azure.Region,
		},
		APIVersion: azureclient.APIVersions["Microsoft.Network"],
		DependsOn: []string{
			"Microsoft.Network/loadBalancers/" + infraID + "-internal",
		},
	}
}

func networkPublicIPAddress(infraID string, installConfig *installconfig.InstallConfig) *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.PublicIPAddress{
			Sku: &mgmtnetwork.PublicIPAddressSku{
				Name: mgmtnetwork.PublicIPAddressSkuNameStandard,
			},
			PublicIPAddressPropertiesFormat: &mgmtnetwork.PublicIPAddressPropertiesFormat{
				PublicIPAllocationMethod: mgmtnetwork.Static,
			},
			Name:     to.StringPtr(infraID + "-pip-v4"),
			Type:     to.StringPtr("Microsoft.Network/publicIPAddresses"),
			Location: &installConfig.Config.Azure.Region,
		},
		APIVersion: azureclient.APIVersions["Microsoft.Network"],
	}
}

func networkInternalLoadBalancer(infraID string, oc *api.OpenShiftCluster, installConfig *installconfig.InstallConfig) *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.LoadBalancer{
			Sku: &mgmtnetwork.LoadBalancerSku{
				Name: mgmtnetwork.LoadBalancerSkuNameStandard,
			},
			LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
				FrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
					{
						FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
							PrivateIPAllocationMethod: mgmtnetwork.Dynamic,
							Subnet: &mgmtnetwork.Subnet{
								ID: to.StringPtr(oc.Properties.MasterProfile.SubnetID),
							},
						},
						Name: to.StringPtr("internal-lb-ip-v4"),
					},
				},
				BackendAddressPools: &[]mgmtnetwork.BackendAddressPool{
					{
						Name: to.StringPtr(infraID),
					},
				},
				LoadBalancingRules: &[]mgmtnetwork.LoadBalancingRule{
					{
						LoadBalancingRulePropertiesFormat: &mgmtnetwork.LoadBalancingRulePropertiesFormat{
							FrontendIPConfiguration: &mgmtnetwork.SubResource{
								ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', '%s-internal', 'internal-lb-ip-v4')]", infraID)),
							},
							BackendAddressPool: &mgmtnetwork.SubResource{
								ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', '%s-internal', '%[1]s')]", infraID)),
							},
							Probe: &mgmtnetwork.SubResource{
								ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/probes', '%s-internal', 'api-internal-probe')]", infraID)),
							},
							Protocol:             mgmtnetwork.TransportProtocolTCP,
							LoadDistribution:     mgmtnetwork.LoadDistributionDefault,
							FrontendPort:         to.Int32Ptr(6443),
							BackendPort:          to.Int32Ptr(6443),
							IdleTimeoutInMinutes: to.Int32Ptr(30),
							DisableOutboundSnat:  to.BoolPtr(true),
						},
						Name: to.StringPtr("api-internal-v4"),
					},
					{
						LoadBalancingRulePropertiesFormat: &mgmtnetwork.LoadBalancingRulePropertiesFormat{
							FrontendIPConfiguration: &mgmtnetwork.SubResource{
								ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', '%s-internal', 'internal-lb-ip-v4')]", infraID)),
							},
							BackendAddressPool: &mgmtnetwork.SubResource{
								ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', '%s-internal', '%[1]s')]", infraID)),
							},
							Probe: &mgmtnetwork.SubResource{
								ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/probes', '%s-internal', 'sint-probe')]", infraID)),
							},
							Protocol:             mgmtnetwork.TransportProtocolTCP,
							LoadDistribution:     mgmtnetwork.LoadDistributionDefault,
							FrontendPort:         to.Int32Ptr(22623),
							BackendPort:          to.Int32Ptr(22623),
							IdleTimeoutInMinutes: to.Int32Ptr(30),
						},
						Name: to.StringPtr("sint-v4"),
					},
				},
				Probes: &[]mgmtnetwork.Probe{
					{
						ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
							Protocol:          mgmtnetwork.ProbeProtocolHTTPS,
							Port:              to.Int32Ptr(6443),
							IntervalInSeconds: to.Int32Ptr(5),
							NumberOfProbes:    to.Int32Ptr(2),
							RequestPath:       to.StringPtr("/readyz"),
						},
						Name: to.StringPtr("api-internal-probe"),
					},
					{
						ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
							Protocol:          mgmtnetwork.ProbeProtocolHTTPS,
							Port:              to.Int32Ptr(22623),
							IntervalInSeconds: to.Int32Ptr(5),
							NumberOfProbes:    to.Int32Ptr(2),
							RequestPath:       to.StringPtr("/healthz"),
						},
						Name: to.StringPtr("sint-probe"),
					},
				},
			},
			Name:     to.StringPtr(infraID + "-internal"),
			Type:     to.StringPtr("Microsoft.Network/loadBalancers"),
			Location: &installConfig.Config.Azure.Region,
		},
		APIVersion: azureclient.APIVersions["Microsoft.Network"],
	}
}

func networkPublicLoadBalancer(infraID string, oc *api.OpenShiftCluster, installConfig *installconfig.InstallConfig) *arm.Resource {
	lb := &mgmtnetwork.LoadBalancer{
		Sku: &mgmtnetwork.LoadBalancerSku{
			Name: mgmtnetwork.LoadBalancerSkuNameStandard,
		},
		LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
			FrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
				{
					FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
						PublicIPAddress: &mgmtnetwork.PublicIPAddress{
							ID: to.StringPtr("[resourceId('Microsoft.Network/publicIPAddresses', '" + infraID + "-pip-v4')]"),
						},
					},
					Name: to.StringPtr("public-lb-ip-v4"),
				},
			},
			BackendAddressPools: &[]mgmtnetwork.BackendAddressPool{
				{
					Name: to.StringPtr(infraID),
				},
			},
			LoadBalancingRules: &[]mgmtnetwork.LoadBalancingRule{}, //required to override default LB rules for port 80 and 443
			Probes:             &[]mgmtnetwork.Probe{},             //required to override default LB rules for port 80 and 443
			OutboundRules: &[]mgmtnetwork.OutboundRule{
				{
					OutboundRulePropertiesFormat: &mgmtnetwork.OutboundRulePropertiesFormat{
						FrontendIPConfigurations: &[]mgmtnetwork.SubResource{
							{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', '" + infraID + "', 'public-lb-ip-v4')]"),
							},
						},
						BackendAddressPool: &mgmtnetwork.SubResource{
							ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', '%s', '%[1]s')]", infraID)),
						},
						Protocol:             mgmtnetwork.LoadBalancerOutboundRuleProtocolAll,
						IdleTimeoutInMinutes: to.Int32Ptr(30),
					},
					Name: to.StringPtr("outbound-rule-v4"),
				},
			},
		},
		Name:     to.StringPtr(infraID),
		Type:     to.StringPtr("Microsoft.Network/loadBalancers"),
		Location: &installConfig.Config.Azure.Region,
	}

	if oc.Properties.APIServerProfile.Visibility == api.VisibilityPublic {
		*lb.LoadBalancingRules = append(*lb.LoadBalancingRules, mgmtnetwork.LoadBalancingRule{
			LoadBalancingRulePropertiesFormat: &mgmtnetwork.LoadBalancingRulePropertiesFormat{
				FrontendIPConfiguration: &mgmtnetwork.SubResource{
					ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', '%s', 'public-lb-ip-v4')]", infraID)),
				},
				BackendAddressPool: &mgmtnetwork.SubResource{
					ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', '%s', '%[1]s')]", infraID)),
				},
				Probe: &mgmtnetwork.SubResource{
					ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/probes', '%s', 'api-internal-probe')]", infraID)),
				},
				Protocol:             mgmtnetwork.TransportProtocolTCP,
				LoadDistribution:     mgmtnetwork.LoadDistributionDefault,
				FrontendPort:         to.Int32Ptr(6443),
				BackendPort:          to.Int32Ptr(6443),
				IdleTimeoutInMinutes: to.Int32Ptr(30),
				DisableOutboundSnat:  to.BoolPtr(true),
			},
			Name: to.StringPtr("api-internal-v4"),
		})

		*lb.Probes = append(*lb.Probes, mgmtnetwork.Probe{
			ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
				Protocol:          mgmtnetwork.ProbeProtocolHTTPS,
				Port:              to.Int32Ptr(6443),
				IntervalInSeconds: to.Int32Ptr(5),
				NumberOfProbes:    to.Int32Ptr(2),
				RequestPath:       to.StringPtr("/readyz"),
			},
			Name: to.StringPtr("api-internal-probe"),
		})
	}

	return &arm.Resource{
		Resource:   lb,
		APIVersion: azureclient.APIVersions["Microsoft.Network"],
		DependsOn: []string{
			"Microsoft.Network/publicIPAddresses/" + infraID + "-pip-v4",
		},
	}
}

func networkBootstrapNIC(infraID string, oc *api.OpenShiftCluster, installConfig *installconfig.InstallConfig) *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.Interface{
			InterfacePropertiesFormat: &mgmtnetwork.InterfacePropertiesFormat{
				IPConfigurations: &[]mgmtnetwork.InterfaceIPConfiguration{
					{
						InterfaceIPConfigurationPropertiesFormat: &mgmtnetwork.InterfaceIPConfigurationPropertiesFormat{
							LoadBalancerBackendAddressPools: &[]mgmtnetwork.BackendAddressPool{
								{
									ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', '%s', '%[1]s')]", infraID)),
								},
								{
									ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', '%s-internal', '%[1]s')]", infraID)),
								},
							},
							Subnet: &mgmtnetwork.Subnet{
								ID: to.StringPtr(oc.Properties.MasterProfile.SubnetID),
							},
						},
						Name: to.StringPtr("bootstrap-nic-ip-v4"),
					},
				},
			},
			Name:     to.StringPtr(infraID + "-bootstrap-nic"),
			Type:     to.StringPtr("Microsoft.Network/networkInterfaces"),
			Location: &installConfig.Config.Azure.Region,
		},
		APIVersion: azureclient.APIVersions["Microsoft.Network"],
		DependsOn: []string{
			"Microsoft.Network/loadBalancers/" + infraID + "-internal",
			"Microsoft.Network/loadBalancers/" + infraID,
		},
	}
}

func networkMasterNICs(infraID string, oc *api.OpenShiftCluster, installConfig *installconfig.InstallConfig) *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.Interface{
			InterfacePropertiesFormat: &mgmtnetwork.InterfacePropertiesFormat{
				IPConfigurations: &[]mgmtnetwork.InterfaceIPConfiguration{
					{
						InterfaceIPConfigurationPropertiesFormat: &mgmtnetwork.InterfaceIPConfigurationPropertiesFormat{
							LoadBalancerBackendAddressPools: &[]mgmtnetwork.BackendAddressPool{
								{
									ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', '%s', '%[1]s')]", infraID)),
								},
								{
									ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', '%s-internal', '%[1]s')]", infraID)),
								},
							},
							Subnet: &mgmtnetwork.Subnet{
								ID: to.StringPtr(oc.Properties.MasterProfile.SubnetID),
							},
						},
						Name: to.StringPtr("pipConfig"),
					},
				},
			},
			Name:     to.StringPtr(fmt.Sprintf("[concat('%s-master', copyIndex(), '-nic')]", infraID)),
			Type:     to.StringPtr("Microsoft.Network/networkInterfaces"),
			Location: &installConfig.Config.Azure.Region,
		},
		APIVersion: azureclient.APIVersions["Microsoft.Network"],
		Copy: &arm.Copy{
			Name:  "networkcopy",
			Count: int(*installConfig.Config.ControlPlane.Replicas),
		},
		DependsOn: []string{
			"Microsoft.Network/loadBalancers/" + infraID + "-internal",
			"Microsoft.Network/loadBalancers/" + infraID,
		},
	}
}

func computeBoostrapVM(infraID string, oc *api.OpenShiftCluster, installConfig *installconfig.InstallConfig) *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtcompute.VirtualMachine{
			VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
				HardwareProfile: &mgmtcompute.HardwareProfile{
					VMSize: mgmtcompute.VirtualMachineSizeTypesStandardD4sV3,
				},
				StorageProfile: &mgmtcompute.StorageProfile{
					ImageReference: &mgmtcompute.ImageReference{
						Publisher: &installConfig.Config.Azure.Image.Publisher,
						Offer:     &installConfig.Config.Azure.Image.Offer,
						Sku:       &installConfig.Config.Azure.Image.SKU,
						Version:   &installConfig.Config.Azure.Image.Version,
					},
					OsDisk: &mgmtcompute.OSDisk{
						Name:         to.StringPtr(infraID + "-bootstrap_OSDisk"),
						Caching:      mgmtcompute.CachingTypesReadWrite,
						CreateOption: mgmtcompute.DiskCreateOptionTypesFromImage,
						DiskSizeGB:   to.Int32Ptr(100),
						ManagedDisk: &mgmtcompute.ManagedDiskParameters{
							StorageAccountType: mgmtcompute.StorageAccountTypesPremiumLRS,
						},
					},
				},
				OsProfile: &mgmtcompute.OSProfile{
					ComputerName:  to.StringPtr(infraID + "-bootstrap-vm"),
					AdminUsername: to.StringPtr("core"),
					AdminPassword: to.StringPtr("NotActuallyApplied!"),
					CustomData:    to.StringPtr(`[base64(concat('{"ignition":{"version":"2.2.0","config":{"replace":{"source":"https://cluster` + oc.Properties.StorageSuffix + `.blob.core.windows.net/ignition/bootstrap.ign?', listAccountSas(resourceId('Microsoft.Storage/storageAccounts', 'cluster` + oc.Properties.StorageSuffix + `'), '2019-04-01', parameters('sas')).accountSasToken, '"}}}}'))]`),
					LinuxConfiguration: &mgmtcompute.LinuxConfiguration{
						DisablePasswordAuthentication: to.BoolPtr(false),
					},
				},
				NetworkProfile: &mgmtcompute.NetworkProfile{
					NetworkInterfaces: &[]mgmtcompute.NetworkInterfaceReference{
						{
							ID: to.StringPtr("[resourceId('Microsoft.Network/networkInterfaces', '" + infraID + "-bootstrap-nic')]"),
						},
					},
				},
				DiagnosticsProfile: &mgmtcompute.DiagnosticsProfile{
					BootDiagnostics: &mgmtcompute.BootDiagnostics{
						Enabled:    to.BoolPtr(true),
						StorageURI: to.StringPtr("https://cluster" + oc.Properties.StorageSuffix + ".blob.core.windows.net/"),
					},
				},
			},
			Name:     to.StringPtr(infraID + "-bootstrap"),
			Type:     to.StringPtr("Microsoft.Compute/virtualMachines"),
			Location: &installConfig.Config.Azure.Region,
		},
		APIVersion: azureclient.APIVersions["Microsoft.Compute"],
		DependsOn: []string{
			"Microsoft.Network/networkInterfaces/" + infraID + "-bootstrap-nic",
			"Microsoft.Network/privateDnsZones/" + installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain + "/virtualNetworkLinks/" + installConfig.Config.ObjectMeta.Name + "-network-link",
		},
	}
}

func computeMasterVMs(infraID string, zones *[]string, machineMaster *machine.Master, oc *api.OpenShiftCluster, installConfig *installconfig.InstallConfig) *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtcompute.VirtualMachine{
			VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
				HardwareProfile: &mgmtcompute.HardwareProfile{
					VMSize: mgmtcompute.VirtualMachineSizeTypes(installConfig.Config.ControlPlane.Platform.Azure.InstanceType),
				},
				StorageProfile: &mgmtcompute.StorageProfile{
					ImageReference: &mgmtcompute.ImageReference{
						Publisher: &installConfig.Config.Azure.Image.Publisher,
						Offer:     &installConfig.Config.Azure.Image.Offer,
						Sku:       &installConfig.Config.Azure.Image.SKU,
						Version:   &installConfig.Config.Azure.Image.Version,
					},
					OsDisk: &mgmtcompute.OSDisk{
						Name:         to.StringPtr("[concat('" + infraID + "-master-', copyIndex(), '_OSDisk')]"),
						Caching:      mgmtcompute.CachingTypesReadOnly,
						CreateOption: mgmtcompute.DiskCreateOptionTypesFromImage,
						DiskSizeGB:   &installConfig.Config.ControlPlane.Platform.Azure.OSDisk.DiskSizeGB,
						ManagedDisk: &mgmtcompute.ManagedDiskParameters{
							StorageAccountType: mgmtcompute.StorageAccountTypesPremiumLRS,
						},
					},
				},
				OsProfile: &mgmtcompute.OSProfile{
					ComputerName:  to.StringPtr("[concat('" + infraID + "-master-', copyIndex())]"),
					AdminUsername: to.StringPtr("core"),
					AdminPassword: to.StringPtr("NotActuallyApplied!"),
					CustomData:    to.StringPtr(base64.StdEncoding.EncodeToString(machineMaster.File.Data)),
					LinuxConfiguration: &mgmtcompute.LinuxConfiguration{
						DisablePasswordAuthentication: to.BoolPtr(false),
					},
				},
				NetworkProfile: &mgmtcompute.NetworkProfile{
					NetworkInterfaces: &[]mgmtcompute.NetworkInterfaceReference{
						{
							ID: to.StringPtr("[resourceId('Microsoft.Network/networkInterfaces', concat('" + infraID + "-master', copyIndex(), '-nic'))]"),
						},
					},
				},
				DiagnosticsProfile: &mgmtcompute.DiagnosticsProfile{
					BootDiagnostics: &mgmtcompute.BootDiagnostics{
						Enabled:    to.BoolPtr(true),
						StorageURI: to.StringPtr("https://cluster" + oc.Properties.StorageSuffix + ".blob.core.windows.net/"),
					},
				},
			},
			Zones:    zones,
			Name:     to.StringPtr("[concat('" + infraID + "-master-', copyIndex())]"),
			Type:     to.StringPtr("Microsoft.Compute/virtualMachines"),
			Location: &installConfig.Config.Azure.Region,
		},
		APIVersion: azureclient.APIVersions["Microsoft.Compute"],
		Copy: &arm.Copy{
			Name:  "computecopy",
			Count: int(*installConfig.Config.ControlPlane.Replicas),
		},
		DependsOn: []string{
			"[concat('Microsoft.Network/networkInterfaces/" + infraID + "-master', copyIndex(), '-nic')]",
			"Microsoft.Network/privateDnsZones/" + installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain + "/virtualNetworkLinks/" + installConfig.Config.ObjectMeta.Name + "-network-link",
		},
	}
}
