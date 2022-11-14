package main

import (
	"os"

	"github.com/libp2p/go-libp2p-core/crypto"
)

func loadKeys() (crypto.PrivKey, crypto.PubKey) {

	var prvKey crypto.PrivKey
	var pubKey crypto.PubKey
	// If there is an exsisting key, use it
	if _, err := os.Stat("prvKey"); err == nil {

		keyBuff, err := os.ReadFile("prvKey")
		handleError(err, "Error reading Private Key")

		prvKey, err = crypto.UnmarshalPrivateKey(keyBuff)
		handleError(err, "Error Unmarshaling Private Key")

		keyBuff, err = os.ReadFile("pubKey")
		handleError(err, "Error reading Public Key")

		pubKey, err = crypto.UnmarshalPublicKey(keyBuff)
		handleError(err, "Error Unmarshaling Public Key")

	} else {
		// Make a new key if a key pair isn't avalable
		prvKey, pubKey, err = crypto.GenerateKeyPair(crypto.RSA, 2048)
		handleError(err, "Error Generating RSA key pair")

		// Creates a new RSA key pair for this host.
		keyBuff, err := crypto.MarshalPrivateKey(prvKey)
		handleError(err, "Error Marshaling Private Key")

		err = os.WriteFile("prvKey", keyBuff, 0644)
		handleError(err, "Error Writeing Private Key")

		keyBuff, err = crypto.MarshalPublicKey(pubKey)
		handleError(err, "Error Marshaling Public Key")

		err = os.WriteFile("pubKey", keyBuff, 0644)
		handleError(err, "Error Writeing Public Key")

	}

	return prvKey, pubKey

}
