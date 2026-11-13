// Copyright Cloudsmith Ltd 2026
// SPDX-License-Identifier: MPL-2.0

package policy

var DataSourcePolicyListRead = dataSourcePolicyListRead

// SetPolicyListPageSize overrides the page size used by the policy list data
// source and returns a function that restores the previous value.
func SetPolicyListPageSize(size int64) (restore func()) {
	old := policyListPageSize
	policyListPageSize = size
	return func() { policyListPageSize = old }
}
