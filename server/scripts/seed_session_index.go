package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"kefu-server/models"
)

func main() {
	dbPath := flag.String("db", "", "sqlite db path")
	total := flag.Int("n", 1000, "session rows to seed")
	flag.Parse()

	if *dbPath == "" {
		log.Fatal("db path required")
	}
	if *total <= 0 {
		log.Fatal("n must be > 0")
	}

	db, err := gorm.Open(sqlite.Open(*dbPath), &gorm.Config{})
	if err != nil {
		log.Fatalf("open sqlite failed: %v", err)
	}

	if err := db.AutoMigrate(&models.SessionListIndex{}); err != nil {
		log.Fatalf("auto migrate failed: %v", err)
	}
	if err := db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.SessionListIndex{}).Error; err != nil {
		log.Fatalf("clear session_list_indexes failed: %v", err)
	}

	now := time.Now().Unix()
	statuses := []string{
		models.SessionStatusUnAssigned,
		models.SessionStatusAssigned,
		models.SessionStatusUnRead,
		models.SessionStatusUnReply,
		models.SessionStatusClosed,
	}

	batch := make([]models.SessionListIndex, 0, 1000)
	for i := 1; i <= *total; i++ {
		appID := fmt.Sprintf("app-%02d", i%20)
		status := statuses[i%len(statuses)]
		agentID := ""
		if i%7 == 0 {
			agentID = "admin"
			status = models.SessionStatusAssigned
		} else if status != models.SessionStatusUnAssigned {
			agentID = fmt.Sprintf("agent-%02d", i%40)
		}

		lastActive := now - int64(i%86400)
		createdAt := lastActive - int64(i%7200)
		lastVisitor := lastActive - int64(i%1200)
		lastReply := lastActive - int64(i%600)
		if status == models.SessionStatusUnRead || status == models.SessionStatusUnReply || status == models.SessionStatusUnAssigned {
			lastVisitor = lastActive
			lastReply = lastActive - int64(100+i%300)
		}
		unread := 0
		if status == models.SessionStatusUnRead || status == models.SessionStatusUnReply {
			unread = (i % 5) + 1
		}

		row := models.SessionListIndex{
			SID:                fmt.Sprintf("s:visitor-%08d:%s:%010d", i, appID, i),
			VisitorID:          fmt.Sprintf("visitor-%08d", i),
			AppID:              appID,
			Status:             status,
			CurAgentID:         agentID,
			LastClientIP:       fmt.Sprintf("10.0.%d.%d", i%255, (i*3)%255),
			LastUserAgent:      "benchmark-agent",
			LastDevice:         "desktop",
			LastGeo:            "CN",
			LastVisitorMsgTime: lastVisitor,
			LastAgentReplyTime: lastReply,
			LastAgentReadTime:  lastReply,
			CreatedAt:          createdAt,
			LastActiveAt:       lastActive,
			UnreadCount:        unread,
			LastMessage:        fmt.Sprintf("benchmark message %d", i),
			LastMessageType:    models.WSContentTypeText,
			UpdatedAt:          now,
		}
		batch = append(batch, row)
		if len(batch) >= 1000 {
			if err := db.CreateInBatches(batch, 1000).Error; err != nil {
				log.Fatalf("seed insert failed: %v", err)
			}
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		if err := db.CreateInBatches(batch, 1000).Error; err != nil {
			log.Fatalf("seed insert failed: %v", err)
		}
	}

	fmt.Printf("seeded session_list_indexes rows=%d db=%s\n", *total, *dbPath)
}
