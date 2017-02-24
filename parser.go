package readlater

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"

	. "github.com/arnehilmann/goutils"
)

var skipTexts = map[string]bool{"Tweet": true, "Download": true, "@SciencePorn": true}

func parseMultipart(message *mail.Message, boundary string, id int, subject string, dir string) (Article, error) {
	reader := multipart.NewReader(message.Body, boundary)
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		contentType, _, err := mime.ParseMediaType(part.Header.Get("Content-Type"))
		WarnIf(err)
		switch {
		case strings.HasPrefix(contentType, "text/html"):
			return parseTextHtml(part, id)
		case strings.HasPrefix(contentType, "application/pdf"):
			return parseApplicationPdf(part, id, subject, dir)
		}
	}
	return NewArticleWithText(id, subject, "<unparseable multipart>"), nil
}

func parseApplicationPdf(message *multipart.Part, id int, subject string, dir string) (Article, error) {
	var filename string
	_, params, _ := mime.ParseMediaType(message.Header.Get("Content-Type"))
	filename = params["filename"]
	if filename == "" {
		_, params, _ := mime.ParseMediaType(message.Header.Get("Content-Disposition"))
		filename = params["filename"]
	}
	if filename == "" {
		return NilArticle, errors.New(fmt.Sprintf("%v: pdf without filename?!", id))
	}

	filepath := path.Join(dir, filename)
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		var reader io.Reader
		if message.Header.Get("Content-Transfer-Encoding") == "base64" {
			reader = base64.NewDecoder(base64.StdEncoding, message)
		} else {
			reader = message
		}
		blob, err := ioutil.ReadAll(reader)
		WarnIf(err)
		if err != nil {
			return NilArticle, errors.New(fmt.Sprintf("%v: pdf with corrupt content?!", id))
		}
		err = ioutil.WriteFile(filepath, blob, 0755)
		WarnIf(err)
		log.Println(fmt.Sprintf("%v: pdf written", id))
	}

	return NewArticleWithUrl(id, subject, filename), nil
}

func parseTextHtml(message *multipart.Part, id int) (Article, error) {
	var doc *goquery.Document
	var err error
	if message.Header.Get("Content-Transfer-Encoding") == "base64" {
		reader := base64.NewDecoder(base64.StdEncoding, message)
		doc, err = goquery.NewDocumentFromReader(reader)
	} else {
		doc, err = goquery.NewDocumentFromReader(message)
	}
	PanicIf(err)
	var article Article
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		if article != NilArticle {
			return
		}
		linktext := s.Text()
		if strings.HasPrefix(linktext, "@") || strings.HasPrefix(linktext, "#") {
			return
		}
		if !skipTexts[linktext] {
			url, ok := s.Attr("href")
			if ok {
				if strings.HasPrefix(url, "https://twitter.com/") || strings.HasPrefix(url, "mailto:") {
					return
				}
				article = NewArticleWithUrl(id, linktext, url)
			}
		}
	})
	if article == NilArticle {
		return NilArticle, errors.New("cannot parse html")
	}
	return article, nil
}

func parseTextPlain(message *mail.Message, id int, subject string) (Article, error) {
	if message.Body == nil {
		return NilArticle, errors.New(fmt.Sprintf("%v: empty body", id))
	}
	var reader io.Reader
	if message.Header.Get("Content-Transfer-Encoding") == "quoted-printable" {
		reader = quotedprintable.NewReader(message.Body)
	} else {
		reader = message.Body
	}
	bodyBytes, _ := ioutil.ReadAll(reader)

	for _, line := range strings.Split(string(bodyBytes), "\n") {
		if strings.HasPrefix(line, "http") {
			return NewArticleWithUrl(id, subject, strings.TrimSpace(line)), nil
		}
	}
	return NewArticleWithText(id, subject, string(bodyBytes)), nil
}

func ParseMessage(filepath string, dir string) (Article, error) {
	id, err := strconv.Atoi(path.Base(filepath))
	PanicIf(err)

	file, err := os.Open(filepath)
	PanicIf(err)

	reader := bufio.NewReader(file)
	message, err := mail.ReadMessage(reader)
	PanicIf(err)

	dec := new(mime.WordDecoder)

	from, err := dec.DecodeHeader(message.Header.Get("From"))
	WarnIf(err)
	to, err := dec.DecodeHeader(message.Header.Get("To"))
	WarnIf(err)

	if !strings.Contains(from, "arne.hilmann") || !strings.Contains(to, "arne.hilmann") {
		return NilArticle, errors.New(fmt.Sprintf("%v: invalid 'from' or 'to'", id))
	}
	if strings.Contains(to, ", ") {
		return NilArticle, errors.New(fmt.Sprintf("%v: to many recipients", id))
	}
	subject, err := dec.DecodeHeader(message.Header.Get("Subject"))
	if err != nil {
		subject = message.Header.Get("Subject")
	}

	contentType, params, err := mime.ParseMediaType(message.Header.Get("Content-Type"))
	WarnIf(err)
	switch {
	case strings.HasPrefix(contentType, "multipart/"):
		return parseMultipart(message, params["boundary"], id, subject, dir)
	case strings.HasPrefix(contentType, "text/plain"):
		return parseTextPlain(message, id, subject)
	}
	return NilArticle, errors.New(fmt.Sprintf("%v: has no known content type %v", id, contentType))
}
