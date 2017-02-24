package readlater

import (
	"fmt"
	"path"
	"strings"
)

type Article struct {
	id      int
	subject string
	url     string
	text    string
}

func (a Article) String() string {
	if a.url != "" {
		subject := a.subject
		if a.subject == "" {
			subject = path.Base(a.url)
		}
		return fmt.Sprintf("%v [%s](%s)", a.id, subject, a.url)
	} else {
		return fmt.Sprintf("%v *%s* %s", a.id, a.subject, a.text)
	}
}

func (a Article) Markdown() string {
	if a.url != "" {
		subject := a.subject
		if a.subject == "" {
			subject = path.Base(a.url)
		}
		return fmt.Sprintf("* [%s](%s) _%v_\n", subject, a.url, a.id)
	} else {
		lines := []string{}
		for _, line := range strings.Split(a.text, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				lines = append(lines, fmt.Sprintf(" %s", line))
			}
		}
		text := strings.Join(lines, "\n")
		if a.subject == "" {
			return fmt.Sprintf("* %s _%v_\n", text, a.id)
		} else {
			return fmt.Sprintf("* *%s* %s _%v_\n", a.subject, text, a.id)
		}
	}
}

type ById []Article

func (a ById) Len() int           { return len(a) }
func (a ById) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ById) Less(i, j int) bool { return a[i].id < a[j].id }

var NilArticle = Article{}

func NewArticleWithUrl(id int, subject string, url string) Article {
	return Article{id, subject, url, ""}
}

func NewArticleWithText(id int, subject string, text string) Article {
	return Article{id, subject, "", text}
}
