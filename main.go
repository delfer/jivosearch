package main

import (
  "log"
  "os"

  "github.com/urfave/cli"
)

func main() {

  app := cli.NewApp()

  app.Flags = []cli.Flag {
    cli.StringFlag{
      Name: "dialect",
      Value: "postgres",
      Usage: "mysql, postgres, mssql",
      EnvVar: "DB_DIALECT",
    },
    cli.StringFlag{
      Name: "db-connection",
      Value: "host=localhost port=5432 user=admin dbname=gorm password=mypassword sslmode=disable",
      Usage: "connection parameters",
      EnvVar: "DB_CONNECTION",
    },
    cli.StringFlag{
      Name: "name",
      Value: "anonymous",
      Usage: "hostname to write results",
      EnvVar: "HOSTNAME",
    },
    cli.StringFlag{
      Name: "paths",
      Value: "https://commoncrawl.s3.amazonaws.com/crawl-data/CC-MAIN-2019-13/warc.paths.gz",
      Usage: "url of Common Crawl gzipped paths list",
      EnvVar: "PATHS_URL",
    },
  }

  app.Commands = []cli.Command{
    {
      Name:    "schedule",
      Usage:   "create tasks schedule in DB",
      Action:  schedule,
    },
    {
      Name:    "parse",
      Usage:   "parse WARC files",
      Action:  parse,
    },
  }

  err := app.Run(os.Args)
  if err != nil {
    log.Fatal(err)
  }
}
