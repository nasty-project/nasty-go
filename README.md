# nasty-go

Go client library for the [NASty](https://github.com/nasty-project/nasty) storage API. Communicates over WebSocket using JSON-RPC 2.0.

Used by the [NASty CSI driver](https://github.com/nasty-project/nasty-csi) to manage filesystems, subvolumes, snapshots, and sharing protocols (NFS, SMB, iSCSI, NVMe-oF) on a NASty appliance.

## Usage

```go
import nastygo "github.com/nasty-project/nasty-go"

client, err := nastygo.NewClient("wss://nasty.local/ws", "your-api-key", true)
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// Create a subvolume
subvol, err := client.CreateSubvolume(ctx, nastygo.SubvolumeCreateParams{
    Filesystem:    "storage",
    Name:          "my-volume",
    SubvolumeType: "filesystem",
})
```

## Structure

- `client.go` — WebSocket client with reconnection and JSON-RPC transport
- `interface.go` — API types and client interface definition
- `properties.go` — CSI xattr property keys and helpers
- `dashboard/` — In-cluster CSI dashboard data types
