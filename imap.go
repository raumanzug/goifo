package main

// All stuff about imap control.
// Here, some interfaces will be implemented defined [here](config.go).

import (
	"errors"
	"log"

	"github.com/mxk/go-imap/imap"
)

// ruleProcessor_s implements iRuleProcessor
// Processing preconditions of rules we execute a SEARCH command on imap server.
type ruleProcessor_s struct {
	accu           []imap.Field // for gathering search keys used in imap's SEARCH command. cf. [https://pkg.go.dev/github.com/mxk/go-imap/imap#Client.Search]
	pClient        *imap.Client // handle for imap network connection.
	pSearchResults *imap.SeqSet // Search results appeare here.
}

// newRuleProcessor creates a ruleProcessor_s instance.
func newRuleProcessor(pClient *imap.Client) (retval *ruleProcessor_s) {
	retval = &ruleProcessor_s{
		accu:    []imap.Field{},
		pClient: pClient}
	retval.pSearchResults, _ = imap.NewSeqSet("")
	return
}

// append add a search key item for imap's SEARCH command.
func (a *ruleProcessor_s) append(s string) {
	(*a).accu = append((*a).accu, s)
}

// search performs SEARCH command.
func (a *ruleProcessor_s) search() (err error) {
	var cmd *imap.Command

	if len(a.accu) == 0 {
		cmd, err = imap.Wait(a.pClient.Search("ALL"))
	} else {
		cmd, err = imap.Wait(a.pClient.Search(a.accu...))
	}

	if err != nil {
		return
	}

	for _, rsp := range cmd.Data {
		a.pSearchResults.AddNum(rsp.SearchResults()...)
	}

	return
}

// move performs a copy action which is part of performing move instructed by a rule.
func (a *ruleProcessor_s) move(dest string) (err error) {
	if !a.pSearchResults.Empty() {
		_, err = imap.Wait(a.pClient.Copy(a.pSearchResults, dest))
		if err != nil {
			return
		}
	}

	return
}

// markSrcForDel marks emails which were moved so that they can be deleted after closing mailbox.
// It is part of performing move instruction by a rule.
func (a *ruleProcessor_s) markSrcForDel() (err error) {
	if !a.pSearchResults.Empty() {
		_, err = imap.Wait(a.pClient.Store(a.pSearchResults, "+FLAGS.SILENT", imap.NewFlagSet("\\Deleted")))
	}

	return
}

// mailboxProcessor_s implements iMailboxProcessor.
type mailboxProcessor_s struct {
	pClient *imap.Client
}

// newMailboxProcessor creates a mailboxProcess_s instance.
func newMailboxProcessor(pClient *imap.Client) (retval *mailboxProcessor_s) {
	retval = &mailboxProcessor_s{pClient: pClient}
	return
}

// selectMailbox performs the SELECT command which starts working with a mailbox in a imap session.
func (a *mailboxProcessor_s) selectMailbox(name string) (err error) {
	_, err = a.pClient.Select(name, false)
	return
}

func (a *mailboxProcessor_s) newRuleProcessor() iRuleProcessor {
	return newRuleProcessor(a.pClient)
}

// close closes a mailbox and expunge emails marked as deleted by effect of aforementioned marSrcForDel func.
func (a *mailboxProcessor_s) close() (err error) {
	_, err = a.pClient.Close(true)
	return
}

// serverProcessor_s implements iServerProcessor.
type serverProcessor_s struct {
	pClient *imap.Client
}

// newServerProcessor creates a serverProcessor_s instance.
func newServerProcessor() (retval *serverProcessor_s) {
	retval = &serverProcessor_s{}
	return
}

// connect starts a session on an imap server and performs authentication.
func (a *serverProcessor_s) connect(
	host string,
	noTLS bool,
	noSimpleLogin bool,
	noSASLPlainLogin bool,
	noSASLExternal bool,
	username string,
	password string,
	identity string) (err error) {
	if noTLS {
		a.pClient, err = imap.Dial(host)
	} else {
		a.pClient, err = imap.DialTLS(host, nil)
	}
	if err != nil {
		return
	}

	a.pClient.SetLogMask(imap.LogRaw)

	if a.pClient.Caps["STARTTLS"] {
		_, err = a.pClient.StartTLS(nil)
	}
	if err != nil {
		return
	}

	if !noSASLExternal && a.pClient.State() == imap.Login {
		if _, err := a.pClient.Auth(imap.ExternalAuth("")); err != nil {
			log.Print("sasl external:", err)
		}
	}

	if !noSASLPlainLogin && a.pClient.State() == imap.Login {
		if _, err := a.pClient.Auth(imap.PlainAuth(username, password, identity)); err != nil {
			log.Print("sasl plain:", err)
		}
	}

	if !noSimpleLogin && a.pClient.State() == imap.Login {
		if _, err := a.pClient.Login(username, password); err != nil {
			log.Print("login auth:", err)
		}
	}

	if a.pClient.State() != imap.Auth {
		err = errors.New("authentication failed")
		return
	}

	return
}

func (a *serverProcessor_s) newMailboxProcessor() iMailboxProcessor {
	return newMailboxProcessor(a.pClient)
}

// logout ends a session on an imap server.
func (a *serverProcessor_s) logout() (err error) {
	_, err = a.pClient.Logout(logoutTimeout)
	return
}

// configProcessor_s implements iConfigProcessor
type configProcessor_s struct {
}

func (a *configProcessor_s) newServerProcessor() iServerProcessor {
	return newServerProcessor()
}
