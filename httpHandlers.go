package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
)

func logoutHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	// delete cookie?
	// invalidate session in DB

}

func randStringGenerator(length int) string {

	chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	chars = chars + strings.ToLower(chars)

	b := strings.Builder{}

	for i := 0; i < length; i++ {
		idx := rand.Int31n(int32(len(chars)) - 1)
		b.WriteByte(chars[idx])
	}

	return b.String()
}

func userCreate(username, password string) {

	salt := randStringGenerator(10)
	passHashed := sha256.Sum256([]byte(password + salt))
	passHashedHex := hex.EncodeToString(passHashed[:])

	q := "INSERT INTO sec_users (username, password_hash, salt) VALUES (?, ?, ?)"
	_, err := db.Exec(q, username, passHashedHex, salt)
	if err != nil {
		log.Print(err)
	}
}

func authHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	// User POSTing auth credentials
	var err error

	req.ParseForm()

	username := req.Form.Get("username")
	password := req.Form.Get("password")

	var salt string
	q := "SELECT salt FROM sec_users WHERE username = ?"
	err = db.QueryRow(q, username).Scan(&salt)
	if err != nil {
		log.Print(err)
		w.Header().Set("Location", "/")
		w.WriteHeader(http.StatusMovedPermanently)
		return
	}

	passHashed := sha256.Sum256([]byte(password + salt))

	passHashedHex := hex.EncodeToString(passHashed[:])

	log.Printf("User '%s' requested authentication with password '%s', with salt '%s', and hash '%s'", username, password, salt, passHashedHex)

	var userID int
	q = "SELECT id FROM sec_users WHERE username = ? AND password_hash = ?"
	err = db.QueryRow(q, username, passHashedHex).Scan(&userID)
	if err != nil {
		log.Print(err)
	}

	if userID > 0 {
		log.Printf("User '%s' logged in successfully", username)

		// Generate UUID for login cookie

		sessionID := uuid.New().String()

		q = "INSERT INTO sec_sessions (uuid, user_id) VALUES (?, ?)"
		_, err = db.Exec(q, sessionID, userID)
		if err != nil {
			log.Print(err)
		}

		expire := time.Now().Add(time.Hour * 8)
		cookie := &http.Cookie{
			Name:    "sessionID",
			Value:   sessionID,
			Expires: expire,
		}
		http.SetCookie(w, cookie)

		w.Header().Set("Location", fmt.Sprintf("/app/%d", userID))
		w.WriteHeader(http.StatusMovedPermanently)
	} else {
		w.Header().Set("Location", "/")
		w.WriteHeader(http.StatusMovedPermanently)
	}

}

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

	sessionCookie, err := req.Cookie("sessionID")
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if sessionCookie != nil {
		//q := "SELECT id FROM sec_sessions WHERE uuid = ? AND user_id = ?"
		var id int
		q := "SELECT id FROM sec_sessions WHERE uuid = ?"
		err = db.QueryRow(q, sessionCookie.Value).Scan(&id)
		if err != nil {
			if err == sql.ErrNoRows {
				log.Print("session missing!?!")
				w.Header().Set("Location", "/")
				w.WriteHeader(http.StatusMovedPermanently)
				return
			}
		}
	}

	data := WOData{}

	q := "SELECT id, name FROM workorders WHERE id IN (SELECT wo_id FROM wo_assignments WHERE user_id = ?) ORDER BY name DESC"
	rows, err := db.Query(q, userID)
	if err != nil {
		log.Fatal(err)
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
