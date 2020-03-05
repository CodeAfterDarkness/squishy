package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"os"
	"strconv"
	"text/template"

	"github.com/julienschmidt/httprouter"

	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

func main() {

	var err error

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

	router.GET("/app/", appHandler)

	router.GET("/resources/*filepath", resourceHandler)

	router.GET("/wo/:woID", woHandler)

	log.Fatal(http.ListenAndServe(":8080", router))

}

func indexHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {

	fileBytes, err := ioutil.ReadFile("pages/login.html")
	if err != nil {
		log.Print(err)
		return
	}

	w.Write(fileBytes)
}

func resourceHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	filePath := "resources" + ps.ByName("filepath")

	//log.Printf("Received request for file: '%s'", filePath)

	fileBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Print(err)
		return
	}

	//log.Printf("Read '%s' of size %d", filePath, len(fileBytes))

	mimeType := mime.TypeByExtension(filePath)

	w.Header().Set("Content-Type", mimeType)

	w.Write(fileBytes)

}

type WorkOrder struct {
	ID   int
	Name string
}

type WOData struct {
	WorkOrders struct {
		WorkOrderItems []WorkOrder
	}
}

func appHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	fileBytes, err := ioutil.ReadFile("pages/app.html")
	if err != nil {
		log.Print(err)
		return
	}

	data := WOData{}

	q := "SELECT id, name FROM workorders ORDER BY name DESC"
	rows, err := db.Query(q)
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for rows.Next() {
		wo := WorkOrder{}

		err = rows.Scan(&wo.ID, &wo.Name)
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		data.WorkOrders.WorkOrderItems = append(data.WorkOrders.WorkOrderItems, wo)
	}

	// data.WorkOrders.WorkOrderItems = []WorkOrder{
	// 	{Name: "test0", ID: 0},
	// 	{Name: "test2", ID: 2},
	// 	{Name: "test4", ID: 4},
	// }

	t, err := template.New("app").Parse(string(fileBytes))
	if err != nil {
		log.Print(err)
	}

	err = t.Execute(w, data)
	if err != nil {
		log.Print(err)
	}

}

func woHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	woID, err := strconv.Atoi(ps.ByName("woID"))
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Printf("User requested work order %d", woID)

	var data struct {
		ID           int
		CustomerName string
		OrderName    string
		Summary      string
		Details      string
	}

	data.ID = woID

	q := "SELECT customers.name, workorders.name, summary, details FROM workorders LEFT JOIN customers ON (workorders.customer_id = customers.id) WHERE workorders.id = ?"
	row := db.QueryRow(q, woID)

	err = row.Scan(&data.CustomerName, &data.OrderName, &data.Summary, &data.Details)
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonBytes)

}
