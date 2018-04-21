package main

import (
	"context"
	"flag"
	"fmt"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/github"
)

const (
	GITHUB_PR    = "https://github.com/kubernetes/kubernetes/pull"
	GITHUB_ISSUE = "https://github.com/kubernetes/kubernetes/issues"

	USER         = "send-from@mail.com"
	PASSWORD     = "---------"
	SERVER       = "smtp.mail.com"
	SEND_TO      = "send-to@mail.com"
	NICK_NAME    = "Github-Robot"
	SUBJECT      = "Kubernetes-Issue/PR-Of-Scheduling"
	CONTENT_TYPE = "Content-Type: text/plain; charset=UTF-8"
)

var allevents map[string]string
var sendevents map[string]string

var (
	numOfPages = flag.Int("n", 1, "pages.")
	period     = flag.Int("p", 5, "period")
)

func main() {
	flag.Parse()
	allevents = make(map[string]string, 0)
	sendevents = make(map[string]string, 0)
	client := github.NewClient(nil)

	auth := smtp.PlainAuth("", USER, PASSWORD, SERVER)
	to := []string{SEND_TO}

	for {
		select {
		case <-time.After(time.Duration(*period) * time.Second):
			for i := 1; i <= 10; i++ {
				events, _, err := client.Activity.ListIssueEventsForRepository(context.Background(), "kubernetes", "kubernetes", &github.ListOptions{i, 100})
				if err != nil {
					fmt.Println(err)
				}
				for _, v := range events {
					e := fmt.Sprintf("%v", v)
					if strings.Contains(e, "sig/scheduling") {
						if left := strings.LastIndex(e, GITHUB_ISSUE); left >= 0 {
							e = e[left:]
							fmt.Println(e)
							if right := strings.Index(e, `"`); right > 0 {
								if _, ok := allevents[e[:right]]; !ok {
									allevents[e[:right]] = e[:right]
								}
							}
						} else if left := strings.LastIndex(e, GITHUB_PR); left >= 0 {
							e = e[left:]
							fmt.Println(e)
							if right := strings.Index(e, `.p`); right > 0 {
								if _, ok := allevents[e[:right]]; !ok {
									allevents[e[:right]] = e[:right]
								}
							}
						}
					}
				}
			}
			body := "Issue/PR List: \r\n"
			es := make([]string, 0)
			for key, value := range allevents {
				left := strings.LastIndex(value, "/")
				if id, err := strconv.Atoi(value[(left + 1):]); err != nil && id > 62937 {
					if _, ok := sendevents[key]; !ok {
						es = append(es, value)
						sendevents[key] = value
					}
				}
			}
			if len(es) == 0 {
				break
			}
			body += strings.Join(es, "\r\n")
			msg := []byte("To: " + strings.Join(to, ",") + "\r\nFrom: " + NICK_NAME +
				"<" + USER + ">\r\nSubject: " + SUBJECT + "\r\n" + CONTENT_TYPE + "\r\n\r\n" + body)
			err := smtp.SendMail(SERVER+":25", auth, USER, to, msg)
			if err != nil {
				fmt.Printf("send mail error: %v\n", err)
			}
		}
	}

}
