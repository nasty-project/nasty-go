package dashboard

import (
	"context"
	"strings"

	nastygo "github.com/nasty-project/nasty-go"
	"k8s.io/klog/v2"
)

// HealthResourceMaps holds bulk-queried resource maps for health checking.
type HealthResourceMaps struct {
	nfsShareMap    map[string]*nastygo.NFSShare
	nvmeSubsysMap  map[string]*nastygo.NVMeOFSubsystem
	smbShareMap    map[string]*nastygo.SMBShare
	iscsiTargetMap map[string]*nastygo.ISCSITarget
}

// buildHealthResourceMaps queries all protocol resources in bulk for health checking.
func buildHealthResourceMaps(ctx context.Context, client nastygo.ClientInterface) *HealthResourceMaps {
	m := &HealthResourceMaps{
		nfsShareMap:    make(map[string]*nastygo.NFSShare),
		nvmeSubsysMap:  make(map[string]*nastygo.NVMeOFSubsystem),
		smbShareMap:    make(map[string]*nastygo.SMBShare),
		iscsiTargetMap: make(map[string]*nastygo.ISCSITarget),
	}

	nfsShares, err := client.ListNFSShares(ctx)
	if err == nil {
		for i := range nfsShares {
			m.nfsShareMap[nfsShares[i].Path] = &nfsShares[i]
		}
	}

	nvmeSubsystems, err := client.ListNVMeOFSubsystems(ctx)
	if err == nil {
		for i := range nvmeSubsystems {
			m.nvmeSubsysMap[nvmeSubsystems[i].NQN] = &nvmeSubsystems[i]
		}
	}

	smbShares, err := client.ListSMBShares(ctx)
	if err == nil {
		for i := range smbShares {
			m.smbShareMap[smbShares[i].Path] = &smbShares[i]
		}
	}

	iscsiTargets, err := client.ListISCSITargets(ctx)
	if err == nil {
		for i := range iscsiTargets {
			m.iscsiTargetMap[iscsiTargets[i].IQN] = &iscsiTargets[i]
		}
	}

	return m
}

// CheckVolumeHealth checks the health of all managed volumes.
func CheckVolumeHealth(ctx context.Context, client nastygo.ClientInterface) (*HealthReport, error) {
	subvols, err := client.FindSubvolumesByProperty(ctx, nastygo.PropertyManagedBy, nastygo.ManagedByValue, "")
	if err != nil {
		return nil, err
	}

	resources := buildHealthResourceMaps(ctx, client)

	report := &HealthReport{
		Volumes:  make([]VolumeHealth, 0),
		Problems: make([]VolumeHealth, 0),
	}

	for i := range subvols {
		sv := &subvols[i]

		volumeID := sv.Properties[nastygo.PropertyCSIVolumeName]
		if volumeID == "" {
			continue
		}

		health := VolumeHealth{
			VolumeID:  volumeID,
			Dataset:   sv.Pool + "/" + sv.Name,
			DatasetOK: true,
			Status:    HealthStatusHealthy,
			Issues:    make([]string, 0),
		}

		health.Protocol = sv.Properties[nastygo.PropertyProtocol]

		switch health.Protocol {
		case ProtocolNFS:
			CheckNFSHealth(sv, resources.nfsShareMap, &health)
		case ProtocolNVMeOF:
			CheckNVMeOFHealth(sv, resources.nvmeSubsysMap, &health)
		case ProtocolSMB:
			CheckSMBHealth(sv, resources.smbShareMap, &health)
		case ProtocolISCSI:
			CheckISCSIHealth(sv, resources.iscsiTargetMap, &health)
		}

		if len(health.Issues) > 0 {
			health.Status = HealthStatusDegraded
			for _, issue := range health.Issues {
				issueLower := strings.ToLower(issue)
				if strings.Contains(issueLower, "not found") || strings.Contains(issueLower, "disabled") {
					health.Status = HealthStatusUnhealthy
					break
				}
			}
		}

		report.Summary.TotalVolumes++
		switch health.Status {
		case HealthStatusHealthy:
			report.Summary.HealthyVolumes++
		case HealthStatusDegraded:
			report.Summary.DegradedVolumes++
		case HealthStatusUnhealthy:
			report.Summary.UnhealthyVolumes++
		}

		report.Volumes = append(report.Volumes, health)
		if health.Status != HealthStatusHealthy {
			report.Problems = append(report.Problems, health)
		}
	}

	return report, nil
}

// CheckNFSHealth checks if the NFS share for a subvolume is healthy.
func CheckNFSHealth(sv *nastygo.Subvolume, nfsShareMap map[string]*nastygo.NFSShare, health *VolumeHealth) {
	sharePath := sv.Path
	if sharePath == "" {
		health.Issues = append(health.Issues, "NFS share path not found in properties")
		shareOK := false
		health.ShareOK = &shareOK
		return
	}

	share, exists := nfsShareMap[sharePath]
	if !exists {
		health.Issues = append(health.Issues, "NFS share not found for path "+sharePath)
		shareOK := false
		health.ShareOK = &shareOK
		return
	}

	shareOK := true
	if !share.Enabled {
		health.Issues = append(health.Issues, "NFS share is disabled")
		shareOK = false
	}
	health.ShareOK = &shareOK
}

// CheckNVMeOFHealth checks if the NVMe-oF subsystem for a subvolume is healthy.
// Subsystem is found by scanning the NQN map for a suffix matching the volume name.
func CheckNVMeOFHealth(sv *nastygo.Subvolume, nvmeSubsysMap map[string]*nastygo.NVMeOFSubsystem, health *VolumeHealth) {
	volumeName := sv.Properties[nastygo.PropertyCSIVolumeName]
	if volumeName == "" {
		health.Issues = append(health.Issues, "CSI volume name not found in properties")
		subsysOK := false
		health.SubsysOK = &subsysOK
		return
	}

	// Scan subsystems by NQN suffix matching volume name
	suffix := ":" + volumeName
	found := false
	for nqn := range nvmeSubsysMap {
		if strings.HasSuffix(nqn, suffix) {
			found = true
			break
		}
	}

	if !found {
		health.Issues = append(health.Issues, "NVMe-oF subsystem not found for volume "+volumeName)
		subsysOK := false
		health.SubsysOK = &subsysOK
		return
	}

	subsysOK := true
	health.SubsysOK = &subsysOK
}

// CheckSMBHealth checks if the SMB share for a subvolume is healthy.
func CheckSMBHealth(sv *nastygo.Subvolume, smbShareMap map[string]*nastygo.SMBShare, health *VolumeHealth) {
	sharePath := sv.Path

	if sharePath == "" {
		health.Issues = append(health.Issues, "SMB share path not found")
		smbOK := false
		health.SMBShareOK = &smbOK
		return
	}

	share, exists := smbShareMap[sharePath]
	if !exists {
		health.Issues = append(health.Issues, "SMB share not found for path "+sharePath)
		smbOK := false
		health.SMBShareOK = &smbOK
		return
	}

	smbOK := true
	if !share.Enabled {
		health.Issues = append(health.Issues, "SMB share is disabled")
		smbOK = false
	}
	health.SMBShareOK = &smbOK
}

// CheckISCSIHealth checks if the iSCSI target for a subvolume is healthy.
// Target is found by scanning the IQN map for a suffix matching the volume name.
func CheckISCSIHealth(sv *nastygo.Subvolume, iscsiTargetMap map[string]*nastygo.ISCSITarget, health *VolumeHealth) {
	volumeName := sv.Properties[nastygo.PropertyCSIVolumeName]
	if volumeName == "" {
		health.Issues = append(health.Issues, "CSI volume name not found in properties")
		targetOK := false
		health.TargetOK = &targetOK
		return
	}

	// Scan targets by IQN suffix matching volume name
	suffix := ":" + volumeName
	found := false
	for iqn := range iscsiTargetMap {
		if strings.HasSuffix(iqn, suffix) {
			found = true
			break
		}
	}

	if !found {
		health.Issues = append(health.Issues, "iSCSI target not found for volume "+volumeName)
		targetOK := false
		health.TargetOK = &targetOK
		return
	}

	targetOK := true
	health.TargetOK = &targetOK
}

// BuildHealthMapsFromData builds health resource maps from pre-fetched data (no API calls).
func BuildHealthMapsFromData(
	nfsShares []nastygo.NFSShare,
	smbShares []nastygo.SMBShare,
	nvmeSubsystems []nastygo.NVMeOFSubsystem,
	iscsiTargets []nastygo.ISCSITarget,
) *HealthResourceMaps {
	m := &HealthResourceMaps{
		nfsShareMap:    make(map[string]*nastygo.NFSShare, len(nfsShares)),
		nvmeSubsysMap:  make(map[string]*nastygo.NVMeOFSubsystem, len(nvmeSubsystems)),
		smbShareMap:    make(map[string]*nastygo.SMBShare, len(smbShares)),
		iscsiTargetMap: make(map[string]*nastygo.ISCSITarget, len(iscsiTargets)),
	}

	for i := range nfsShares {
		m.nfsShareMap[nfsShares[i].Path] = &nfsShares[i]
	}
	for i := range nvmeSubsystems {
		m.nvmeSubsysMap[nvmeSubsystems[i].NQN] = &nvmeSubsystems[i]
	}
	for i := range smbShares {
		m.smbShareMap[smbShares[i].Path] = &smbShares[i]
	}
	for i := range iscsiTargets {
		m.iscsiTargetMap[iscsiTargets[i].IQN] = &iscsiTargets[i]
	}

	return m
}

// AnnotateHealthFromMaps annotates volumes with health status using pre-fetched resource maps and subvolumes.
func AnnotateHealthFromMaps(volumes []VolumeInfo, managedSubvols []nastygo.Subvolume, resources *HealthResourceMaps) {
	subvolMap := make(map[string]*nastygo.Subvolume, len(managedSubvols))
	for i := range managedSubvols {
		sv := &managedSubvols[i]
		volumeID := sv.Properties[nastygo.PropertyCSIVolumeName]
		if volumeID != "" {
			subvolMap[volumeID] = sv
		}
	}

	for i := range volumes {
		sv, ok := subvolMap[volumes[i].VolumeID]
		if !ok {
			continue
		}

		health := VolumeHealth{
			VolumeID:  volumes[i].VolumeID,
			Dataset:   sv.Pool + "/" + sv.Name,
			DatasetOK: true,
			Status:    HealthStatusHealthy,
			Issues:    make([]string, 0),
		}

		protocol := sv.Properties[nastygo.PropertyProtocol]

		switch protocol {
		case ProtocolNFS:
			CheckNFSHealth(sv, resources.nfsShareMap, &health)
		case ProtocolNVMeOF:
			CheckNVMeOFHealth(sv, resources.nvmeSubsysMap, &health)
		case ProtocolSMB:
			CheckSMBHealth(sv, resources.smbShareMap, &health)
		case ProtocolISCSI:
			CheckISCSIHealth(sv, resources.iscsiTargetMap, &health)
		}

		if len(health.Issues) > 0 {
			health.Status = HealthStatusDegraded
			for _, issue := range health.Issues {
				issueLower := strings.ToLower(issue)
				if strings.Contains(issueLower, "not found") || strings.Contains(issueLower, "disabled") {
					health.Status = HealthStatusUnhealthy
					break
				}
			}
		}

		volumes[i].HealthStatus = string(health.Status)
		if len(health.Issues) > 0 {
			volumes[i].HealthIssue = health.Issues[0]
		}
	}
}

// AnnotateVolumesWithHealth runs health checks and annotates VolumeInfo slices with health status.
func AnnotateVolumesWithHealth(ctx context.Context, client nastygo.ClientInterface, volumes []VolumeInfo) {
	healthReport, err := CheckVolumeHealth(ctx, client)
	if err != nil {
		klog.Warningf("Failed to check volume health: %v", err)
		return
	}

	healthMap := make(map[string]*VolumeHealth, len(healthReport.Volumes))
	for i := range healthReport.Volumes {
		healthMap[healthReport.Volumes[i].VolumeID] = &healthReport.Volumes[i]
	}

	for i := range volumes {
		if h, ok := healthMap[volumes[i].VolumeID]; ok {
			volumes[i].HealthStatus = string(h.Status)
			if len(h.Issues) > 0 {
				volumes[i].HealthIssue = h.Issues[0]
			}
		}
	}
}
