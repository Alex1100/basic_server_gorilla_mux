package routers

import (
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	gmux "github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/url"
)

type Page struct {
	Books []Book
}

type SearchResult struct {
	Title  string `xml:"title,attr"`
	Author string `xml:"author,attr"`
	Year   string `xml:"hyr,attr"`
	ID     string `xml:"owi,attr"`
}

type Book struct {
	PK             int
	Title          string
	Author         string
	Classification string
}

type ClassifySearchResponse struct {
	Results []SearchResult `xml:"works>work"`
}

type ClassifyBookResponse struct {
	BookData struct {
		Title  string `xml:"title,attr"`
		Author string `xml:"author,attr"`
		ID     string `xml:"owi,attr"`
	} `xml:"work"`
	Classification struct {
		MostPopular string `xml:"sfa,attr"`
	} `xml:"recommendations>ddc>mostPopular"`
}

func find(id string) (ClassifyBookResponse, error) {
	var c ClassifyBookResponse
	body, err := classifyAPI("http://classify.oclc.org/classify2/Classify?summary=true&owi=" + url.QueryEscape(id))

	if err != nil {
		fmt.Printf("ERROR IS: %s", err)
		return ClassifyBookResponse{}, err
	}

	err = xml.Unmarshal(body, &c)
	return c, err
}

func search(query string) ([]SearchResult, error) {
	var c ClassifySearchResponse
	body, err := classifyAPI("http://classify.oclc.org/classify2/Classify?summary=true&title=" + url.QueryEscape(query))

	if err != nil {
		fmt.Printf("ERROR IS: %s", err)
		return []SearchResult{}, err
	}

	err = xml.Unmarshal(body, &c)
	return c.Results, err
}

func classifyAPI(url string) ([]byte, error) {
	var resp *http.Response
	var err error

	if resp, err = http.Get(url); err != nil {
		fmt.Printf("ERROR IS: %s", err)
		return []byte{}, err
	}

	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

type Routes struct {
	db *sql.DB
}

func NewRoutes(db *sql.DB) *Routes {
	return &Routes{db: db}
}

func (database *Routes) RootRoute(w http.ResponseWriter, r *http.Request) {
	templates := template.Must(template.ParseFiles("templates/index.html"))

	p := Page{Books: []Book{}}

	rows, err := database.db.Query("SELECT pk, title, author, classification FROM books")

	if err != nil {
		fmt.Printf("ERROR IS: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	for rows.Next() {
		var b Book
		rows.Scan(&b.PK, &b.Title, &b.Author, &b.Classification)
		p.Books = append(p.Books, b)
	}

	if err := templates.ExecuteTemplate(w, "index.html", p); err != nil {
		fmt.Printf("ERROR IS: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	return
}

func (database *Routes) SortBooks(w http.ResponseWriter, r *http.Request) {
	p := Page{Books: []Book{}}

	columnName := r.FormValue("sortBy")

	if columnName != "title" && columnName != "author" && columnName != "classification" {
		http.Error(w, "Invalid Column Name", http.StatusBadRequest)
		return
	}

	rows, err := database.db.Query("SELECT pk, title, author, classification FROM books ORDER BY " + r.FormValue("sortBy"))

	if err != nil {
		fmt.Printf("ERROR IS: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	for rows.Next() {
		var b Book
		rows.Scan(&b.PK, &b.Title, &b.Author, &b.Classification)
		p.Books = append(p.Books, b)
	}

	if err := json.NewEncoder(w).Encode(p.Books); err != nil {
		fmt.Printf("ERROR IS: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	return
}

func (database *Routes) SearchBooks(w http.ResponseWriter, r *http.Request) {
	var results []SearchResult
	var err error

	if results, err = search(r.FormValue("search")); err != nil {
		fmt.Printf("ERROR IS: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	encoder := json.NewEncoder(w)

	if err := encoder.Encode(results); err != nil {
		fmt.Printf("ERROR IS: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	return
}

func (database *Routes) InsertBook(w http.ResponseWriter, r *http.Request) {
	var book ClassifyBookResponse
	var err error

	if book, err = find(r.FormValue("id")); err != nil {
		fmt.Printf("ERROR IS: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err = database.db.Ping(); err != nil {
		fmt.Printf("ERROR IS: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	result, err := database.db.Exec("INSERT INTO books(title, author, id, classification) values($1, $2, $3, $4)",
		book.BookData.Title, book.BookData.Author, book.BookData.ID, book.Classification.MostPopular)

	if err != nil {
		fmt.Printf("ERROR IS: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	pk, _ := result.LastInsertId()

	b := Book{
		PK:             int(pk),
		Title:          book.BookData.Title,
		Author:         book.BookData.Author,
		Classification: book.Classification.MostPopular,
	}

	if err := json.NewEncoder(w).Encode(b); err != nil {
		fmt.Printf("ERROR IS: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	return
}

func (database *Routes) DeleteBook(w http.ResponseWriter, r *http.Request) {
	if _, err := database.db.Exec("DELETE FROM books WHERE books.pk = $1", gmux.Vars(r)["pk"]); err != nil {
		fmt.Printf("ERROR IS: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	return
}
