package main

import (
	"embed"
	_ "embed"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/olahol/melody"
)

type ToDo struct {
	ID    int
	Title string
	Done  bool
}

type TemplateData struct {
	Timestamp int64
	TodoList  []ToDo
}

var (
	todoID   int
	todoList []ToDo

	//go:embed index.go.html
	indexHTML string

	//go:embed assets/*
	assets embed.FS

	indexTmpl = template.Must(template.New("index").Parse(indexHTML))
	m         = melody.New()
)

func main() {
	http.HandleFunc("/", IndexHandler)
	http.HandleFunc("/add", AddHandler)
	http.HandleFunc("/toggle", ToggleHandler)
	http.HandleFunc("/delete", DeleteHandler)
	http.HandleFunc("/ws", WebSocketHandler)
	http.Handle("/assets/", AssetFileHandler())
	log.Println("Server running at http://localhost:8080")
	err := http.ListenAndServe("localhost:8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}

func AssetFileHandler() http.Handler {
	return http.FileServer(http.FS(assets))
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	data := TemplateData{
		Timestamp: time.Now().Unix(),
		TodoList:  todoList,
	}

	err := indexTmpl.Execute(w, data)
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

func DeleteHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("delete")
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	id := r.FormValue("id")
	for i, todo := range todoList {
		if strconv.Itoa(todo.ID) == id {
			todoList = append(todoList[:i], todoList[i+1:]...)
			break
		}
	}
	m.Broadcast([]byte("update"))
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	m.HandleRequest(w, r)
}
