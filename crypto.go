package main

import (
	"fmt"
	"os"

	"github.com/libp2p/go-libp2p-core/crypto"
)

func loadKeys() (crypto.PrivKey, crypto.PubKey) {

	var prvKey crypto.PrivKey
	var pubKey crypto.PubKey
	// If there is an exsisting key, use it
	if _, err := os.Stat("prvKey"); err == nil {

		keyBuff, err := os.ReadFile("prvKey")
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error reading Private Key")
			panic(err)
		}
		prvKey, err = crypto.UnmarshalPrivateKey(keyBuff)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error Unmarshaling Private Key")
			panic(err)
		}

		keyBuff, err = os.ReadFile("pubKey")
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error reading Public Key")
			panic(err)
		}

		pubKey, err = crypto.UnmarshalPublicKey(keyBuff)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error Unmarshaling Public Key")
			panic(err)
		}

	} else {
		// Make a new key if a key pair isn't avalable
		prvKey, pubKey, err = crypto.GenerateKeyPair(crypto.RSA, 2048)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error Generating RSA key pair")
			panic(err)
		}

		// Creates a new RSA key pair for this host.
		keyBuff, err := crypto.MarshalPrivateKey(prvKey)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error Marshaling Private Key")
			panic(err)
		}

		err = os.WriteFile("prvKey", keyBuff, 0644)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error Writeing to prvKey")
		}

		keyBuff, err = crypto.MarshalPublicKey(pubKey)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error Marshaling Public Key")
			panic(err)
		}

		err = os.WriteFile("pubKey", keyBuff, 0644)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error Writeing to pubKey")
		}

	}

	return prvKey, pubKey

}
