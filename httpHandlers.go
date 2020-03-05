package main

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"strconv"
	"text/template"

	"github.com/julienschmidt/httprouter"
)

type WorkOrder struct {
	ID           int
	OrderName    string
	CustomerName string
	CustomerID   int
	Summary      string
	Details      string
}

type WOData struct {
	WorkOrders struct {
		WorkOrderItems []WorkOrder
	}
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

func appHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	fileBytes, err := ioutil.ReadFile("pages/app.html")
	if err != nil {
		log.Print(err)
		return
	}

	userID, err := strconv.Atoi(ps.ByName("userid"))
	if err != nil {
		log.Printf("invalid userid %v", ps.ByName("userid"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Printf("User %d requested app", userID)

	data := WOData{}

	q := "SELECT id, name FROM workorders WHERE id IN (SELECT wo_id FROM wo_assignments WHERE user_id = ?) ORDER BY name DESC"
	rows, err := db.Query(q, userID)
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for rows.Next() {
		wo := WorkOrder{}

		err = rows.Scan(&wo.ID, &wo.OrderName)
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		data.WorkOrders.WorkOrderItems = append(data.WorkOrders.WorkOrderItems, wo)
	}

	t, err := template.New("app").Parse(string(fileBytes))
	if err != nil {
		log.Print(err)
	}

	err = t.Execute(w, data)
	if err != nil {
		log.Print(err)
	}
}

func woReadHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

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

func woCreateHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	// Accept JSON-encoded

	bodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer req.Body.Close()

	wo := WorkOrder{}

	log.Printf("Received: %s", string(bodyBytes))

	err = json.Unmarshal(bodyBytes, &wo)
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var q string
	var customerID int64

	q = "SELECT id FROM customers WHERE name = ?"
	row := db.QueryRow(q, wo.CustomerName)
	err = row.Scan(&customerID)
	if err != nil {
		if err == sql.ErrNoRows {
			q = "INSERT INTO customers (name) VALUES (?)"
			rs, err := db.Exec(q, wo.CustomerName)
			if err != nil {
				log.Print(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			customerID, err = rs.LastInsertId()
			if err != nil {
				log.Print(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		} else {
			log.Print(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	q = "INSERT INTO workorders (name, customer_id, summary, details) VALUES (?,?,?,?)"
	rs, err := db.Exec(q, wo.OrderName, customerID, wo.Summary, wo.Details)
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_ = rs

}
