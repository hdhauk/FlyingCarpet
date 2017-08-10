package main

import (
	"crypto/rand"
	"io"

	"golang.org/x/crypto/nacl/secretbox"
)

func encrypt(chunk []byte, passphrase string) (encrypted []byte) {

	var key [32]byte
	copy(key[:], passphrase)

	var nonce [24]byte
	_, err := io.ReadFull(rand.Reader, nonce[:])
	if err != nil {
		panic(err)
	}

	encrypted = secretbox.Seal(nonce[:], chunk, &nonce, &key)
	return
}

func decrypt(chunk []byte, passphrase string) (decrypted []byte) {

	var key [32]byte
	copy(key[:], passphrase)

	var decryptNonce [24]byte
	copy(decryptNonce[:], chunk[:24])

	decrypted, ok := secretbox.Open(nil, chunk[24:], &decryptNonce, &key)
	if !ok {
		panic("error decrypting")
	}
	return
}
