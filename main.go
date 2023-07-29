package main

import (
	"log"
)

func main() {
	// initialize global variables.
	if err := initConstants(); err != nil {
		log.Fatal("problem during initializing constants", err)
	}

	// load config file and interprete yaml.
	var configData goifo_conf_s
	if err := loadConfig(configFile, &configData); err != nil {
		log.Fatal("config file could not be interpret as an yaml file:", err)
	}

	// performs a dry run of goifo to get sure material on imap server will not be crippled because
	// errors in config file.  stops goifo's action if errors occure.
	{
		configProcessor := dryRunConfigProcessor_s{}

		if err := process_goifo_conf(&configProcessor, &configData); err != nil {
			log.Fatal("[dry run] ", err)
		}
	}

	// performs the real run of goifo.
	{
		configProcessor := configProcessor_s{}

		if err := process_goifo_conf(&configProcessor, &configData); err != nil {
			log.Fatal(err)
		}
	}

}
