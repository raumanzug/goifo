package main

// Constants, global variables and their initializing funcs.

import (
	"os"
	"path"
	"time"
)

const projectName = "goifo"

// logoutTimeout defines the time span an imap session will be alive.
// cf. [here](https://pkg.go.dev/github.com/mxk/go-imap/imap#Client.Logout)
const logoutTimeout = 1000 * time.Second

// variables initialized by initConstants func, see below
var (
	xdgConfigDir string // Standard config dir according to XDG.  Most likely ~/.config
	configDir    string // Most likely ~/.config/goifo
	configFile   string // Name of config file.  Most likely ~/.config/goifo/config.yaml
	caFile       string // Name of file containing ca certificates.  Most likely ~/.config/ca.pem
)

// initConstants initialize aforementioned global variables
// Call this function before any other action of goiko's source code!
func initConstants() (err error) {
	xdgConfigDir, err := os.UserConfigDir()
	if err != nil {
		return
	}

	configDir = path.Join(xdgConfigDir, projectName)
	configFile = path.Join(configDir, "config.yaml")
	caFile = path.Join(xdgConfigDir, "ca.pem")

	return
}
