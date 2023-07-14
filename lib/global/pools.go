package global

import (
	"sync"

	"pggat2/lib/util/pools"
)

// i really don't know how I feel about these global pools
var (
	bytesPool pools.Log2[byte]
	bytesMu   sync.Mutex
)

func GetBytes(length int32) []byte {
	bytesMu.Lock()
	defer bytesMu.Unlock()
	return bytesPool.Get(length)
}

func PutBytes(v []byte) {
	bytesMu.Lock()
	defer bytesMu.Unlock()
	bytesPool.Put(v)
}
