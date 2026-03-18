package dashboard

// CalculateSummary computes summary statistics from volumes, snapshots, and clones.
func CalculateSummary(volumes []VolumeInfo, snapshots []SnapshotInfo, clones []CloneInfo) SummaryData {
	summary := SummaryData{
		TotalVolumes:   len(volumes),
		TotalSnapshots: len(snapshots),
		TotalClones:    len(clones),
	}

	var totalBytes int64
	for i := range volumes {
		switch volumes[i].Protocol {
		case ProtocolNFS:
			summary.NFSVolumes++
		case ProtocolNVMeOF:
			summary.NVMeOFVolumes++
		case ProtocolISCSI:
			summary.ISCSIVolumes++
		case ProtocolSMB:
			summary.SMBVolumes++
		}
		totalBytes += volumes[i].CapacityBytes
		if volumes[i].HealthStatus != "" && volumes[i].HealthStatus != string(HealthStatusHealthy) {
			summary.UnhealthyVolumes++
		} else {
			summary.HealthyVolumes++
		}
	}

	summary.CapacityBytes = totalBytes
	summary.TotalCapacity = FormatBytes(totalBytes)

	return summary
}

// MatchK8sBinding tries to find a K8s binding by dataset path first (new volumes),
// then falls back to csi_volume_name (old volumes).
func MatchK8sBinding(bindings map[string]*K8sVolumeBinding, dataset, volumeID string) *K8sVolumeBinding {
	if b, ok := bindings[dataset]; ok {
		return b
	}
	if volumeID != "" && volumeID != dataset {
		if b, ok := bindings[volumeID]; ok {
			return b
		}
	}
	return nil
}
