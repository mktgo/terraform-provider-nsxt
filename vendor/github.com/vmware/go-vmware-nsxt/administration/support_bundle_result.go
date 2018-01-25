/* Copyright © 2017 VMware, Inc. All Rights Reserved.
   SPDX-License-Identifier: BSD-2-Clause

   Generated by: https://github.com/swagger-api/swagger-codegen.git */

package administration

type SupportBundleResult struct {

	// Nodes where bundles were not generated or not copied to remote server
	FailedNodes []FailedNodeSupportBundleResult `json:"failed_nodes,omitempty"`

	// Nodes where bundle generation is pending or in progress
	RemainingNodes []RemainingSupportBundleNode `json:"remaining_nodes,omitempty"`

	// Request properties
	RequestProperties *SupportBundleRequest `json:"request_properties,omitempty"`

	// Nodes whose bundles were successfully copied to remote file server
	SuccessNodes []SuccessNodeSupportBundleResult `json:"success_nodes,omitempty"`
}