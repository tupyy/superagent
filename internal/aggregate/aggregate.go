package aggregate

import (
	v1alpha1 "github.com/kubev2v/migration-planner/api/v1alpha1"
)

func MergeInventories(inventories []v1alpha1.Inventory) v1alpha1.Inventory {
	result := v1alpha1.Inventory{
		VcenterId: "aggregated_vcenter_id",
		Clusters:  make(map[string]v1alpha1.InventoryData),
	}
	version := "aggregated_version"
	result.VcenterVersion = &version

	for _, inv := range inventories {
		for name, data := range inv.Clusters {
			key := inv.VcenterId + "/" + name
			result.Clusters[key] = data
		}

		if inv.Vcenter != nil {
			if result.Vcenter == nil {
				cp := *inv.Vcenter
				result.Vcenter = &cp
			} else {
				merged := mergeInventoryData(*result.Vcenter, *inv.Vcenter)
				result.Vcenter = &merged
			}
		}
	}

	return result
}

func mergeInventoryData(a, b v1alpha1.InventoryData) v1alpha1.InventoryData {
	return v1alpha1.InventoryData{
		Infra: mergeInfra(a.Infra, b.Infra),
		Vms:   mergeVMs(a.Vms, b.Vms),
	}
}

func mergeInfra(a, b v1alpha1.Infra) v1alpha1.Infra {
	result := v1alpha1.Infra{
		Datastores:      append(a.Datastores, b.Datastores...),
		Networks:        append(a.Networks, b.Networks...),
		HostPowerStates: mergeIntMap(a.HostPowerStates, b.HostPowerStates),
		TotalHosts:      a.TotalHosts + b.TotalHosts,
	}

	if a.TotalDatacenters != nil || b.TotalDatacenters != nil {
		v := ptrIntVal(a.TotalDatacenters) + ptrIntVal(b.TotalDatacenters)
		result.TotalDatacenters = &v
	}

	if a.Hosts != nil || b.Hosts != nil {
		merged := append(ptrSlice(a.Hosts), ptrSlice(b.Hosts)...)
		result.Hosts = &merged
	}

	if a.ClustersPerDatacenter != nil || b.ClustersPerDatacenter != nil {
		merged := append(ptrSlice(a.ClustersPerDatacenter), ptrSlice(b.ClustersPerDatacenter)...)
		result.ClustersPerDatacenter = &merged
	}

	return result
}

func mergeVMs(a, b v1alpha1.VMs) v1alpha1.VMs {
	result := v1alpha1.VMs{
		CpuCores:             mergeBreakdown(a.CpuCores, b.CpuCores),
		DiskCount:            mergeBreakdown(a.DiskCount, b.DiskCount),
		DiskGB:               mergeBreakdown(a.DiskGB, b.DiskGB),
		RamGB:                mergeBreakdown(a.RamGB, b.RamGB),
		PowerStates:          mergeIntMap(a.PowerStates, b.PowerStates),
		MigrationWarnings:    mergeMigrationIssues(a.MigrationWarnings, b.MigrationWarnings),
		NotMigratableReasons: mergeMigrationIssues(a.NotMigratableReasons, b.NotMigratableReasons),
		Total:                a.Total + b.Total,
		TotalMigratable:      a.TotalMigratable + b.TotalMigratable,
	}

	if a.NicCount != nil || b.NicCount != nil {
		v := mergeBreakdown(ptrBreakdownVal(a.NicCount), ptrBreakdownVal(b.NicCount))
		result.NicCount = &v
	}

	result.TotalMigratableWithWarnings = sumPtrInt(a.TotalMigratableWithWarnings, b.TotalMigratableWithWarnings)
	result.TotalWithSharedDisks = sumPtrInt(a.TotalWithSharedDisks, b.TotalWithSharedDisks)

	result.ComplexityDistribution = mergeDiskSizeTierMap(a.ComplexityDistribution, b.ComplexityDistribution)
	result.DiskComplexityTier = mergeDiskSizeTierMap(a.DiskComplexityTier, b.DiskComplexityTier)
	result.DiskSizeTier = mergeDiskSizeTierMap(a.DiskSizeTier, b.DiskSizeTier)
	result.DiskTypes = mergeDiskTypeMap(a.DiskTypes, b.DiskTypes)

	result.DistributionByComplexity = mergePtrIntMap(a.DistributionByComplexity, b.DistributionByComplexity)
	result.DistributionByCpuTier = mergePtrIntMap(a.DistributionByCpuTier, b.DistributionByCpuTier)
	result.DistributionByMemoryTier = mergePtrIntMap(a.DistributionByMemoryTier, b.DistributionByMemoryTier)
	result.DistributionByNicCount = mergePtrIntMap(a.DistributionByNicCount, b.DistributionByNicCount)

	result.Os = mergePtrIntMap(a.Os, b.Os)
	result.OsInfo = mergeOsInfoMap(a.OsInfo, b.OsInfo)

	return result
}

func mergeBreakdown(a, b v1alpha1.VMResourceBreakdown) v1alpha1.VMResourceBreakdown {
	return v1alpha1.VMResourceBreakdown{
		Total:                          a.Total + b.Total,
		TotalForMigratable:             a.TotalForMigratable + b.TotalForMigratable,
		TotalForMigratableWithWarnings: a.TotalForMigratableWithWarnings + b.TotalForMigratableWithWarnings,
		TotalForNotMigratable:          a.TotalForNotMigratable + b.TotalForNotMigratable,
	}
}

func mergeIntMap(a, b map[string]int) map[string]int {
	if a == nil && b == nil {
		return nil
	}
	result := make(map[string]int)
	for k, v := range a {
		result[k] += v
	}
	for k, v := range b {
		result[k] += v
	}
	return result
}

func mergePtrIntMap(a, b *map[string]int) *map[string]int {
	if a == nil && b == nil {
		return nil
	}
	var ma, mb map[string]int
	if a != nil {
		ma = *a
	}
	if b != nil {
		mb = *b
	}
	result := mergeIntMap(ma, mb)
	return &result
}

func mergeDiskSizeTierMap(a, b *map[string]v1alpha1.DiskSizeTierSummary) *map[string]v1alpha1.DiskSizeTierSummary {
	if a == nil && b == nil {
		return nil
	}
	result := make(map[string]v1alpha1.DiskSizeTierSummary)
	if a != nil {
		for k, v := range *a {
			result[k] = v
		}
	}
	if b != nil {
		for k, v := range *b {
			existing := result[k]
			existing.VmCount += v.VmCount
			existing.TotalSizeTB += v.TotalSizeTB
			result[k] = existing
		}
	}
	return &result
}

func mergeDiskTypeMap(a, b *map[string]v1alpha1.DiskTypeSummary) *map[string]v1alpha1.DiskTypeSummary {
	if a == nil && b == nil {
		return nil
	}
	result := make(map[string]v1alpha1.DiskTypeSummary)
	if a != nil {
		for k, v := range *a {
			result[k] = v
		}
	}
	if b != nil {
		for k, v := range *b {
			existing := result[k]
			existing.VmCount += v.VmCount
			existing.TotalSizeTB += v.TotalSizeTB
			result[k] = existing
		}
	}
	return &result
}

func mergeOsInfoMap(a, b *map[string]v1alpha1.OsInfo) *map[string]v1alpha1.OsInfo {
	if a == nil && b == nil {
		return nil
	}
	result := make(map[string]v1alpha1.OsInfo)
	if a != nil {
		for k, v := range *a {
			result[k] = v
		}
	}
	if b != nil {
		for k, v := range *b {
			existing := result[k]
			existing.Count += v.Count
			if v.Supported {
				existing.Supported = true
			}
			if existing.UpgradeRecommendation == nil {
				existing.UpgradeRecommendation = v.UpgradeRecommendation
			}
			result[k] = existing
		}
	}
	return &result
}

func mergeMigrationIssues(a, b []v1alpha1.MigrationIssue) []v1alpha1.MigrationIssue {
	if a == nil && b == nil {
		return nil
	}
	index := make(map[string]int)
	var result []v1alpha1.MigrationIssue
	for _, issue := range a {
		key := issue.Label + "|" + issue.Assessment
		index[key] = len(result)
		result = append(result, issue)
	}
	for _, issue := range b {
		key := issue.Label + "|" + issue.Assessment
		if idx, ok := index[key]; ok {
			result[idx].Count += issue.Count
		} else {
			index[key] = len(result)
			result = append(result, issue)
		}
	}
	return result
}

func sumPtrInt(a, b *int) *int {
	if a == nil && b == nil {
		return nil
	}
	v := ptrIntVal(a) + ptrIntVal(b)
	return &v
}

func ptrIntVal(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func ptrBreakdownVal(p *v1alpha1.VMResourceBreakdown) v1alpha1.VMResourceBreakdown {
	if p == nil {
		return v1alpha1.VMResourceBreakdown{}
	}
	return *p
}

func ptrSlice[T any](p *[]T) []T {
	if p == nil {
		return nil
	}
	return *p
}
