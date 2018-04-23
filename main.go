package main

import (
	"context"
	"fmt"
	"net/smtp"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/go-github/github"
	"github.com/wgliang/cron"
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

type Event struct {
	URL  string
	Time int64
}

var allevents map[string]Event
var sendevents map[string]Event
var client *github.Client
var to []string
var auth smtp.Auth
var c *cron.Cron

func init() {
	allevents = make(map[string]Event, 0)
	sendevents = make(map[string]Event, 0)
	client = github.NewClient(nil)

	auth = smtp.PlainAuth("", USER, PASSWORD, SERVER)
	to = []string{SEND_TO}

	c = cron.New()
	c.Start()
	defer c.Stop()

}

func fetchEvents(pages int) {
	for i := 1; i <= pages; i++ {
		fmt.Println("Task page:", i)
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
							allevents[e[:right]] = Event{
								e[:right],
								time.Now().Unix(),
							}
						}
					}
				} else if left := strings.LastIndex(e, GITHUB_PR); left >= 0 {
					e = e[left:]
					fmt.Println(e)
					if right := strings.Index(e, `.p`); right > 0 {
						if _, ok := allevents[e[:right]]; !ok {
							allevents[e[:right]] = Event{
								e[:right],
								time.Now().Unix(),
							}
						}
					}
				}
			}
		}
		time.Sleep(time.Second * 10)
	}
}

func task(taskType string) {
	if taskType == "hour" {
		fmt.Println("Run hour task:", time.Now())
		fetchEvents(10)
		sendEmail(taskType)
	} else {
		fetchEvents(100)
		fmt.Println("Run summary task:", time.Now())
		sendEmail(taskType)
	}

}

func sendEmail(taskType string) {
	body := "Hour Issue/PR List: \r\n\r\n"
	var es, summary []string
	if taskType == "summary" {
		summary = summaryEvents()
	}

	es = mergeEvents()
	if len(es) == 0 {
		return
	}
	body += strings.Join(es, "\r\n")

	if taskType == "summary" {
		body += "\r\n\r\nSummary Issues/PR List: \r\n\r\n"
		body += strings.Join(summary, "\r\n")
	}

	msg := []byte("To: " + strings.Join(to, ",") + "\r\nFrom: " + NICK_NAME +
		"<" + USER + ">\r\nSubject: " + SUBJECT + "\r\n" + CONTENT_TYPE + "\r\n\r\n" + body)
	err := smtp.SendMail(SERVER+":25", auth, USER, to, msg)
	if err != nil {
		fmt.Printf("send mail error: %v\n", err)
	}
}

func mergeEvents() []string {
	es := make([]string, 0)
	for key, value := range allevents {
		left := strings.LastIndex(value.URL, "/")
		if id, err := strconv.Atoi(value.URL[(left + 1):]); err != nil && id > 62937 {
			if _, ok := sendevents[key]; !ok {
				es = append(es, value.URL)
				sendevents[key] = value
			}
		}
	}
	return es
}

func summaryEvents() []string {
	summary := make([]string, 0)

	for _, value := range sendevents {
		if value.Time > (time.Now().Unix() - 86400) {
			summary = append(summary, value.URL)
		}
	}
	return summary
}

func main() {
	// Run every hour between 9:00 -- 23:00.
	c.AddFunc("0 9-23 * * * *", func() { task("hour") })

	// Run morning.
	c.AddFunc("0 8 * * * *", func() { task("summary") })

	chExit := make(chan os.Signal, 1)
	signal.Notify(chExit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	select {
	case <-chExit:
		fmt.Println("logcool EXIT...Bye.")
	}
}
