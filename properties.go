// Package nastygo provides a WebSocket client for the NASty storage API.
package nastygo

import "strconv"

// Xattr property keys stored on bcachefs subvolumes to track CSI metadata.
// All use the "nasty-csi:" prefix. Viewable via `getfattr -d <path>` on NASty.
const (
	PropertyPrefix        = "nasty-csi:"
	PropertyManagedBy     = "nasty-csi:managed_by"
	PropertyCSIVolumeName = "nasty-csi:csi_volume_name"
	PropertyCapacityBytes = "nasty-csi:capacity_bytes"
	PropertyProtocol      = "nasty-csi:protocol"
	PropertyDeleteStrategy = "nasty-csi:delete_strategy"
	PropertyCreatedAt     = "nasty-csi:created_at"

	// Adoption — cross-cluster volume migration.
	PropertyAdoptable    = "nasty-csi:adoptable"
	PropertyPVCName      = "nasty-csi:pvc_name"
	PropertyPVCNamespace = "nasty-csi:pvc_namespace"
	PropertyStorageClass = "nasty-csi:storage_class"

	// Multi-cluster isolation.
	PropertyClusterID = "nasty-csi:cluster_id"
)

// Property values.
const (
	ManagedByValue    = "nasty-csi"
	PropertyValueTrue = "true"

	ProtocolNFS    = "nfs"
	ProtocolNVMeOF = "nvmeof"
	ProtocolISCSI  = "iscsi"
	ProtocolSMB    = "smb"

	DeleteStrategyDelete = "delete"
	DeleteStrategyRetain = "retain"
)

// VolumeParams contains parameters for writing CSI metadata to a subvolume.
type VolumeParams struct {
	VolumeID       string
	Protocol       string
	CreatedAt      string
	DeleteStrategy string
	CapacityBytes  int64
	PVCName        string
	PVCNamespace   string
	StorageClass   string
	ClusterID      string
	Adoptable      bool
}

// VolumeProperties returns the xattr properties map for a CSI volume.
func VolumeProperties(p VolumeParams) map[string]string {
	props := map[string]string{
		PropertyManagedBy:      ManagedByValue,
		PropertyCSIVolumeName:  p.VolumeID,
		PropertyCapacityBytes:  strconv.FormatInt(p.CapacityBytes, 10),
		PropertyProtocol:       p.Protocol,
		PropertyCreatedAt:      p.CreatedAt,
		PropertyDeleteStrategy: p.DeleteStrategy,
	}
	if p.PVCName != "" {
		props[PropertyPVCName] = p.PVCName
	}
	if p.PVCNamespace != "" {
		props[PropertyPVCNamespace] = p.PVCNamespace
	}
	if p.StorageClass != "" {
		props[PropertyStorageClass] = p.StorageClass
	}
	if p.Adoptable {
		props[PropertyAdoptable] = "true"
	}
	if p.ClusterID != "" {
		props[PropertyClusterID] = p.ClusterID
	}
	return props
}

// StringToInt64 converts a string to int64, returns 0 on error.
func StringToInt64(s string) int64 {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return i
}
