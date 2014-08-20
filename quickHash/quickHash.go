package quickHash

import (
	"encoding/hex"
	"hash/fnv"
)

//fnv hashm quicker and fewer collisions
//in case of a collision, the url is overriden, resulting in 1 potential miss match in search result, which is minor and neglectable
func Hash(str string) string {
	hasher := fnv.New64a()
	hasher.Write([]byte(str))
	return hex.EncodeToString(hasher.Sum(nil))
}
