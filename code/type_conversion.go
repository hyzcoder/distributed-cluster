package code

import (
	"encoding/binary"
)

//func float64ToBytes(f float64) []byte {
//	bits := math.Float64bits(f)
//	bytes := make([]byte,8)
//	binary.LittleEndian.PutUint64(bytes,bits)
//	return bytes
//}
//
//func bytesToFloat64(bytes []byte) float64 {
//	bits := binary.LittleEndian.Uint64(bytes)
//	return math.Float64frombits(bits)
//}

func Int64ToBytes(i int64) []byte {
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(i))
	return buf
}

func BytesToInt64(buf []byte) int64 {
	return int64(binary.BigEndian.Uint64(buf))
}