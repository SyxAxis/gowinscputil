package pkg

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	mathrnd "math/rand"

	"github.com/kayrus/putty"
	"golang.org/x/crypto/ssh"
)

/*

	Module that will read a Putty formatted PPK key off disk
	check and convert to a standard OpenSSH formatted key
	hand it back as []byte so it can be used to make SFTP
	connections through the Go SSH module.
	Note: only the source is on disk, the rest is done in memory

*/

var puttyPrivateKey interface{}

func ConvertPuttyFormattedKey(puttyKeyPath string, showoutputkey bool) ([]byte, error) {

	// break open the putty key format
	puttyKey, puttyPrivateKey := GetPrivateKeyFromPutty(puttyKeyPath)
	var priv_pem []byte

	// RSA or ED25519 ( may need DSA but anyone serious about this stuff shouldn't be using DSA anymore! )
	switch puttyKey.Algo {
	case "ssh-ed25519":
		priv := *puttyPrivateKey.(*ed25519.PrivateKey)
		priv_pem = exportED25519PrivateKeyAsPemStr(priv)
		// 		fmt.Println(priv_pem)
	case "ssh-rsa":
		priv := puttyPrivateKey.(*rsa.PrivateKey)
		priv_pem = exportRsaPrivateKeyAsPemStr(priv)
	}

	if showoutputkey {
		fmt.Println(string(priv_pem))
		return priv_pem, nil
	}

	return priv_pem, nil

	// fmt.Println("\n=================================================\n")

	// original source if you need both the priv/pub parts for RSA
	// // Create the keys
	// // priv, _ := GenerateRsaKeyPair()
	// // Export the keys to pem string
	// priv_pem := ExportRsaPrivateKeyAsPemStr(priv)
	// pub_pem, _ := ExportRsaPublicKeyAsPemStr(pub)
	// // Import the keys from pem string
	// priv_parsed, _ := ParseRsaPrivateKeyFromPemStr(priv_pem)
	// pub_parsed, _ := ParseRsaPublicKeyFromPemStr(pub_pem)
	// // Export the newly imported keys
	// priv_parsed_pem := ExportRsaPrivateKeyAsPemStr(priv_parsed)
	// pub_parsed_pem, _ := ExportRsaPublicKeyAsPemStr(pub_parsed)
	// fmt.Println(priv_parsed_pem)
	// fmt.Println(pub_parsed_pem)

}

func GetPrivateKeyFromPutty(puttyKeyFile string) (*putty.Key, interface{}) {
	// read the key
	puttyKey, err := putty.NewFromFile(puttyKeyFile)
	if err != nil {
		log.Fatal(err)
	}

	// parse putty key
	if puttyKey.Encryption != "none" {
		// If the key is encrypted, decrypt it
		puttyPrivateKey, err = puttyKey.ParseRawPrivateKey([]byte("testkey"))
		if err != nil {
			log.Fatal(err)
		}
	} else {
		puttyPrivateKey, err = puttyKey.ParseRawPrivateKey(nil)
		if err != nil {
			log.Fatal(err)
		}
	}

	//log.Printf("%+#v", privateKey)
	//fmt.Printf("%+#v", privateKey)

	return puttyKey, puttyPrivateKey

}
func exportRsaPrivateKeyAsPemStr(privkey *rsa.PrivateKey) []byte {
	privkey_bytes := x509.MarshalPKCS1PrivateKey(privkey)
	privkey_pem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privkey_bytes,
		},
	)
	return privkey_pem
}

func exportED25519PrivateKeyAsPemStr(privkey ed25519.PrivateKey) []byte {

	// github.com/mikesmitty/edkey/
	// Generate a new private/public keypair for OpenSSH
	// pubKey, privKey, _ := ed25519.GenerateKey(rand.Reader)
	// publicKey, _ := ssh.NewPublicKey(pubKey)

	// pemKey := &pem.Block{
	// 	Type:  "OPENSSH PRIVATE KEY",
	// 	Bytes: edkey.MarshalED25519PrivateKey(privKey),
	// }
	// privateKey := pem.EncodeToMemory(pemKey)
	// authorizedKey := ssh.MarshalAuthorizedKey(publicKey)

	privkey_pem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "OPENSSH PRIVATE KEY",
			Bytes: marshalED25519PrivateKey(privkey),
		},
	)
	return privkey_pem
}

func parseRsaPrivateKeyFromPemStr(privPEM string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(privPEM))
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the key")
	}

	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return priv, nil
}

/*
	Writes ed25519 private keys into the new OpenSSH private key format.

I have no idea why this isn't implemented anywhere yet, you can do seemingly
everything except write it to disk in the OpenSSH private key format.
*/
func marshalED25519PrivateKey(key ed25519.PrivateKey) []byte {
	// Add our key header (followed by a null byte)
	magic := append([]byte("openssh-key-v1"), 0)

	var w struct {
		CipherName   string
		KdfName      string
		KdfOpts      string
		NumKeys      uint32
		PubKey       []byte
		PrivKeyBlock []byte
	}

	// Fill out the private key fields
	pk1 := struct {
		Check1  uint32
		Check2  uint32
		Keytype string
		Pub     []byte
		Priv    []byte
		Comment string
		Pad     []byte `ssh:"rest"`
	}{}

	//
	// Uses MATH RAND and NOT crypto RAND
	//
	// Set our check ints
	ci := mathrnd.Uint32()
	pk1.Check1 = ci
	pk1.Check2 = ci

	// Set our key type
	pk1.Keytype = ssh.KeyAlgoED25519

	// Add the pubkey to the optionally-encrypted block
	pk, ok := key.Public().(ed25519.PublicKey)
	if !ok {
		//fmt.Fprintln(os.Stderr, "ed25519.PublicKey type assertion failed on an ed25519 public key. This should never ever happen.")
		return nil
	}
	pubKey := []byte(pk)
	pk1.Pub = pubKey

	// Add our private key
	pk1.Priv = []byte(key)

	// Might be useful to put something in here at some point
	pk1.Comment = ""

	// Add some padding to match the encryption block size within PrivKeyBlock (without Pad field)
	// 8 doesn't match the documentation, but that's what ssh-keygen uses for unencrypted keys. *shrug*
	bs := 8
	blockLen := len(ssh.Marshal(pk1))
	padLen := (bs - (blockLen % bs)) % bs
	pk1.Pad = make([]byte, padLen)

	// Padding is a sequence of bytes like: 1, 2, 3...
	for i := 0; i < padLen; i++ {
		pk1.Pad[i] = byte(i + 1)
	}

	// Generate the pubkey prefix "\0\0\0\nssh-ed25519\0\0\0 "
	prefix := []byte{0x0, 0x0, 0x0, 0x0b}
	prefix = append(prefix, []byte(ssh.KeyAlgoED25519)...)
	prefix = append(prefix, []byte{0x0, 0x0, 0x0, 0x20}...)

	// Only going to support unencrypted keys for now
	w.CipherName = "none"
	w.KdfName = "none"
	w.KdfOpts = ""
	w.NumKeys = 1
	w.PubKey = append(prefix, pubKey...)
	w.PrivKeyBlock = ssh.Marshal(pk1)

	magic = append(magic, ssh.Marshal(w)...)

	return magic
}

func exportRsaPublicKeyAsPemStr(pubkey *rsa.PublicKey) (string, error) {
	pubkey_bytes, err := x509.MarshalPKIXPublicKey(pubkey)
	if err != nil {
		return "", err
	}
	pubkey_pem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: pubkey_bytes,
		},
	)

	return string(pubkey_pem), nil
}

func ParseRsaPublicKeyFromPemStr(pubPEM string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pubPEM))
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	switch pub := pub.(type) {
	case *rsa.PublicKey:
		return pub, nil
	default:
		break // fall through
	}
	return nil, errors.New("Key type is not RSA")
}

func generateRsaKeyPair() (*rsa.PrivateKey, *rsa.PublicKey) {
	privkey, _ := rsa.GenerateKey(rand.Reader, 2048)
	return privkey, &privkey.PublicKey
}

func generateED25519KeyPair() (ed25519.PrivateKey, ed25519.PublicKey) {
	pubkey, privkey, _ := ed25519.GenerateKey(rand.Reader)
	return privkey, pubkey
}
