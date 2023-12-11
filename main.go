package main

import (
	_ "embed"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/olahol/melody"
)

type ToDo struct {
	ID    int
	Title string
	Done  bool
}

var (
	todoID   int
	todoList []ToDo

	//go:embed index.html
	indexHTML string

	indexTmpl = template.Must(template.New("index").Parse(indexHTML))
	m         = melody.New()
)

func main() {
	http.HandleFunc("/", IndexHandler)
	http.HandleFunc("/add", AddHandler)
	http.HandleFunc("/toggle", ToggleHandler)
	http.HandleFunc("/ws", WebSocketHandler)
	err := http.ListenAndServe("localhost:8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	err := indexTmpl.Execute(w, todoList)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func AddHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	title := r.FormValue("title")
	todoList = append(todoList, ToDo{ID: todoID, Title: title})
	todoID++
	m.Broadcast([]byte("update"))
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func ToggleHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("toggle")
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	id := r.FormValue("id")
	for i, todo := range todoList {
		if strconv.Itoa(todo.ID) == id {
			todoList[i].Done = !todo.Done
			break
		}
	}
	m.Broadcast([]byte("update"))
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	m.HandleRequest(w, r)
}
