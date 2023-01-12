package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
)

var availableAlgs = []string{
	ssh.CertAlgoRSAv01,
	ssh.CertAlgoDSAv01,
	ssh.CertAlgoECDSA256v01,
	ssh.CertAlgoECDSA384v01,
	ssh.CertAlgoECDSA521v01,
	ssh.CertAlgoSKECDSA256v01,
	ssh.CertAlgoED25519v01,
	ssh.CertAlgoSKED25519v01,
	ssh.CertAlgoRSASHA256v01,
	ssh.CertAlgoRSASHA512v01,
	ssh.KeyAlgoRSA,
	ssh.KeyAlgoDSA,
	ssh.KeyAlgoECDSA256,
	ssh.KeyAlgoSKECDSA256,
	ssh.KeyAlgoECDSA384,
	ssh.KeyAlgoECDSA521,
	ssh.KeyAlgoED25519,
	ssh.KeyAlgoSKED25519,
	ssh.KeyAlgoRSASHA256,
	ssh.KeyAlgoRSASHA512,
}

var algorithms = flag.String("algorithms", strings.Join(availableAlgs, ","), "Comma seperated list of algorithms to try")

var handshakeErr = errors.New("handshake error")

func main() {
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		log.Fatalf("usage: %s <host>", os.Args[0])
	}

	dst := args[0]

	if strings.Index(dst, ":") < 0 {
		dst = dst + ":22"
	}
	host, port, err := net.SplitHostPort(dst)
	if err != nil {
		log.Fatalf("failed to parse: %s, err: %s", dst, err)
	}

	algsList := strings.Split(*algorithms, ",")

	seenKeys := make(map[string]ssh.PublicKey)

	var collectedErrors []error

	for _, alg := range algsList {
		var didHandshake bool
		config := &ssh.ClientConfig{
			Auth:              []ssh.AuthMethod{},
			HostKeyAlgorithms: []string{alg},
			HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
				didHandshake = true

				fp := ssh.FingerprintSHA256(key)
				if _, seen := seenKeys[fp]; seen {
					return handshakeErr
				}

				seenKeys[fp] = key

				fmt.Printf("key: %s\n", bytes.TrimSpace(ssh.MarshalAuthorizedKey(key)))
				fmt.Printf("sha256: %s\n", ssh.FingerprintSHA256(key))
				fmt.Printf("md5: %s\n", ssh.FingerprintLegacyMD5(key))

				cert, ok := key.(*ssh.Certificate)
				if ok {
					certJson, err := json.MarshalIndent(cert, "", "  ")
					if err != nil {
						panic(err)
					}
					fmt.Printf("cert: %s\n", certJson)
				}

				fmt.Println()
				return handshakeErr
			},
		}

		client, err := ssh.Dial("tcp", host+":"+port, config)
		if didHandshake {
			continue
		}
		if err != nil {
			collectedErrors = append(collectedErrors, err)
			continue
		}
		client.Close()
	}

	if len(seenKeys) == 0 {
		log.Fatalf("Failed to fetch keys: %+v", collectedErrors)
	}

}
