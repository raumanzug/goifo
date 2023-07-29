package main

// stuff for interpreting yaml config file.

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/itchyny/timefmt-go"
	"gopkg.in/yaml.v3"
)

// structs describing structure of yaml config file.

type goifo_conf_s struct {
	Servers []server_s `yaml:""`
}

type server_s struct {
	Host           string      `yaml: ""`
	NoTLS          bool        `yaml: ",omitempty"`
	NoSimpleLogin  bool        `yaml: ",omitempty"`
	NoSASLPlain    bool        `yaml: ",omitempty"`
	NoSASLExternal bool        `yaml: ",omitempty"`
	Username       string      `yaml: ",omitempty"`
	Password       string      `yaml: ",omitempty"`
	Identity       string      `yaml: ",omitempty"`
	Mailboxes      []mailbox_s `yaml: ",omitempty"`
}

type mailbox_s struct {
	Name  string      `yaml: ""`
	Rules []yaml.Node `yaml: ",omitempty"`
}

type rule_s struct {
	Preconditions []yaml.Node            `yaml: ""`
	Action        map[string][]yaml.Node `yaml: ""`
}

type precondition_s struct {
	Field  string      `yaml: ""`
	Values []yaml.Node `yaml: ""`
}

// loadConfig loads the cconfig file named by actualConfigFile.
// pConfigData will contain read data if no error occure.
// If error occure this is indicated by return value err != 0.
func loadConfig(actualConfigFile string, pConfigData *goifo_conf_s) (err error) {
	in, err := os.ReadFile(actualConfigFile)
	if err != nil {
		return
	}

	err = yaml.Unmarshal(in, pConfigData)
	if err != nil {
		return
	}

	return
}

// iStringCollector is a callback interface for structs gathering search keys
// for the imap's SEARCH command.
// cf. [https://pkg.go.dev/github.com/mxk/go-imap/imap#Client.Search]
type iStringCollector interface {
	append(s string)
}

// iRuleProcessor is a callback interface for structs implementing
// imap operations for processing a rule.
type iRuleProcessor interface {
	iStringCollector
	search() (err error)          // perform imap's SEARCH command for processing preconditions
	move(dest string) (err error) // perform imap's COPY command for copying emails processing move actions
	markSrcForDel() (err error)   // mark emails as deleted by imap's STORE command so that emails are erased after closing mailbox.
}

// iMailboxProcessor is a callback interface for structs implementing
// imap operations for processing a mailbox.
type iMailboxProcessor interface {
	selectMailbox(name string) (err error) // perform imap's SELECT command opening a mailbox.
	newRuleProcessor() iRuleProcessor      // produce iRuleProcessor for processing rules related to this mailbox.
	close() (err error)                    // perform imap's CLOSE command for shutting down a mailbox and deleting marked emails.
}

// iServerProcessor is a callback interface for structs implementing
// imap operations for processing a server.
type iServerProcessor interface {
	connect(
		host string,
		noTLS bool,
		noSimpleLogin bool,
		noSASLPlainLogin bool,
		noSASLExternal bool,
		username string,
		password string,
		identity string) (err error) // connects an imap server and authenticate
	newMailboxProcessor() iMailboxProcessor // produce iMailboxProcessor for processing mailbox related to this server.
	logout() (err error)                    // perform imap's LOGOUT command for shutting down imap sessions.
}

// iConfigProcessor is a callback interface for structs implementing
// handling data from config file.
type iConfigProcessor interface {
	newServerProcessor() iServerProcessor // produce iServerProcessor for processing servers mentioned in config file.
}

// implements aforementioned interfaces with dry run structs.
// A dry run is for running config file parsing without imap operation.
// Each of the following implementation does actually nothing.

type dryRunRuleProcessor_s struct {
}

func (processor dryRunRuleProcessor_s) append(s string) {
}

func (processor dryRunRuleProcessor_s) search() (err error) {
	return
}

func (processor dryRunRuleProcessor_s) move(dest string) (err error) {
	return
}

func (processor dryRunRuleProcessor_s) markSrcForDel() (err error) {
	return
}

type dryRunMailboxProcessor_s struct {
}

func (processor dryRunMailboxProcessor_s) selectMailbox(name string) (err error) {
	return
}

func (processor dryRunMailboxProcessor_s) newRuleProcessor() iRuleProcessor {
	return dryRunRuleProcessor_s{}
}

func (processor dryRunMailboxProcessor_s) close() (err error) {
	return
}

type dryRunServerProcessor_s struct {
}

func (processor dryRunServerProcessor_s) connect(
	host string,
	noTLS bool,
	noSimpleLogin bool,
	noSASLPlainLogin bool,
	noSASLExternal bool,
	username string,
	password string,
	identity string) (err error) {
	return
}

func (processor dryRunServerProcessor_s) newMailboxProcessor() iMailboxProcessor {
	return dryRunMailboxProcessor_s{}
}

func (processor dryRunServerProcessor_s) logout() (err error) {
	return
}

type dryRunConfigProcessor_s struct {
}

func (processor dryRunConfigProcessor_s) newServerProcessor() iServerProcessor {
	return dryRunServerProcessor_s{}
}

// weaveLocation adds location information for error messages given in argument original.
// line and column number are given in arguments line and column.
func weaveLocation(line, column int, original string) (retval string) {
	return fmt.Sprintf("%s:%d.%d: %s", configFile, line, column, original)
}

// searchFieldError is issued if a field is not admitted in preconditions.
type searchFieldError struct {
	line        int
	column      int
	searchField string
}

func (e searchFieldError) Error() string {
	return weaveLocation(e.line, e.column, fmt.Sprintf("unknown search field %s", e.searchField))
}

// argLengthError is issued if the number of values given in precondition is wrong.
type argLengthError struct {
	line          int
	column        int
	searchField   string
	nrArgActual   uint32
	nrArgExpected uint32
}

// actionNotDefined is issued if action is unknown.
// The only action admitted so far is move.
type actionNotDefinedError struct {
	line        int
	column      int
	actionField string
}

func (e actionNotDefinedError) Error() string {
	return weaveLocation(e.line, e.column, fmt.Sprintf("unknown action type %s", e.actionField))
}

func (e argLengthError) Error() string {
	return weaveLocation(e.line, e.column, fmt.Sprintf("search field %s takes %d args.  %d args given", e.searchField, e.nrArgExpected, e.nrArgActual))
}

// process_string_value provides a string for using as search key in imap's SEARCH command.
func process_string_value(collector iStringCollector, pValue *yaml.Node) (err error) {
	var s string
	err = pValue.Decode(&s)
	if err != nil {
		return
	}

	collector.append(s)

	return
}

// process_uint32_value provides a uint32 for using as search key in imap's SEARCH command.
func process_uint32_value(collector iStringCollector, pValue *yaml.Node) (err error) {
	var n uint32
	err = pValue.Decode(&n)
	if err != nil {
		return
	}

	s := fmt.Sprintf("%d", n)
	collector.append(s)

	return
}

// process_duration_value provides a duration for producing search keys for imap's SEARCH command.
func process_duration_value(collector iStringCollector, pValue *yaml.Node) (err error) {
	var d time.Duration
	err = pValue.Decode(&d)
	if err != nil {
		return
	}

	t := time.Now().Add(-d)
	s := timefmt.Format(t, "%d-%b-%Y")
	collector.append(s)

	return
}

// process_time_value provides a date for using as search key in imap's SEARCH command.
func process_time_value(collector iStringCollector, pValue *yaml.Node) (err error) {
	var t time.Time
	err = pValue.Decode(&t)
	if err != nil {
		return
	}

	s := timefmt.Format(t, "%d-%b-%Y")
	collector.append(s)

	return
}

// process a precondition, i.e. it provides search keys for use in imap's SEARCH command.
func process_precondition(collector iStringCollector, pValue *yaml.Node) (err error) {
	var precondition precondition_s
	err = pValue.Decode(&precondition)
	if err != nil {
		return
	}

	switch f := precondition.Field; f {
	case "ALL":
		collector.append(f)
		if l := uint32(len(precondition.Values)); l != 0 {
			err = argLengthError{pValue.Line, pValue.Column, f, l, 0}
			return
		}
	case "ANSWERED":
		collector.append(f)
		if l := uint32(len(precondition.Values)); l != 0 {
			err = argLengthError{pValue.Line, pValue.Column, f, l, 0}
			return
		}
	case "BCC":
		collector.append(f)
		if l := uint32(len(precondition.Values)); l != 1 {
			err = argLengthError{pValue.Line, pValue.Column, f, l, 1}
			return
		}
		err = process_string_value(collector, &precondition.Values[0])
	case "BEFORE":
		collector.append(f)
		if l := uint32(len(precondition.Values)); l != 1 {
			err = argLengthError{pValue.Line, pValue.Column, f, l, 1}
			return
		}
		err = process_time_value(collector, &precondition.Values[0])
	case "BODY":
		collector.append(f)
		if l := uint32(len(precondition.Values)); l != 1 {
			err = argLengthError{pValue.Line, pValue.Column, f, l, 1}
			return
		}
		err = process_string_value(collector, &precondition.Values[0])
	case "CC":
		collector.append(f)
		if l := uint32(len(precondition.Values)); l != 1 {
			err = argLengthError{pValue.Line, pValue.Column, f, l, 1}
			return
		}
		err = process_string_value(collector, &precondition.Values[0])
	case "DELETED":
		collector.append(f)
		if l := uint32(len(precondition.Values)); l != 0 {
			err = argLengthError{pValue.Line, pValue.Column, f, l, 0}
			return
		}
	case "DRAFT":
		collector.append(f)
		if l := uint32(len(precondition.Values)); l != 0 {
			err = argLengthError{pValue.Line, pValue.Column, f, l, 0}
			return
		}
	case "FLAGGED":
		collector.append(f)
		if l := uint32(len(precondition.Values)); l != 0 {
			err = argLengthError{pValue.Line, pValue.Column, f, l, 0}
			return
		}
	case "FROM":
		collector.append(f)
		if l := uint32(len(precondition.Values)); l != 1 {
			err = argLengthError{pValue.Line, pValue.Column, f, l, 1}
			return
		}
		err = process_string_value(collector, &precondition.Values[0])
	case "HEADER":
		collector.append(f)
		if l := uint32(len(precondition.Values)); l != 2 {
			err = argLengthError{pValue.Line, pValue.Column, f, l, 2}
			return
		}
		err = process_string_value(collector, &precondition.Values[0])
		if err != nil {
			return
		}
		err = process_string_value(collector, &precondition.Values[1])
	case "KEYWORD":
		collector.append(f)
		if l := uint32(len(precondition.Values)); l != 1 {
			err = argLengthError{pValue.Line, pValue.Column, f, l, 1}
			return
		}
		err = process_string_value(collector, &precondition.Values[0])
	case "LARGER":
		collector.append(f)
		if l := uint32(len(precondition.Values)); l != 1 {
			err = argLengthError{pValue.Line, pValue.Column, f, l, 1}
			return
		}
		err = process_uint32_value(collector, &precondition.Values[0])
	case "MSG":
		for _, v := range precondition.Values {
			err = process_uint32_value(collector, &v)
			if err != nil {
				return
			}
		}
	case "NEW":
		collector.append(f)
		if l := uint32(len(precondition.Values)); l != 0 {
			err = argLengthError{pValue.Line, pValue.Column, f, l, 0}
			return
		}
	case "NOT":
		collector.append(f)
		if l := uint32(len(precondition.Values)); l != 1 {
			err = argLengthError{pValue.Line, pValue.Column, f, l, 1}
			return
		}
		err = process_precondition(collector, &precondition.Values[0])
	case "OLD":
		collector.append(f)
		if l := uint32(len(precondition.Values)); l != 0 {
			err = argLengthError{pValue.Line, pValue.Column, f, l, 0}
			return
		}
	case "OLDERTHAN":
		collector.append("BEFORE")
		if l := uint32(len(precondition.Values)); l != 1 {
			err = argLengthError{pValue.Line, pValue.Column, f, l, 1}
			return
		}
		err = process_duration_value(collector, &precondition.Values[0])
	case "ON":
		collector.append(f)
		if l := uint32(len(precondition.Values)); l != 1 {
			err = argLengthError{pValue.Line, pValue.Column, f, l, 1}
			return
		}
		err = process_time_value(collector, &precondition.Values[0])
	case "OR":
		if len(precondition.Values) > 0 {
			values := precondition.Values
			for ; len(values) > 1; values = values[1:] {
				collector.append(f)
				err = errors.Join(err, process_precondition(collector, &values[0]))
			}
			err = errors.Join(err, process_precondition(collector, &values[0]))
		}
	case "RECENT":
		collector.append(f)
		if l := uint32(len(precondition.Values)); l != 0 {
			err = argLengthError{pValue.Line, pValue.Column, f, l, 0}
			return
		}
	case "SEEN":
		collector.append(f)
		if l := uint32(len(precondition.Values)); l != 0 {
			err = argLengthError{pValue.Line, pValue.Column, f, l, 0}
			return
		}
	case "SENTBEFORE":
		collector.append(f)
		if l := uint32(len(precondition.Values)); l != 1 {
			err = argLengthError{pValue.Line, pValue.Column, f, l, 1}
			return
		}
		err = process_time_value(collector, &precondition.Values[0])
	case "SENTON":
		collector.append(f)
		if l := uint32(len(precondition.Values)); l != 1 {
			err = argLengthError{pValue.Line, pValue.Column, f, l, 1}
			return
		}
		err = process_time_value(collector, &precondition.Values[0])
	case "SENTSINCE":
		collector.append(f)
		if l := uint32(len(precondition.Values)); l != 1 {
			err = argLengthError{pValue.Line, pValue.Column, f, l, 1}
			return
		}
		err = process_time_value(collector, &precondition.Values[0])
	case "SINCE":
		collector.append(f)
		if l := uint32(len(precondition.Values)); l != 1 {
			err = argLengthError{pValue.Line, pValue.Column, f, l, 1}
			return
		}
		err = process_time_value(collector, &precondition.Values[0])
	case "SMALLER":
		collector.append(f)
		if l := uint32(len(precondition.Values)); l != 1 {
			err = argLengthError{pValue.Line, pValue.Column, f, l, 1}
			return
		}
		err = process_uint32_value(collector, &precondition.Values[0])
	case "SUBJECT":
		collector.append(f)
		if l := uint32(len(precondition.Values)); l != 1 {
			err = argLengthError{pValue.Line, pValue.Column, f, l, 1}
			return
		}
		err = process_string_value(collector, &precondition.Values[0])
	case "TEXT":
		collector.append(f)
		if l := uint32(len(precondition.Values)); l != 1 {
			err = argLengthError{pValue.Line, pValue.Column, f, l, 1}
			return
		}
		err = process_string_value(collector, &precondition.Values[0])
	case "TO":
		collector.append(f)
		if l := uint32(len(precondition.Values)); l != 1 {
			err = argLengthError{pValue.Line, pValue.Column, f, l, 1}
			return
		}
		err = process_string_value(collector, &precondition.Values[0])
	case "UID":
		collector.append("(")
		collector.append(f)
		for _, v := range precondition.Values {
			err = process_string_value(collector, &v)
			if err != nil {
				return
			}
		}
		collector.append(")")
	case "UNANSWERED":
		collector.append(f)
		if l := uint32(len(precondition.Values)); l != 0 {
			err = argLengthError{pValue.Line, pValue.Column, f, l, 0}
			return
		}
	case "UNDELETED":
		collector.append(f)
		if l := uint32(len(precondition.Values)); l != 0 {
			err = argLengthError{pValue.Line, pValue.Column, f, l, 0}
			return
		}
	case "UNDRAFT":
		collector.append(f)
		if l := uint32(len(precondition.Values)); l != 0 {
			err = argLengthError{pValue.Line, pValue.Column, f, l, 0}
			return
		}
	case "UNFLAGGED":
		collector.append(f)
		if l := uint32(len(precondition.Values)); l != 0 {
			err = argLengthError{pValue.Line, pValue.Column, f, l, 0}
			return
		}
	case "UNKEYWORD":
		collector.append(f)
		if l := uint32(len(precondition.Values)); l != 1 {
			err = argLengthError{pValue.Line, pValue.Column, f, l, 1}
			return
		}
		err = process_string_value(collector, &precondition.Values[0])
	case "UNSEEN":
		collector.append(f)
		if l := uint32(len(precondition.Values)); l != 0 {
			err = argLengthError{pValue.Line, pValue.Column, f, l, 0}
			return
		}
	default:
		err = searchFieldError{pValue.Line, pValue.Column, f}
	}

	return
}

// process_move_action perform the copy part of a move action instructed by a rule.
func process_move_action(processor iRuleProcessor, v []yaml.Node) (err error) {
	for _, destRaw := range v {
		var dest string
		decodeError := destRaw.Decode(&dest)
		if decodeError != nil {
			err = errors.Join(err, decodeError)
			continue
		}
		{
			moveError := processor.move(dest)
			if moveError != nil {
				err = errors.Join(err, moveError)
				return
			}
		}
	}

	return
}

// process_rule perform actions related to a rule.
func process_rule(processor iRuleProcessor, pRule *yaml.Node) (err error) {
	var rule rule_s
	err = pRule.Decode(&rule)
	if err != nil {
		return
	}

	for _, precondition := range rule.Preconditions {
		err = errors.Join(err, process_precondition(processor, &precondition))
	}

	if err != nil {
		return
	}

	err = processor.search()
	if err != nil {
		return
	}

	{
		isSrcToBeDeleted := false

		for k, v := range rule.Action {
			switch k {
			case "move":
				{
					moveError := process_move_action(processor, v)
					if moveError != nil {
						err = errors.Join(err, moveError)
						return
					}
				}
				isSrcToBeDeleted = true
			default:
				err = errors.Join(err, actionNotDefinedError{pRule.Line, pRule.Column, k})
			}
		}

		if isSrcToBeDeleted {
			err = errors.Join(err, processor.markSrcForDel())
		}
	}

	return
}

// process_mailbox performs actions related to a mailbox.
func process_mailbox(processor iMailboxProcessor, pMailbox *mailbox_s) (err error) {
	err = processor.selectMailbox(pMailbox.Name)
	if err != nil {
		return
	}

	defer func() {
		err = errors.Join(err, processor.close())
	}()

	for _, rule := range pMailbox.Rules {
		ruleProcessor := processor.newRuleProcessor()
		err = errors.Join(err, process_rule(ruleProcessor, &rule))
	}

	return
}

// process_server performs actions related to a server.
func process_server(processor iServerProcessor, pServer *server_s) (err error) {
	err = processor.connect(
		pServer.Host,
		pServer.NoTLS,
		pServer.NoSimpleLogin,
		pServer.NoSASLPlain,
		pServer.NoSASLExternal,
		pServer.Username,
		pServer.Password,
		pServer.Identity)
	if err != nil {
		return
	}

	defer func() {
		err = errors.Join(err, processor.logout())
	}()

	for _, mailbox := range pServer.Mailboxes {
		mailboxProcessor := processor.newMailboxProcessor()
		err = errors.Join(err, process_mailbox(mailboxProcessor, &mailbox))
	}

	return
}

// process_goifo_conf performs actions instructed by yaml config file.
func process_goifo_conf(processor iConfigProcessor, pConfigData *goifo_conf_s) (err error) {
	for _, server := range pConfigData.Servers {
		serverProcessor := processor.newServerProcessor()
		err = errors.Join(err, process_server(serverProcessor, &server))
	}

	return
}
