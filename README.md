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

## Retry safety

The client reconnects automatically, but only explicitly allowlisted read-only
RPCs are replayed after a lost response. Requests not yet attempted may be retried.
Mutations and unknown methods are **not** assumed idempotent, including updates
and deletes.

If a mutation may have reached the server but its response is lost, times out,
or cannot be decoded, its error wraps `nastygo.ErrUncertainOutcome`. Check with
`errors.Is(err, nastygo.ErrUncertainOutcome)` and reconcile server state before
retrying (for example, look up the requested subvolume by its stable identity).
This error does not mean the operation failed or was rolled back. It also wraps
the underlying cause, including context cancellation/deadline errors.

A definitive API error is returned unchanged. Cancellation before attempting a
request does not produce an uncertain-outcome error. Error messages do not include
request parameters, results, or authentication tokens.

## Structure

- `client.go` — WebSocket client with reconnection and JSON-RPC transport
- `interface.go` — API types and client interface definition
- `properties.go` — CSI xattr property keys and helpers
- `dashboard/` — In-cluster CSI dashboard data types
