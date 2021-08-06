package db

import (
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"

	//"google.golang.org/appengine/log"
	log "github.com/sirupsen/logrus"
)

func ConnectToDb(dbConfig mysql.Config) (*gorm.DB, error) {

	dbConfig.ParseTime = true

	redact := func(r rune) rune {
		return '*'
	}

	redactedPw := strings.Map(redact, dbConfig.Passwd)

	log.Debugf("Connection String (pw redacted): %s:%s@/%s", dbConfig.User, redactedPw, dbConfig.Addr)

	connectionString := dbConfig.FormatDSN()
	db, err := gorm.Open("mysql", connectionString)
	if err != nil {
		return nil, fmt.Errorf("Unable to connect to the database: %s", err.Error())
	}
	return db, err
}
