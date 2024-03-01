package main

import (
	"crypto/tls"
	"crypto/x509"
	"log"
)

func main() {
	// initialize global variables.
	if err := initConstants(); err != nil {
		log.Fatal("problem during initializing constants", err)
	}

	pCertPool, err := x509.SystemCertPool()
	if err != nil {
		log.Fatal("problem getting system cert pool", err)
	}
	if err := get_ca_cert_pool(pCertPool); err != nil {
		log.Fatal("problem reading ca certificates", err)
	}
	tlsConfig := tls.Config{
		RootCAs: pCertPool}

	// load config file and interprete yaml.
	var configData goifo_conf_s
	if err := loadConfig(configFile, &configData); err != nil {
		log.Fatal("config file could not be interpret as an yaml file:", err)
	}

	// performs a dry run of goifo to get sure material on imap server will not be crippled because
	// errors in config file.  stops goifo's action if errors occure.
	{
		configProcessor := dryRunConfigProcessor_s{}

		if err := process_goifo_conf(&configProcessor, &configData, &tlsConfig); err != nil {
			log.Fatal("[dry run] ", err)
		}
	}

	// performs the real run of goifo.
	{
		configProcessor := configProcessor_s{}

		if err := process_goifo_conf(&configProcessor, &configData, &tlsConfig); err != nil {
			log.Fatal(err)
		}
	}

}
