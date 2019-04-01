package main

import (
	"net/http"
	//"fmt"
	"io"
	"log"
	"os"

	"github.com/datatogether/warc"
	"github.com/urfave/cli"
	//"github.com/davecgh/go-spew/spew"
)

func parse(*cli.Context) error {
	log.SetFlags(log.LstdFlags)
	log.Println("Started")

	// Open file local
	f, err := os.Open("C:\\workspace\\commoncrawl\\CC-MAIN-20190215183319-20190215205319-00000.warc.gz")
	if err != nil {
		return err
	}
	defer f.Close()

    // Open file remote
    resp, err := http.Get("https://commoncrawl.s3.amazonaws.com/crawl-data/CC-MAIN-2019-09/segments/1550247479101.30/warc/CC-MAIN-20190215183319-20190215205319-00000.warc.gz")
    if err != nil {
        return err
    }
    defer resp.Body.Close()

	log.Println("File opened")

	// Create reader from file
	//rdr, err := warc.NewReader(f)
	rdr, err := warc.NewReader(resp.Body)
	if err != nil {
		return err
	}
	log.Println("Reader created")

	for i :=0; ; i++ {
		rec, err := rdr.Read()
		if err == io.EOF {
			break
		}
		if i % 10000 == 0 {
			log.Println(i, rec.Type)
		}
	}

	// Get records from reaader

	// records, err := rdr.ReadAll()
	// if err != nil {
	// 	return err
	// }
	// log.Println("All readed")

	// for i, rec := range records {
	// 	fmt.Println(i)
	// 	fmt.Println(rec.Type)
	// }

	// rec := records[2]
	// fmt.Println(rec.Type)

	// body, err := rec.Body()
	// if err != nil {
	// 	return err
	// }
	// fmt.Println(rec.Type)
	// fmt.Println(body)

	return nil
}
