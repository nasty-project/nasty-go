// Package nastygo provides a WebSocket client for the NASty storage API.
package nastygo

import (
	"context"
)

// ClientInterface defines the interface for NASty API operations.
// This allows for dependency injection and easier testing.
//
//nolint:interfacebloat // NASty API client naturally has many methods covering different resource types
type ClientInterface interface {
	// Filesystem
	QueryFilesystem(ctx context.Context, fsName string) (*Filesystem, error)

	// Subvolumes
	CreateSubvolume(ctx context.Context, params SubvolumeCreateParams) (*Subvolume, error)
	DeleteSubvolume(ctx context.Context, filesystem, name string) error
	GetSubvolume(ctx context.Context, filesystem, name string) (*Subvolume, error)
	ListAllSubvolumes(ctx context.Context, filesystem string) ([]Subvolume, error)
	ResizeSubvolume(ctx context.Context, filesystem, name string, volsizeBytes uint64) (*Subvolume, error)
	CloneSubvolume(ctx context.Context, filesystem, name, newName string) (*Subvolume, error)

	// Properties (xattrs)
	SetSubvolumeProperties(ctx context.Context, filesystem, name string, props map[string]string) (*Subvolume, error)
	RemoveSubvolumeProperties(ctx context.Context, filesystem, name string, keys []string) (*Subvolume, error)
	FindSubvolumesByProperty(ctx context.Context, key, value, filesystem string) ([]Subvolume, error)
	FindManagedSubvolumes(ctx context.Context, filesystem string) ([]Subvolume, error)
	FindSubvolumeByCSIVolumeName(ctx context.Context, filesystem, volumeName string) (*Subvolume, error)

	// Snapshots
	CreateSnapshot(ctx context.Context, params SnapshotCreateParams) (*Snapshot, error)
	DeleteSnapshot(ctx context.Context, filesystem, subvolume, name string) error
	ListSnapshots(ctx context.Context, filesystem string) ([]Snapshot, error)
	CloneSnapshot(ctx context.Context, params SnapshotCloneParams) (*Subvolume, error)

	// NFS
	CreateNFSShare(ctx context.Context, params NFSShareCreateParams) (*NFSShare, error)
	DeleteNFSShare(ctx context.Context, id string) error
	ListNFSShares(ctx context.Context) ([]NFSShare, error)
	GetNFSShare(ctx context.Context, id string) (*NFSShare, error)

	// SMB
	CreateSMBShare(ctx context.Context, params SMBShareCreateParams) (*SMBShare, error)
	DeleteSMBShare(ctx context.Context, id string) error
	ListSMBShares(ctx context.Context) ([]SMBShare, error)
	GetSMBShare(ctx context.Context, id string) (*SMBShare, error)

	// iSCSI
	CreateISCSITarget(ctx context.Context, params ISCSITargetCreateParams) (*ISCSITarget, error)
	AddISCSILun(ctx context.Context, targetID, backstorePath string) (*ISCSITarget, error)
	AddISCSIACL(ctx context.Context, targetID, initiatorIQN string) (*ISCSITarget, error)
	DeleteISCSITarget(ctx context.Context, id string) error
	ListISCSITargets(ctx context.Context) ([]ISCSITarget, error)
	GetISCSITargetByIQN(ctx context.Context, iqn string) (*ISCSITarget, error)

	// NVMe-oF
	CreateNVMeOFSubsystem(ctx context.Context, params NVMeOFCreateParams) (*NVMeOFSubsystem, error)
	DeleteNVMeOFSubsystem(ctx context.Context, id string) error
	ListNVMeOFSubsystems(ctx context.Context) ([]NVMeOFSubsystem, error)
	GetNVMeOFSubsystemByNQN(ctx context.Context, nqn string) (*NVMeOFSubsystem, error)

	// Connection
	Close()
	IsConnected() bool
}

// Filesystem represents a NASty storage filesystem.
type Filesystem struct {
	Name           string            `json:"name"`
	Mounted        bool              `json:"mounted"`
	MountPoint     *string           `json:"mount_point"`
	TotalBytes     uint64            `json:"total_bytes"`
	UsedBytes      uint64            `json:"used_bytes"`
	AvailableBytes uint64            `json:"available_bytes"`
	Options        FilesystemOptions `json:"options"`
}

// FilesystemOptions contains filesystem-level settings (bcachefs).
type FilesystemOptions struct {
	Encrypted *bool `json:"encrypted"`
}

// Subvolume represents a NASty subvolume (filesystem or block device).
type Subvolume struct {
	Name          string            `json:"name"`
	Filesystem    string            `json:"filesystem"`
	SubvolumeType string            `json:"subvolume_type"` // "filesystem" or "block"
	Path          string            `json:"path"`
	UsedBytes     *uint64           `json:"used_bytes"`
	Compression   *string           `json:"compression"`
	Comments      *string           `json:"comments"`
	VolsizeBytes  *uint64           `json:"volsize_bytes"`
	BlockDevice   *string           `json:"block_device"`
	Snapshots     []string          `json:"snapshots"`
	Owner         *string           `json:"owner"`
	Properties    map[string]string `json:"properties"`
}

// SubvolumeCreateParams holds parameters for subvolume creation.
type SubvolumeCreateParams struct {
	Filesystem       string  `json:"filesystem"`
	Name             string  `json:"name"`
	SubvolumeType    string  `json:"subvolume_type"`
	VolsizeBytes     *uint64 `json:"volsize_bytes,omitempty"`
	Compression      string  `json:"compression,omitempty"`
	Comments         string  `json:"comments,omitempty"`
	ForegroundTarget string  `json:"foreground_target,omitempty"`
	BackgroundTarget string  `json:"background_target,omitempty"`
	PromoteTarget    string  `json:"promote_target,omitempty"`
}

// Snapshot represents a NASty snapshot.
type Snapshot struct {
	Name       string `json:"name"`
	Subvolume  string `json:"subvolume"`
	Filesystem string `json:"filesystem"`
	Path       string `json:"path"`
	ReadOnly   bool   `json:"read_only"`
	Parent     string `json:"parent,omitempty"`
}

// SnapshotCreateParams holds parameters for snapshot creation.
type SnapshotCreateParams struct {
	Filesystem string `json:"filesystem"`
	Subvolume  string `json:"subvolume"`
	Name       string `json:"name"`
	ReadOnly   bool   `json:"read_only,omitempty"`
}

// SnapshotCloneParams holds parameters for cloning a snapshot into a new writable subvolume.
type SnapshotCloneParams struct {
	Filesystem string `json:"filesystem"`
	Subvolume  string `json:"subvolume"`
	Snapshot   string `json:"snapshot"`
	NewName    string `json:"new_name"`
}

// NFSShare represents a NASty NFS share.
type NFSShare struct {
	ID      string      `json:"id"`
	Path    string      `json:"path"`
	Comment *string     `json:"comment"`
	Clients []NFSClient `json:"clients"`
	Enabled bool        `json:"enabled"`
}

// NFSClient represents a single NFS client entry with host and options.
type NFSClient struct {
	Host    string `json:"host"`
	Options string `json:"options"`
}

// NFSShareCreateParams holds parameters for NFS share creation.
type NFSShareCreateParams struct {
	Path    string      `json:"path"`
	Comment string      `json:"comment,omitempty"`
	Clients []NFSClient `json:"clients"`
	Enabled *bool       `json:"enabled,omitempty"`
}

// SMBShare represents a NASty SMB share.
type SMBShare struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Path    string  `json:"path"`
	Comment *string `json:"comment"`
	Enabled bool    `json:"enabled"`
}

// SMBShareCreateParams holds parameters for SMB share creation.
type SMBShareCreateParams struct {
	Name       string   `json:"name"`
	Path       string   `json:"path"`
	Comment    string   `json:"comment,omitempty"`
	ValidUsers []string `json:"valid_users,omitempty"`
}

// ISCSIPortal represents an iSCSI portal.
type ISCSIPortal struct {
	IP   string `json:"ip"`
	Port uint16 `json:"port"`
}

// ISCSILun represents a LUN attached to an iSCSI target.
type ISCSILun struct {
	LunID         uint32  `json:"lun_id"`
	BackstorePath string  `json:"backstore_path"`
	BackstoreName string  `json:"backstore_name"`
	BackstoreType string  `json:"backstore_type"`
	SizeBytes     *uint64 `json:"size_bytes"`
}

// ISCSITarget represents a NASty iSCSI target.
type ISCSITarget struct {
	ID      string        `json:"id"`
	IQN     string        `json:"iqn"`
	Portals []ISCSIPortal `json:"portals"`
	Luns    []ISCSILun    `json:"luns"`
	Enabled bool          `json:"enabled"`
}

// ISCSITargetCreateParams holds parameters for iSCSI target creation.
type ISCSITargetCreateParams struct {
	Name       string   `json:"name"`
	DevicePath string   `json:"device_path,omitempty"`
	Acls       []ACLEntry `json:"acls,omitempty"`
}

// ACLEntry represents an initiator ACL for iSCSI target creation.
type ACLEntry struct {
	InitiatorIQN string `json:"initiator_iqn"`
	UserID       string `json:"userid,omitempty"`
	Password     string `json:"password,omitempty"`
}

// NVMeOFNamespace represents a namespace within an NVMe-oF subsystem.
type NVMeOFNamespace struct {
	NSID       uint32 `json:"nsid"`
	DevicePath string `json:"device_path"`
	Enabled    bool   `json:"enabled"`
}

// NVMeOFPort represents a port on an NVMe-oF subsystem.
type NVMeOFPort struct {
	PortID    uint16 `json:"port_id"`
	Transport string `json:"transport"`
	Addr      string `json:"addr"`
	ServiceID string `json:"service_id"`
}

// NVMeOFSubsystem represents a NASty NVMe-oF subsystem.
type NVMeOFSubsystem struct {
	ID           string            `json:"id"`
	NQN          string            `json:"nqn"`
	Namespaces   []NVMeOFNamespace `json:"namespaces"`
	Ports        []NVMeOFPort      `json:"ports"`
	AllowedHosts []string          `json:"allowed_hosts"`
	AllowAnyHost bool              `json:"allow_any_host"`
	Enabled      bool              `json:"enabled"`
}

// NVMeOFCreateParams holds parameters for NVMe-oF subsystem creation.
type NVMeOFCreateParams struct {
	Name         string   `json:"name"`
	DevicePath   string   `json:"device_path,omitempty"`
	Addr         string   `json:"addr,omitempty"`
	Port         *uint16  `json:"port,omitempty"`
	AllowedHosts []string `json:"allowed_hosts,omitempty"`
}

// Verify that Client implements ClientInterface at compile time.
var _ ClientInterface = (*Client)(nil)
