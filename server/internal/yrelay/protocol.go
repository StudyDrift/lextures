// Package yrelay provides shared Y.js WebSocket sync helpers used by
// collaborative documents and collaboration boards.
package yrelay

import "bytes"

// writeVarUint encodes a uint64 as a lib0 variable-length integer.
func writeVarUint(buf *bytes.Buffer, n uint64) {
	for n >= 0x80 {
		buf.WriteByte(byte(n&0x7F) | 0x80)
		n >>= 7
	}
	buf.WriteByte(byte(n))
}

// EncodeSyncUpdate wraps a Y.js update as a sync update message [0, 2, len, ...data].
func EncodeSyncUpdate(data []byte) []byte {
	var buf bytes.Buffer
	buf.WriteByte(0) // messageSync
	buf.WriteByte(2) // messageYjsUpdate
	writeVarUint(&buf, uint64(len(data)))
	buf.Write(data)
	return buf.Bytes()
}

// EncodeEmptySyncStep1 returns [0, 0, 0] = msgSync, syncStep1, empty state vector.
func EncodeEmptySyncStep1() []byte {
	return []byte{0, 0, 0}
}

// EncodeEmptySyncStep2 returns [0, 1, 0] = msgSync, syncStep2, empty update.
func EncodeEmptySyncStep2() []byte {
	return []byte{0, 1, 0}
}

// ExtractUpdateFromMsg extracts the raw update bytes from a sync step 2 or update message.
// msg format: [0, subType, varintLen, ...updateBytes]
func ExtractUpdateFromMsg(msg []byte) []byte {
	if len(msg) < 3 {
		return nil
	}
	r := bytes.NewReader(msg[2:])
	var n uint64
	var shift uint
	for {
		b, err := r.ReadByte()
		if err != nil {
			return nil
		}
		n |= uint64(b&0x7F) << shift
		if b < 0x80 {
			break
		}
		shift += 7
		if shift > 63 {
			return nil
		}
	}
	out := make([]byte, n)
	if _, err := r.Read(out); err != nil && n > 0 {
		return nil
	}
	return out
}
