package main

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"github.com/mxk/go-imap/imap"

	. "github.com/arnehilmann/goutils"
)

func fetch(server string, username string, fetchedDir string) error {
	log.Println("server: ", server)

	var password_rot13, _ = ioutil.ReadFile("pw")
	var password = Rot13(strings.TrimSpace(string(password_rot13)))

	var (
		c   *imap.Client
		cmd *imap.Command
		rsp *imap.Response
		err error
	)

	var config *tls.Config
	c, err = imap.DialTLS(server, config)
	PanicIf(err)

	defer c.Logout(60 * time.Second)

	log.Println("Server says hello:", c.Data[0].Info)
	c.Data = nil

	if c.State() == imap.Login {
		_, err = c.Login(username, password)
		PanicIf(err)
	} else {
		log.Panic("not in login mode?!")
	}

	c.Select("INBOX", true)

	cmd, err = c.Search("(OR (FROM \"arne.hilmann@gmail.com\") (FROM \"arne.hilmann@t-online.de\"))" +
		"(OR (TO \"arne.hilmann@gmail.com\") (TO \"arne.hilmann@t-online.de\"))")
	PanicIf(err)

	set, _ := imap.NewSeqSet("")
	for cmd.InProgress() {
		c.Recv(-1)
		for _, rsp = range cmd.Data {
			for _, uid := range rsp.SearchResults() {
				if !isAlreadyFetched(fetchedDir, uid) {
					set.AddNum(uid)
				}
			}
		}
		cmd.Data = nil
	}

	if set.Empty() {
		return nil
	}

	cmd, _ = c.Fetch(set, "RFC822")
	os.MkdirAll(fetchedDir, 0755)

	for cmd.InProgress() {
		c.Recv(-1)

		for _, rsp = range cmd.Data {
			log.Println(rsp)
			body := imap.AsBytes(rsp.MessageInfo().Attrs["RFC822"])
			err = ioutil.WriteFile(calcFilepath(fetchedDir, rsp.MessageInfo().Seq), body, 0644)
			WarnIf(err)
		}
		cmd.Data = nil

		for _, rsp = range c.Data {
			fmt.Println("Server data:", rsp)
		}
		c.Data = nil
	}

	if rsp, err := cmd.Result(imap.OK); err != nil {
		if err == imap.ErrAborted {
			fmt.Println("Fetch command aborted")
		} else {
			fmt.Println("Fetch error:", rsp.Info)
		}
	}
	return nil
}

func isAlreadyFetched(fetchedDir string, seq uint32) bool {
	_, err := os.Stat(calcFilepath(fetchedDir, seq))
	return err == nil
}

func calcFilepath(fetchedDir string, seq uint32) string {
	return path.Join(fetchedDir, fmt.Sprintf("%06d", seq))
}

func main() {
	fetch("secureimap.t-online.de", "arne.hilmann@t-online.de", "out/fetched")
}
