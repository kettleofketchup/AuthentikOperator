package hash

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
)

// ComputeSecretHash computes a deterministic SHA256 hash of a string map.
// Keys are sorted to ensure order-independence.
func ComputeSecretHash(data map[string]string) string {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(data[k])
		b.WriteString("\n")
	}

	sum := sha256.Sum256([]byte(b.String()))
	return fmt.Sprintf("sha256:%x", sum)
}
