package main

import (
	"net/url"
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mssql"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/urfave/cli"
	//	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

type Task struct {
	gorm.Model
	TaskID     int `gorm:"AUTO_INCREMENT"`
	Url        *url.URL
	Started    *time.Time
	Completed  *time.Time
	WorkerName string
}

func schedule(c *cli.Context) error {
	db, err := gorm.Open(c.GlobalString("dialect"), c.GlobalString("db-connection"))
	if err != nil {
		return err
	}
	defer db.Close()

	db.AutoMigrate(&Task{})

	fmt.Print("HW!")

	return nil
}
