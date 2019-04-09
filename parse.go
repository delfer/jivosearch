package main

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/datatogether/warc"
	"github.com/jinzhu/gorm"
	"github.com/kamilsk/breaker"
	"github.com/kamilsk/retry"
	"github.com/kamilsk/retry/backoff"
	"github.com/kamilsk/retry/strategy"
	"github.com/urfave/cli"
)

func parse(c *cli.Context) error {
	log.Println("Started")

	// Connect to DB
	db, err := gorm.Open(c.GlobalString("dialect"), c.GlobalString("db-connection"))
	if err != nil {
		return err
	}
	defer db.Close()

	db.AutoMigrate(&JivoSite{})

	// Start DB Status saver
	dbUpdateCh := make(chan *Task, 100000)
	go dbStatusSaver(dbUpdateCh, db)

	// Start all pages parser
	anySiteCh := make(chan *AnySite, 100)
	tgtSiteCh := make(chan *TargetSite, 100)
	outSiteCh := make(chan *TargetSite, 100)
	numThreads, _ := strconv.Atoi(c.GlobalString("threads"))
	for i := 0; i < numThreads; i++ {
		go pageParser(anySiteCh, tgtSiteCh)
		go widgetParser(tgtSiteCh, outSiteCh)
	}
	go dbWriter(outSiteCh, db, c.String("name"))

	// Main loop over WARC files
	for {
		// Read WARC file url with retry
		var task Task
		action := func(uint) error {
			res := db.Where("worker_name = '' OR (completed IS NULL AND started < ?)", time.Now().AddDate(0, 0, -1)).First(&task)
			if res.Error != nil {
				log.Println(res.Error)
			}
			return res.Error
		}
		if err := retry.Retry(breaker.BreakByTimeout(5*time.Minute), action, strategy.Backoff(backoff.Exponential(time.Second, 1.2))); err != nil {
			panic(err)
		}

		// Set start time and worker name
		now := time.Now()
		task.Started = &now
		task.WorkerName = c.String("name")

		dbUpdateCh <- &task

		// Open file remote
		resp, err := http.Get(task.URL)
		if err != nil || resp.StatusCode != 200 {
			task.Started = nil
			task.WorkerName = ""
			dbUpdateCh <- &task
			continue
		}
		defer resp.Body.Close()

		log.Println("File opened")

		// Create reader from file
		rdr, err := warc.NewReader(resp.Body)
		if err != nil {
			task.Started = nil
			task.WorkerName = ""
			dbUpdateCh <- &task
			continue
		}

		log.Println("Read started")

		// Read records
		for i := 0; ; i++ {
			rec, err := rdr.Read()
			if err == io.EOF {
				break
			}
			if rec.Type == warc.RecordTypeResponse {
				body, _ := rec.Body()
				anySiteCh <- &AnySite{URL: rec.TargetURI(), Body: body, SourceWarc: task.URL}
			}
		}

		now = time.Now()
		task.Completed = &now
		dbUpdateCh <- &task
	}
}

func dbStatusSaver(ch <-chan *Task, db *gorm.DB) {
	for {
		task := <-ch
		log.Println("Update", task)
		// Update with Retry
		action := func(uint) error {
			res := db.Save(&task)
			if res.Error != nil {
				log.Println(res.Error)
			}
			return res.Error
		}
		if err := retry.Retry(breaker.BreakByTimeout(5*time.Minute), action, strategy.Backoff(backoff.Exponential(time.Second, 1.2))); err != nil {
			panic(err)
		}
	}
}

func pageParser(in <-chan *AnySite, out chan<- *TargetSite) {
	for {
		site := <-in
		body := string(site.Body)
		if strings.Contains(body, "code.jivosite.com") {
			out <- &TargetSite{URL: site.URL, Body: body, SourceWarc: site.SourceWarc}
		}
	}
}

func widgetParser(in <-chan *TargetSite, out chan<- *TargetSite) {
	r, _ := regexp.Compile(`widget_id\s*=\s*['"](\w+)['"]`)
	for {
		site := <-in
		widgetIDMatches := r.FindStringSubmatch(site.Body)
		site.WidgetID = widgetIDMatches[1]
		u, err := url.Parse(site.URL)
		if err == nil {
			site.URL = u.Scheme + "://" + u.Host
		}
		out <- site
	}
}

func dbWriter(in <-chan *TargetSite, db *gorm.DB, workerName string) {
	i := 0
	for {
		site := <-in
		i++
		// Retry
		action := func(uint) error {
			now := time.Now()
			out := JivoSite{
				URL:        site.URL,
				WidgetID:   site.WidgetID,
				WorkerName: workerName,
				Found:      &now,
				SourceWarc: site.SourceWarc,
			}
			res := db.FirstOrCreate(&JivoSite{URL: site.URL}, &out)
			if res.Error != nil {
				log.Println(res.Error)
			}
			return res.Error
		}
		if err := retry.Retry(breaker.BreakByTimeout(5*time.Minute), action, strategy.Backoff(backoff.Exponential(time.Second, 1.2))); err != nil {
			panic(err)
		}
		if i%100 == 0 {
			log.Println("Written " + strconv.Itoa(i))
		}
	}
}
