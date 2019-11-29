// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package warp

import "github.com/hchauvin/name_manager/pkg/name_manager"

func listFamilyNames(mng name_manager.NameManager) ([]string, error) {
	names, err := mng.List()
	if err != nil {
		return nil, err
	}

	familySet := make(map[string]struct{})
	for _, name := range names {
		familySet[name.Family] = struct{}{}
	}

	families := make([]string, 0, len(familySet))
	for family := range familySet {
		families = append(families, family)
	}
	return families, nil
}
