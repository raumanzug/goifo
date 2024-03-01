`goifo` clears up your imap servers by sending emails stored on
it to mailboxes or deleting them. `goifo` is software intended
to use as cron job and is controlled by a configuration file
`"${HOME}/.config/goifo/config.yaml"`.  It can be started without any
command line parameter, i.e. simply by

	`go env GOPATH`/bin/goifo

if you have `goifo` installed correctly.

# CA x509 certificates

Optional file `"{HOME}/.config/ca.pem"` contains x509 CA certificates
in PEM format for validating x509 server certificates which IMAP servers
present during TLS sessions.

# Configuring `goifo`

`goifo` is configured by a [yaml](http://yaml.org) file located in
`"${HOME}/.config/goifo/config.yaml"`.  A file like this:

	servers:
	  - host: imap.die-sieben-zwerge.de
	    notls: false
	    nosimplelogin: false
	    nosaslplain: false
	    nosaslexternal: false
	    username: "schneewittchen@die-sieben-zwerge.de"
	    password: "wird_nicht_verraten"
	    identity: ""
	    mailboxes:
	      - name: INBOX
	        rules:
	          - preconditions:
	               - field:  FROM
	                 values:
	                    -  "schwiegermutter@aol.com"
	            action:
	              move: []
	          - preconditions:
	               - field:  FROM
	                 values:
	                    -  "@jobagent.stepstone.de"
	            action:
	              move:
	                - StepStone
	          - preconditions:
	               - field:  OLDERTHAN
	                 values:
	                    -  2500h
	               - field:  SEEN
	                 values: []
	            action:
	              move:
	                - Archive
	          - preconditions:
	               - field: BODY
	                 values:
	                    - "enlarge your penis"
	            action:
	              move:
	                - Spam
	      - name: Archive
	        rules:
	          - preconditions:
	               - field:  OLDERTHAN
	                 values:
	                    -  12000h
	            action:
	              move: []

`goifo` can handle multiple imap servers.  Each imap server manages
mailboxes.  For each mailbox this configuration file contains a sequence of
instructions.  Aforementioned configuration file manages one imap server.
This server manages several mailboxes.  For two of them, `INBOX` and
`Archive`, `goifo` instruct to move or delete emails according to rules
given in this configuration file.

Below the keyword `servers` for each imap server `goifo` should manage a
block is given.  Consider the following statements in this configuration
file:

	host: imap.die-sieben-zwerge.de
	username: "schneewittchen@die-sieben-zwerge.de"
	password: "wird_nicht_verraten"
	identity: ""

`host` set the DNS name of the imap server.  `username`/`password`
contains the credentials if imap server does not use external SASL
authentication.  `identity` is relevant only for SASL authentication.

	notls: false
	nosimplelogin: false
	nosaslplain: false
	nosaslexternal: false

`notls` states whether `goifo` should not use TLS on socket layer.
`nosimplelogin` states whether simple login performed by `LOGIN`
command at imap server should be forbidden.  `nosaslplain` and
`nosaslexternal` state whether plain or external SASL authentication
should be forbidden, respectively.

The block below `mailboxes` imposes a series of rules.  Each instruction
consists in a sequence of precondition and one action.  Aforementioned
configuration file contains following rule:

	preconditions:
	   - field: BODY
             values:
                - "enlarge your penis"
	actions:
	  move:
	    - Spam

It instructs `goifo` to move any email with string "enlarge your
penis" in its body to mailbox "Spam".  Each precondition consists in
a keyword taken from a predefined set given at `field` and a sequence
of arguments given at `values:`.  Herein, the keyword `BODY` says that
thie precondition holds if the emails body contains string "enlarge
your penis".  Below `action` we see the `move` action which instructs
`goifo` to move emails satisfying the preconditions to mailbox `Spam`.


## Admitted precondition keywords.

Each [searching criteria given in the imap specification for the search command](https://www.ietf.org/rfc/rfc3501.html#section-6.4.4)
can be used as precondition, `ALL`, `ANSWERED`, `BCC` etc.  Their
arguments have to be given under `values`.  Besides this we also can use
keyword `MSG` and `OLDERTHAN`.

### `MSG` keyword

`values` enumerates numbers that means sequence numbers in the mailbox.

### `OLDERTHAN` keyword

the only argument means a time duration, e.g. `12h` for 12 hours.
[Here](https://pkg.go.dev/time@go1.20.6#ParseDuration) the format is explained.
This duration will be subtracted by recent time and set as argument for a
`BEFORE` search keyword.

### Time parameters in search criteria

`goifo` expects ISO dates and not dates in imap manner.  Instead of
10-Dec-1968 in imap's format we use 1968-12-10 which is a date in
ISO format.  The precondition

	- field: SINCE
	  values:
             - 1968-12-10

holds for each email on an imap mailbox with an internal date later than
10th of december in 1968.


### Logical operations on preconditions

One gets the resulting preconditions by AND operation on the set of
preconditions given in a rule.  But we are also able to apply logical
OR and NOT operations on preconditions.  See here:

	- field: OR
	  values:
	     - field: NOT
	       values:
	          - field: SINCE
	            values:
	               - 1968-12-10
	     - field: UNSEEN
	       values: []
	     - field: BODY
	       values:
	          - "Schneewittchen"

### Actions

the only type of action is the `move` action.  Their arguments mean
the destinations where the emails filtered by the preconditions will be
moved to.  If this list is empty these emails will be deleted.
