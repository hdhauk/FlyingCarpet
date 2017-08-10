package main

import "testing"

const testFile = "./bin/flyingcarpet"

func BenchmarkGetHashSHA256(b *testing.B) {
	for n := 0; n < b.N; n++ {
		getHashSHA256(testFile)
	}
}

func BenchmarkGetHashMD5(b *testing.B) {
	for n := 0; n < b.N; n++ {
		getHash(testFile)
	}
}
