package hash

import (
	"testing"
)

func TestComputeSecretHash(t *testing.T) {
	data := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	hash1 := ComputeSecretHash(data)
	if hash1 == "" {
		t.Fatal("hash should not be empty")
	}

	hash2 := ComputeSecretHash(data)
	if hash1 != hash2 {
		t.Error("same data should produce same hash")
	}

	data["key3"] = "value3"
	hash3 := ComputeSecretHash(data)
	if hash1 == hash3 {
		t.Error("different data should produce different hash")
	}
}

func TestComputeSecretHash_OrderIndependent(t *testing.T) {
	data1 := map[string]string{"a": "1", "b": "2", "c": "3"}
	data2 := map[string]string{"c": "3", "a": "1", "b": "2"}

	if ComputeSecretHash(data1) != ComputeSecretHash(data2) {
		t.Error("hash should be order-independent")
	}
}

func TestComputeSecretHash_EmptyMap(t *testing.T) {
	h1 := ComputeSecretHash(nil)
	h2 := ComputeSecretHash(map[string]string{})
	if h1 == "" || h2 == "" {
		t.Error("hash of empty data should not be empty")
	}
	if h1 != h2 {
		t.Error("nil map and empty map should produce same hash")
	}
}
