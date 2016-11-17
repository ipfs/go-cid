// +build gofuzz

package cid

func Fuzz(data []byte) int {
	cid, err := Cast(data)

	if err != nil {
		return 0
	}

	_ = cid.Bytes()
	if !cid.Equals(cid) {
		panic("inequality")
	}
	return 1
}
