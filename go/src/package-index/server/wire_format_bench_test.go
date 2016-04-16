package server

import "testing"

// Example (Macbook Pro, 2.3 GHz Intel Core i7, 16 GB 1600 MHz DDR3):
//
// BenchmarkParseMessageNoDeps  20000000         74.8 ns/op         2 B/op           2 allocs/op
// BenchmarkParseMessage10Deps  1000000          1297 ns/op         351 B/op         14 allocs/op

func BenchmarkParseMessageNoDeps(b *testing.B) {
	raw := []byte("A|B|\n")
	for i := 0; i < b.N; i++ {
		parseMessage(raw)
	}
}

func BenchmarkParseMessage10Deps(b *testing.B) {
	raw := []byte("A|B|C,D,E,F,G,H,I,J,K,L\n")
	for i := 0; i < b.N; i++ {
		parseMessage(raw)
	}
}
