package main

import (
	"log"
	"net/url"
	"regexp"
	"strconv"
	"time"

	// 	"net/http"
	"github.com/jinzhu/gorm"

	"github.com/kamilsk/breaker"
	"github.com/kamilsk/retry"
	"github.com/kamilsk/retry/backoff"
	"github.com/kamilsk/retry/strategy"
	ocrclient "github.com/tleyden/open-ocr-client"
	"github.com/urfave/cli"
)

func rate(c *cli.Context) error {
	log.Println("Started")

	// Connect to DB
	db, err := gorm.Open(c.GlobalString("dialect"), c.GlobalString("db-connection"))
	if err != nil {
		return err
	}
	defer db.Close()

	toOCR := make(chan *JivoSite, 10000)
	toDB := make(chan *JivoSite, 10000)

	numThreads, _ := strconv.Atoi(c.GlobalString("threads"))
	for i := 0; i < numThreads; i++ {
		for j := 0; j < 10; j++ {
			go ocrCommunicator(toOCR, toDB, c.String("ocr-server"))
		}
		go cycSaver(toDB, db)
	}

	// Configure DB pool size
	db.DB().SetMaxIdleConns(numThreads)
	db.DB().SetMaxOpenConns(numThreads)
	db.DB().SetConnMaxLifetime(0)

	sites := []*JivoSite{}
	db.Where("cyc = 0").Find(&sites)

	log.Println("Got sites list", len(sites))

	for _, site := range sites {
		toOCR <- site
	}

	log.Println("All sites in OCR queue")
	timer := time.NewTicker(1 * time.Second)
	for {
		<-timer.C
		log.Println(len(toOCR), len(toDB))
		if len(toOCR) == 0 && len(toDB) == 0 {
			break
		}
	}

	log.Println("Done!")
	return nil
}

func ocrCommunicator(in <-chan *JivoSite, out chan<- *JivoSite, openOcrURL string) {
	client := ocrclient.NewHttpClient(openOcrURL)
	reNums := regexp.MustCompile(`[\d\s]+`)
	reSpace := regexp.MustCompile(`[\s]`)
	for {
		site := <-in
		cyc := -1
		// Get with Retry
		action := func(uint) error {
			u, err := url.Parse(site.URL)
			if err != nil {
				return nil
			}

			ocrRequest := ocrclient.OcrRequest{
				ImgUrl:     "https://yandex.ru/cycounter?" + u.Host,
				EngineType: ocrclient.ENGINE_TESSERACT,
			}

			ocrDecoded, err := client.DecodeImageUrl(ocrRequest)
			if err != nil {
				log.Println(err)
				return err
			}

			ocrNum := reNums.FindString(ocrDecoded)
			ocrNum = reSpace.ReplaceAllString(ocrNum, "")

			ocrI, err := strconv.Atoi(ocrNum)
			if err != nil {
				log.Println(err)
				return err
			}

			cyc = ocrI
			return nil
		}
		if err := retry.Retry(breaker.BreakByTimeout(20*time.Second), action, strategy.Backoff(backoff.Exponential(time.Second, 1.2))); err != nil {
			log.Println(err)
		}

		site.CYC = cyc
		out <- site
	}
}

func cycSaver(in <-chan *JivoSite, db *gorm.DB) {
	for {
		site := <-in
		log.Println(site)
		// Update with Retry
		action := func(uint) error {
			res := db.Model(&site).Update("cyc", site.CYC)
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
