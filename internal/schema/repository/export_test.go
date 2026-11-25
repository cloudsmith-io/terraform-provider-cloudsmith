// Copyright Cloudsmith Ltd 2026
// SPDX-License-Identifier: MPL-2.0

package repository

var ResourceRepositoryStorageRegionUpdate = resourceRepositoryStorageRegionUpdate

func ServerDefaultValues() map[string]string {
	values := make(map[string]string, len(serverDefaults))
	for name, documented := range serverDefaults {
		values[name] = documented.Value
	}

	return values
}
