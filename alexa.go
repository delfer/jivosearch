package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"log"
	"math/rand"
	"net/url"
	"strconv"
	"time"

	"net/http"

	"github.com/jinzhu/gorm"

	"github.com/kamilsk/breaker"
	"github.com/kamilsk/retry"
	"github.com/kamilsk/retry/backoff"
	"github.com/kamilsk/retry/strategy"
	"github.com/tevino/abool"
	"github.com/urfave/cli"
)

func alexa(c *cli.Context) error {
	log.Println("Started")

	// Connect to DB
	db, err := gorm.Open(c.GlobalString("dialect"), c.GlobalString("db-connection"))
	if err != nil {
		return err
	}
	defer db.Close()

	toAlexa := make(chan *JivoSite, 10000)
	toDB := make(chan *JivoSite, 10000)

	alexaCommunicatorDone := abool.New()
	alexaSaverDone := abool.New()

	numThreads, _ := strconv.Atoi(c.GlobalString("threads"))
	for i := 0; i < numThreads; i++ {
		go alexaCommunicator(toAlexa, toDB, alexaCommunicatorDone)
		go alexaSaver(toDB, db, alexaSaverDone)
	}

	// Configure DB pool size
	db.DB().SetMaxIdleConns(numThreads)
	db.DB().SetMaxOpenConns(numThreads)
	db.DB().SetConnMaxLifetime(0)

	sites := []*JivoSite{}
	db.Where("alexa_popularity = 0 OR alexa_popularity IS NULL").Find(&sites)

	log.Println("Got sites list", len(sites))

	for _, site := range sites {
		toAlexa <- site
	}

	log.Println("All sites in OCR queue")
	timer := time.NewTicker(1 * time.Second)
	for {
		<-timer.C
		log.Println(len(toAlexa), len(toDB))
		if len(toAlexa) == 0 && len(toDB) == 0 && alexaCommunicatorDone.IsSet() && alexaSaverDone.IsSet() {
			break
		}
	}

	log.Println("Done!")
	return nil
}

type alexaPOPULARITY struct {
	TEXT int `xml:"TEXT,attr"`
}

type alexaREACH struct {
	RANK int `xml:"RANK,attr"`
}

type alexaSD struct {
	POPULARITY alexaPOPULARITY
	REACH      alexaREACH
}

type alexaRLS struct {
	TITLE string `xml:"TITLE,attr"`
}

type alexaResp struct {
	RLS alexaRLS
	SD  alexaSD
	URL string `xml:"URL,attr"`
}

func alexaCommunicator(in <-chan *JivoSite, out chan<- *JivoSite, done *abool.AtomicBool) {
	for {
		site := <-in
		done.UnSet()

		pop := -1
		rank := -1
		// Get with Retry
		action := func(uint) error {
			u, err := url.Parse(site.URL)
			if err != nil {
				return nil
			}

			resp, err := http.Get("http://data.alexa.com/data?cli=10&url=" + u.Host)
			if err != nil {
				log.Println(err)
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				log.Println("Status error: " + resp.Status)
				return errors.New("Status error: " + resp.Status)
			}

			var buf bytes.Buffer
			tee := io.TeeReader(resp.Body, &buf)

			var a alexaResp

			err = xml.NewDecoder(tee).Decode(&a)
			if err != nil {
				log.Println(err)
				return err
			}

			if len(a.URL) < 3 {
				log.Println("Alexa empty response")
				log.Println(buf.String())
				log.Println(a)
				return errors.New("Alexa empty response")
			}

			if len(a.RLS.TITLE) > 0 {
				log.Println("Alexa reate limit reached")
				log.Println(buf.String())
				log.Println(a)
				timer1 := time.NewTimer(time.Duration(rand.Intn(60)) * time.Minute)
				<-timer1.C
				return errors.New("Alexa reate limit reached")
			} else if (a.SD.POPULARITY.TEXT == 0 || a.SD.REACH.RANK == 0) {
				// No data in Alexa
				pop = -1
				rank = -1
				return nil
			}

			pop = a.SD.POPULARITY.TEXT
			rank = a.SD.REACH.RANK

			return nil
		}
		if err := retry.Retry(breaker.BreakByTimeout(12*time.Hour), action, strategy.Backoff(backoff.Exponential(time.Second, 1.2))); err != nil {
			log.Println(err)
		}

		site.AlexaPopularity = pop
		site.AlexaRank = rank

		out <- site
		done.Set()
	}
}

func alexaSaver(in <-chan *JivoSite, db *gorm.DB, done *abool.AtomicBool) {
	for {
		site := <-in
		done.UnSet()

		log.Println(site)

		// Update with Retry
		action := func(uint) error {
			res := db.Model(&site).Update("alexa_popularity", site.AlexaPopularity).Update("alexa_rank", site.AlexaRank)
			if res.Error != nil {
				log.Println(res.Error)
			}
			return res.Error
		}
		if err := retry.Retry(breaker.BreakByTimeout(5*time.Minute), action, strategy.Backoff(backoff.Exponential(time.Second, 1.2))); err != nil {
			panic(err)
		}
		done.Set()
	}
}
