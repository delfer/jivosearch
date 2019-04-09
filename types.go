package main

import (
	"time"
)

type Task struct {
	URL        string `gorm:"PRIMARY_KEY;UNIQUE_INDEX"`
	WorkerName string
	Started    *time.Time
	Completed  *time.Time
}

type AnySite struct {
	URL        string
	Body       []byte
	SourceWarc string
}

type TargetSite struct {
	URL        string
	Body       string
	WidgetID   string
	SourceWarc string
}

type JivoSite struct {
	URL        string `gorm:"PRIMARY_KEY;UNIQUE_INDEX"`
	Checked    bool
	WidgetID   string
	CYC        int
	Comment    string
	Found      *time.Time
	WorkerName string
	SourceWarc string
}
