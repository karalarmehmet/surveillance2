package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"strconv"
	"time"

	"github.com/aymerick/douceur/inliner"
	"github.com/karalarmehmet/surveillance/internal/channeldata"
	mail "github.com/xhit/go-simple-mail/v2"
	"jaytaylor.com/html2text"
)

// NewWorker initializes a new worker with an ID and a worker pool channel.
func NewWorker(id int, workerPool chan chan channeldata.MailJob) Worker {
	return Worker{
		id:         id,
		jobQueue:   make(chan channeldata.MailJob),
		workerPool: workerPool,
		quitChan:   make(chan bool),
	}
}

// Worker holds info for a pool worker.
type Worker struct {
	id         int
	jobQueue   chan channeldata.MailJob
	workerPool chan chan channeldata.MailJob
	quitChan   chan bool
}

// start launches the worker.
func (w Worker) start() {
	go func() {
		for {
			// Add jobQueue to the worker pool.
			w.workerPool <- w.jobQueue

			select {
			case job := <-w.jobQueue:
				w.processMailQueueJob(job.MailMessage)
			case <-w.quitChan:
				fmt.Printf("worker%d stopping\n", w.id)
				return
			}
		}
	}()
}

// NewDispatcher creates and returns a new Dispatcher object.
func NewDispatcher(jobQueue chan channeldata.MailJob, maxWorkers int) *Dispatcher {
	workerPool := make(chan chan channeldata.MailJob, maxWorkers)
	return &Dispatcher{
		jobQueue:   jobQueue,
		maxWorkers: maxWorkers,
		workerPool: workerPool,
	}
}

// Dispatcher holds info for a dispatcher.
type Dispatcher struct {
	workerPool chan chan channeldata.MailJob
	maxWorkers int
	jobQueue   chan channeldata.MailJob
}

// run initializes and starts all workers, then begins dispatching jobs.
func (d *Dispatcher) run() {
	for i := 0; i < d.maxWorkers; i++ {
		worker := NewWorker(i+1, d.workerPool)
		worker.start()
	}

	go d.dispatch()
}

// dispatch handles job distribution by using a for range loop.
func (d *Dispatcher) dispatch() {
	for job := range d.jobQueue {
		go func(job channeldata.MailJob) {
			workerJobQueue := <-d.workerPool
			workerJobQueue <- job
		}(job)
	}
}

// processMailQueueJob processes the mail job and sends the email.
func (w Worker) processMailQueueJob(mailMessage channeldata.MailData) {

	data := struct {
		Content       template.HTML
		From          string
		FromName      string
		PreferenceMap map[string]string
		IntMap        map[string]int
		StringMap     map[string]string
		FloatMap      map[string]float32
		RowSets       map[string]interface{}
	}{
		Content:       mailMessage.Content,
		FromName:      mailMessage.FromName,
		From:          mailMessage.FromAddress,
		PreferenceMap: preferenceMap,
		IntMap:        mailMessage.IntMap,
		StringMap:     mailMessage.StringMap,
		FloatMap:      mailMessage.FloatMap,
		RowSets:       mailMessage.RowSets,
	}

	paths := []string{
		"./views/mail.tmpl",
	}

	t := template.Must(template.New("mail.tmpl").ParseFiles(paths...))

	var tpl bytes.Buffer
	if err := t.Execute(&tpl, data); err != nil {
		fmt.Print(err)
		return // Early return on error
	}

	result := tpl.String()

	plainText, err := html2text.FromString(result, html2text.Options{PrettyTables: true})
	if err != nil {
		plainText = ""
	}

	var formattedMessage string

	formattedMessage, err = inliner.Inline(result)
	if err != nil {
		log.Println(err)
		formattedMessage = result
	}

	port, _ := strconv.Atoi(preferenceMap["smtp_port"])

	server := mail.NewSMTPClient()
	server.Host = preferenceMap["smtp_server"]
	server.Port = port
	server.Username = preferenceMap["smtp_user"]
	server.Password = preferenceMap["smtp_password"]
	if preferenceMap["smtp_server"] == "localhost" {
		server.Authentication = mail.AuthPlain
	} else {
		server.Authentication = mail.AuthLogin
	}
	server.Encryption = mail.EncryptionNone
	server.KeepAlive = false
	server.ConnectTimeout = 10 * time.Second
	server.SendTimeout = 10 * time.Second

	smtpClient, err := server.Connect()
	if err != nil {
		log.Println("error is ", err)
		return // Early return on error
	}

	email := mail.NewMSG()
	email.SetFrom(mailMessage.FromAddress).
		AddTo(mailMessage.ToAddress).
		SetSubject(mailMessage.Subject)

	if len(mailMessage.AdditionalTo) > 0 {
		for _, x := range mailMessage.AdditionalTo {
			email.AddTo(x)
		}
	}

	if len(mailMessage.CC) > 0 {
		for _, x := range mailMessage.CC {
			email.AddCc(x)
		}
	}

	if len(mailMessage.Attachments) > 0 {
		for _, x := range mailMessage.Attachments {
			email.AddAttachment(x)
		}
	}

	email.SetBody(mail.TextHTML, formattedMessage)
	email.AddAlternative(mail.TextPlain, plainText)

	err = email.Send(smtpClient)
	if err != nil {
		log.Println(err)
	} else {
		log.Println("Email Sent")
	}
}
