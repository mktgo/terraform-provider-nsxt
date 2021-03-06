/* Copyright © 2019 VMware, Inc. All Rights Reserved.
   SPDX-License-Identifier: MPL-2.0 */

package nsxt

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/infra"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/infra/tier_1s/locale_services"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/model"
	"log"
	"strconv"
	"strings"
)

func resourceNsxtPolicyTier1GatewayInterface() *schema.Resource {
	return &schema.Resource{
		Create: resourceNsxtPolicyTier1GatewayInterfaceCreate,
		Read:   resourceNsxtPolicyTier1GatewayInterfaceRead,
		Update: resourceNsxtPolicyTier1GatewayInterfaceUpdate,
		Delete: resourceNsxtPolicyTier1GatewayInterfaceDelete,
		Importer: &schema.ResourceImporter{
			State: resourceNsxtPolicyTier1GatewayInterfaceImport,
		},

		Schema: map[string]*schema.Schema{
			"nsx_id":                 getNsxIDSchema(),
			"path":                   getPathSchema(),
			"display_name":           getDisplayNameSchema(),
			"description":            getDescriptionSchema(),
			"revision":               getRevisionSchema(),
			"tag":                    getTagsSchema(),
			"gateway_path":           getPolicyPathSchema(true, true, "Policy path for tier1 gateway"),
			"segment_path":           getPolicyPathSchema(true, true, "Policy path for connected segment"),
			"subnets":                getGatewayInterfaceSubnetsSchema(),
			"mtu":                    getMtuSchema(),
			"ipv6_ndra_profile_path": getIPv6NDRAPathSchema(),
			"urpf_mode":              getGatewayInterfaceUrpfModeSchema(),
		},
	}
}

func getGatewayInterfaceSubnetList(d *schema.ResourceData) []model.InterfaceSubnet {
	subnets := interface2StringList(d.Get("subnets").([]interface{}))
	var interfaceSubnetList []model.InterfaceSubnet
	for _, subnet := range subnets {
		result := strings.Split(subnet, "/")
		var ipAddresses []string
		ipAddresses = append(ipAddresses, result[0])
		prefix, _ := strconv.Atoi(result[1])
		interfaceSubnet := model.InterfaceSubnet{
			IpAddresses: ipAddresses,
			PrefixLen:   int64(prefix),
		}
		interfaceSubnetList = append(interfaceSubnetList, interfaceSubnet)
	}

	return interfaceSubnetList
}

func resourceNsxtPolicyTier1GatewayInterfaceCreate(d *schema.ResourceData, m interface{}) error {
	connector := getPolicyConnector(m)
	client := locale_services.NewDefaultInterfacesClient(connector)

	id := d.Get("nsx_id").(string)
	tier1Path := d.Get("gateway_path").(string)
	tier1ID := getPolicyIDFromPath(tier1Path)
	localeService, err := resourceNsxtPolicyTier1GatewayGetLocaleServiceEntry(tier1ID, connector)
	if err != nil {
		return err
	}
	if localeService == nil {
		return fmt.Errorf("Edge cluster is mandatory on gateway %s in order to create interfaces", tier1ID)
	}
	localeServiceID := *localeService.Id

	if id == "" {
		id = newUUID()
	} else {
		_, err := client.Get(tier1ID, localeServiceID, id)
		if err == nil {
			return fmt.Errorf("Interface with ID '%s' already exists on Tier1 Gateway %s", id, tier1ID)
		} else if !isNotFoundError(err) {
			return err
		}
	}

	displayName := d.Get("display_name").(string)
	description := d.Get("description").(string)
	segmentPath := d.Get("segment_path").(string)
	tags := getPolicyTagsFromSchema(d)
	interfaceSubnetList := getGatewayInterfaceSubnetList(d)
	var ipv6ProfilePaths []string
	if d.Get("ipv6_ndra_profile_path").(string) != "" {
		ipv6ProfilePaths = append(ipv6ProfilePaths, d.Get("ipv6_ndra_profile_path").(string))
	}
	mtu := int64(d.Get("mtu").(int))
	obj := model.Tier1Interface{
		Id:               &id,
		DisplayName:      &displayName,
		Description:      &description,
		Tags:             tags,
		Subnets:          interfaceSubnetList,
		SegmentPath:      segmentPath,
		Ipv6ProfilePaths: ipv6ProfilePaths,
	}

	if mtu > 0 {
		obj.Mtu = &mtu
	}

	if nsxVersionHigherOrEqual("3.0.0") {
		urpfMode := d.Get("urpf_mode").(string)
		obj.UrpfMode = &urpfMode
	}

	// Create the resource using PATCH
	log.Printf("[INFO] Creating tier1 interface with ID %s", id)
	err = client.Patch(tier1ID, localeServiceID, id, obj, nil)
	if err != nil {
		return handleCreateError("Tier1 Interface", id, err)
	}

	d.SetId(id)
	d.Set("nsx_id", id)

	return resourceNsxtPolicyTier1GatewayInterfaceRead(d, m)
}

func resourceNsxtPolicyTier1GatewayInterfaceRead(d *schema.ResourceData, m interface{}) error {
	connector := getPolicyConnector(m)
	client := locale_services.NewDefaultInterfacesClient(connector)

	id := d.Id()
	tier1Path := d.Get("gateway_path").(string)
	tier1ID := getPolicyIDFromPath(tier1Path)
	if id == "" || tier1ID == "" {
		return fmt.Errorf("Error obtaining Tier1 Interface id")
	}

	obj, err := client.Get(tier1ID, defaultPolicyLocaleServiceID, id)
	if err != nil {
		return handleReadError(d, "Tier1 Interface", id, err)
	}

	d.Set("display_name", obj.DisplayName)
	d.Set("description", obj.Description)
	setPolicyTagsInSchema(d, obj.Tags)
	d.Set("nsx_id", id)
	d.Set("path", obj.Path)
	d.Set("revision", obj.Revision)
	d.Set("segment_path", obj.SegmentPath)
	if obj.Ipv6ProfilePaths != nil {
		d.Set("ipv6_ndra_profile_path", obj.Ipv6ProfilePaths[0]) // only one supported for now
	}
	d.Set("mtu", obj.Mtu)

	if obj.Subnets != nil {
		var subnetList []string
		for _, subnet := range obj.Subnets {
			cidr := fmt.Sprintf("%s/%d", subnet.IpAddresses[0], subnet.PrefixLen)
			subnetList = append(subnetList, cidr)
		}
		d.Set("subnets", subnetList)
	}

	d.Set("urpf_mode", obj.UrpfMode)

	return nil
}

func resourceNsxtPolicyTier1GatewayInterfaceUpdate(d *schema.ResourceData, m interface{}) error {
	connector := getPolicyConnector(m)
	client := locale_services.NewDefaultInterfacesClient(connector)

	id := d.Id()
	tier1Path := d.Get("gateway_path").(string)
	tier1ID := getPolicyIDFromPath(tier1Path)
	if id == "" || tier1ID == "" {
		return fmt.Errorf("Error obtaining Tier1 id")
	}

	displayName := d.Get("display_name").(string)
	description := d.Get("description").(string)
	tags := getPolicyTagsFromSchema(d)
	interfaceSubnetList := getGatewayInterfaceSubnetList(d)
	segmentPath := d.Get("segment_path").(string)
	var ipv6ProfilePaths []string
	if d.Get("ipv6_ndra_profile_path").(string) != "" {
		ipv6ProfilePaths = append(ipv6ProfilePaths, d.Get("ipv6_ndra_profile_path").(string))
	}
	mtu := int64(d.Get("mtu").(int))
	revision := int64(d.Get("revision").(int))
	obj := model.Tier1Interface{
		DisplayName:      &displayName,
		Description:      &description,
		Tags:             tags,
		Subnets:          interfaceSubnetList,
		SegmentPath:      segmentPath,
		Ipv6ProfilePaths: ipv6ProfilePaths,
		Revision:         &revision,
	}

	if mtu > 0 {
		obj.Mtu = &mtu
	}

	if nsxVersionHigherOrEqual("3.0.0") {
		urpfMode := d.Get("urpf_mode").(string)
		obj.UrpfMode = &urpfMode
	}

	_, err := client.Update(tier1ID, defaultPolicyLocaleServiceID, id, obj, nil)
	if err != nil {
		return handleUpdateError("Tier1 Interface", id, err)
	}

	return resourceNsxtPolicyTier1GatewayInterfaceRead(d, m)
}

func resourceNsxtPolicyTier1GatewayInterfaceDelete(d *schema.ResourceData, m interface{}) error {
	connector := getPolicyConnector(m)
	client := locale_services.NewDefaultInterfacesClient(connector)

	id := d.Id()
	tier1Path := d.Get("gateway_path").(string)
	tier1ID := getPolicyIDFromPath(tier1Path)
	if id == "" || tier1ID == "" {
		return fmt.Errorf("Error obtaining Tier1 Interface id")
	}

	err := client.Delete(tier1ID, defaultPolicyLocaleServiceID, id, nil)
	if err != nil {
		return handleDeleteError("Tier1 Interface", id, err)
	}

	return nil
}

func resourceNsxtPolicyTier1GatewayInterfaceImport(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	importID := d.Id()
	s := strings.Split(importID, "/")
	if len(s) != 3 {
		return nil, fmt.Errorf("Please provide <gateway-id>/<locale-service-id>/<interface-id> as an input")
	}

	gwID := s[0]
	connector := getPolicyConnector(m)
	client := infra.NewDefaultTier1sClient(connector)
	tier1GW, err := client.Get(gwID)
	if err != nil {
		return nil, err
	}
	d.Set("gateway_path", tier1GW.Path)
	d.Set("locale_service_id", s[1])

	d.SetId(s[2])

	return []*schema.ResourceData{d}, nil

}
