package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mssql"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/kamilsk/breaker"
	"github.com/kamilsk/retry"
	"github.com/kamilsk/retry/backoff"
	"github.com/kamilsk/retry/strategy"
	"github.com/urfave/cli"
)

type Task struct {
	URL        string `gorm:"PRIMARY_KEY;UNIQUE_INDEX"`
	WorkerName string
	Started    *time.Time
	Completed  *time.Time
}

func schedule(c *cli.Context) error {

	db, err := gorm.Open(c.GlobalString("dialect"), c.GlobalString("db-connection"))
	if err != nil {
		return err
	}
	defer db.Close()

	db.AutoMigrate(&Task{})

	// Open remote file
	resp, err := http.Get(c.String("paths-server") + c.String("paths-uri"))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.New(c.String("paths-server") + c.String("paths-uri") + ": " + resp.Status)
	}

	// Save response in buffer
	var buf bytes.Buffer
	buf.ReadFrom(resp.Body)
	resp.Body.Close()

	// Un gzip
	gzipReader, err := gzip.NewReader(&buf)
	if err != nil {
		return err
	}

	log.Println("Got paths, start writing to DB")

	// Read lines from file
	budReader := bufio.NewReader(gzipReader)
	// begin a transaction
	for i := 1; ; i++ {
		line, isPrefix, err := budReader.ReadLine()

		if err != nil {
			log.Println(err)
			break
		}

		if isPrefix {
			panic("URL in paths list too long")
		}

		task := Task{URL: c.String("paths-server") + string(line)}

		// Retry
		action := func(uint) error {
			res := db.Where(&task).FirstOrCreate(&task, &task)
			if res.Error != nil {
				log.Println(res.Error)
			}
			return res.Error
		}
		if err := retry.Retry(breaker.BreakByTimeout(5*time.Minute), action, strategy.Backoff(backoff.Exponential(time.Second, 1.2))); err != nil {
			panic(err)
		}

		//Output
		if i%10 == 0 {
			fmt.Print(".")
		}

		if i%1000 == 0 {
			fmt.Println()
			log.Println("Wrote " + strconv.Itoa(i))
		}
	}
	// commit the transaction

	fmt.Println()
	log.Println("Done")

	return nil
}
