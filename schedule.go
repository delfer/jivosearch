package main

import (
	"strconv"
	"bufio"
	"compress/gzip"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mssql"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
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

	// Un gzip
	gzipReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}

	log.Println("Got paths, start writing to DB")

	// Read lines from file
	budReader := bufio.NewReader(gzipReader)
	// begin a transaction
	tx := db.Begin()
	for i := 1; ; i++ {
		line, isPrefix, err := budReader.ReadLine()

		if err != nil {
			break
		}

		if isPrefix {
			panic("URL in paths list too long")
		}

		task := Task{URL: c.String("paths-server") + string(line)}
		tx.Where(&task).FirstOrCreate(&task, &task)

		if i%100 == 0 {
			fmt.Print(".")
		}

		if i%1000 == 0 {
			// split the transaction
			tx.Commit()
			tx = db.Begin()
			fmt.Println()
			log.Println("Wrote "+ strconv.Itoa(i))
		}
	}
	// commit the transaction
	tx.Commit()

	log.Println("Done")

	return nil
}
