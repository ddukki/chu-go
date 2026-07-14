package conn

import (
	"fmt"

	"github.com/ClickHouse/ch-go/proto"
)

// decodeBlockInfoSafe decodes BlockInfo, handling fields added in later
// ClickHouse revisions that ch-go v0.73.0's BlockInfo.Decode rejects.
//
// Known fields:
//   1 — is_overflows (bool), 1 byte
//   2 — bucket_num (Int32), 4-bytes LE
//   3 — out_of_order_buckets (vector<Int32>, rev 54480+): VarInt(length) + N × 4-bytes LE
//   0 — end marker
func decodeBlockInfoSafe(r *proto.Reader) error {
	for {
		f, err := r.UVarInt()
		if err != nil {
			return err
		}
		switch f {
		case 0:
			return nil
		case 1:
			if _, err := r.Bool(); err != nil {
				return err
			}
		case 2:
			if _, err := r.Int32(); err != nil {
				return err
			}
		case 3:
			n, err := r.UVarInt()
			if err != nil {
				return err
			}
					for i := uint64(0); i < n; i++ {
				if _, err := r.Int32(); err != nil {
					return err
				}
			}
		default:
			return &Error{
				Kind:    KindProtocol,
				Message: "decode block info",
				Err:     fmt.Errorf("unknown BlockInfo field %d", f),
			}
		}
	}
}
