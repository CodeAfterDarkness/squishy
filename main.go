package main

import (
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"text/template"

	"github.com/julienschmidt/httprouter"
)

func main() {

	log.SetFlags(log.Lshortfile | log.LstdFlags)

	router := httprouter.New()

	router.GET("/", indexHandler)

	router.GET("/app/:username", appHandler)

	router.GET("/resources/*filepath", resourceHandler)

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

type AppData struct {
	Username string
}

func appHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	fileBytes, err := ioutil.ReadFile("pages/app.html")
	if err != nil {
		log.Print(err)
		return
	}

	data := AppData{
		Username: ps.ByName("username"),
	}

	t, err := template.New("foo").Parse(string(fileBytes))
	if err != nil {
		log.Print(err)
	}

	err = t.Execute(w, data)
	if err != nil {
		log.Print(err)
	}

}
