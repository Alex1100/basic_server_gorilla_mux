package main

import (
	"basic_server_gorilla_mux/routers"
	"database/sql"
	"fmt"
	gmux "github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"log"
	"net/http"
)

const (
	host     = "elmer.db.elephantsql.com"
	port     = 5432
	user     = "htldhvag"
	password = ""
	dbname   = "htldhvag"
)

func main() {

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	fmt.Println(psqlInfo)

	db, err := sql.Open("postgres", psqlInfo)

	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	mux := gmux.NewRouter()

	routes := routers.NewRoutes(db)

	mux.HandleFunc("/login", routes.Login).Methods("GET")
	mux.HandleFunc("/", routes.RootRoute).Methods("GET")
	mux.HandleFunc("/search", routes.SearchBooks).Methods("POST")
	mux.HandleFunc("/books", routes.SortBooks).Methods("GET")
	mux.HandleFunc("/books", routes.InsertBook).Methods("PUT")
	mux.HandleFunc("/books/{pk}", routes.DeleteBook).Methods("DELETE")

	fmt.Println(http.ListenAndServe(":8080", mux))
}
