package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/julienschmidt/httprouter"

	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

func main() {

	var err error

	rand.Seed(time.Now().UnixNano())

	log.SetFlags(log.Lshortfile | log.LstdFlags)

	mysqlPass := os.Getenv("MYSQL_SECRET")

	db, err = sql.Open("mysql", fmt.Sprintf("root:%s@tcp(127.0.0.1:3306)/squishy?parseTime=true", mysqlPass))
	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Print("Successfully opened database squishy\n")
	}

	router := httprouter.New()

	router.GET("/", indexHandler)

	router.GET("/app/:userid", appHandler)

	router.GET("/resources/*filepath", resourceHandler)

	router.GET("/wo/:woID", woReadHandler)
	router.POST("/wo", woCreateHandler)

	router.POST("/auth", authHandler)

	router.GET("/logout", logoutHandler)

	log.Fatal(http.ListenAndServe(":8080", router))

}
