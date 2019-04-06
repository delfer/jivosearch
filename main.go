package main

import (
	"log"
	"os"

	"github.com/urfave/cli"
)

func main() {

	app := cli.NewApp()

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "dialect",
			Value:  "postgres",
			Usage:  "mysql, postgres, mssql",
			EnvVar: "DB_DIALECT",
		},
		cli.StringFlag{
			Name:   "db-connection",
			Value:  "host=localhost port=5432 user=admin dbname=gorm password=mypassword sslmode=disable",
			Usage:  "connection parameters",
			EnvVar: "DB_CONNECTION",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:   "schedule",
			Usage:  "schedule tasks in DB",
			Action: schedule,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "paths-server",
					Value:  "https://commoncrawl.s3.amazonaws.com/",
					Usage:  "protocol and server of Common Crawl gzipped paths list",
					EnvVar: "PATHS_SERVER",
				},
				cli.StringFlag{
					Name:   "paths-uri",
					Value:  "crawl-data/CC-MAIN-2019-13/warc.paths.gz",
					Usage:  "uri of Common Crawl gzipped paths list",
					EnvVar: "PATHS_URI",
				},
			},
		},
		{
			Name:   "parse",
			Usage:  "parse WARC files",
			Action: parse,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "name",
					Value:  "anonymous",
					Usage:  "hostname to write results",
					EnvVar: "HOSTNAME",
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
