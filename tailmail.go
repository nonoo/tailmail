package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/smtp"
	"os"
	"regexp"

	"github.com/hpcloud/tail"
)

var settings struct {
	ToEmails      []string
	FromEmail     string
	Subject       string
	SMTPUser      string
	SMTPPass      string
	SMTPHost      string
	SMTPPort      int
	LogFile       string
	MailLineCount int
	RegExp        string
}

func sendMails(lines []string) {
	var rcpt string
	for i, toAddr := range settings.ToEmails {
		if i > 0 {
			rcpt += ", "
		}
		rcpt += toAddr
	}

	for _, toAddr := range settings.ToEmails {
		log.Println("tailmail: sending email to", toAddr)

		// Connect to the remote SMTP server.
		c, err := smtp.Dial(fmt.Sprintf("%s:%d", settings.SMTPHost, settings.SMTPPort))
		if err != nil {
			log.Fatal(err)
		}

		// Set the sender and recipient first
		if err = c.Mail(settings.FromEmail); err != nil {
			log.Fatal(err)
		}
		if err = c.Rcpt(toAddr); err != nil {
			log.Fatal(err)
		}

		// Send the email body.
		wc, err := c.Data()
		if err != nil {
			log.Fatal(err)
		}
		var linesStr string
		for _, l := range lines {
			linesStr = linesStr + l + "\r\n"
		}
		_, err = fmt.Fprintf(wc, "To: %s\r\nSubject: %s\r\n\r\n%s", rcpt,
			settings.Subject, linesStr)
		if err != nil {
			log.Fatal(err)
		}
		err = wc.Close()
		if err != nil {
			log.Fatal(err)
		}

		// Send the QUIT command and close the connection.
		err = c.Quit()
		if err != nil {
			log.Fatal(err)
		}
	}
}

func main() {
	configFileName := "config.json"

	flag.StringVar(&configFileName, "c", configFileName, "config file to use, default: config.json")
	flag.Parse()

	cf, err := os.Open("config.json")
	if err != nil {
		log.Fatal(err)
	}

	if err = json.NewDecoder(cf).Decode(&settings); err != nil {
		log.Fatal("tailmail: error parsing config file:", err.Error())
	}

	// Jump to the end of the log.
	loc := tail.SeekInfo{Offset: 0, Whence: 2}
	tc, err := tail.TailFile(settings.LogFile, tail.Config{Follow: true,
		Location: &loc})
	if err != nil {
		log.Fatal(err)
	}

	r := regexp.MustCompile(settings.RegExp)
	var lineBuf []string
	for line := range tc.Lines {
		lineBuf = append(lineBuf, line.Text)
		if len(lineBuf) > settings.MailLineCount {
			lineBuf = lineBuf[1:]
		}

		if r.MatchString(line.Text) {
			log.Println("tailmail: got match:")
			for _, ml := range lineBuf {
				fmt.Println(ml)
			}
			sendMails(lineBuf)
		}
	}
}
