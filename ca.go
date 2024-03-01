package main

// All stuff abount reading ca.pem

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

func get_ca_cert_pool(pCertPool *x509.CertPool) (err error) {
	data, err := os.ReadFile(caFile)
	if os.IsNotExist(err) {
		err = nil
		return
	} else if err != nil {
		return
	}

	for {
		block, rest := pem.Decode(data)
		data = rest
		if block == nil {
			break
		}

		switch block.Type {
		case "CERTIFICATE":
			certificate, cerr := x509.ParseCertificate(block.Bytes)
			if cerr != nil {
				err = errors.Join(
					err,
					cerr)
				continue
			}

			pCertPool.AddCert(certificate)

		default:
			err = errors.Join(
				err,
				errors.New(fmt.Sprintf("pem type %s not supported.\n", block.Type)))
		}
	}

	return
}
