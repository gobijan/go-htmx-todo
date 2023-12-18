package main

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/olahol/melody"
)

type App struct {
	todoService TodoCrud
	m           *melody.Melody
	indexTmpl   *template.Template
	assets      embed.FS
}

type ToDo struct {
	ID    int
	Title string
	Done  bool
}

type TemplateData struct {
	Timestamp int64
	TodoList  []ToDo
}

type TodoCrud interface {
	All() []ToDo
	Add(todo ToDo)
	Toggle(id int)
	Delete(id int)
	Rename(id int, title string)
}

type TodoService struct {
	TodoCrud
	sync.Mutex
	TodoList []ToDo
	ToDoID   int
}

func (t *TodoService) All() []ToDo {
	t.Lock()
	defer t.Unlock()
	return t.TodoList
}

func (t *TodoService) Add(todo ToDo) {
	t.Lock()
	defer t.Unlock()
	todo.ID = t.ToDoID
	t.TodoList = append(t.TodoList, todo)
	t.ToDoID++
}

func (t *TodoService) Toggle(id int) {
	t.Lock()
	defer t.Unlock()
	for i, todo := range t.TodoList {
		if todo.ID == id {
			t.TodoList[i].Done = !todo.Done
			break
		}
	}
}

func (t *TodoService) Delete(id int) {
	t.Lock()
	defer t.Unlock()
	for i, todo := range t.TodoList {
		if todo.ID == id {
			t.TodoList = append(t.TodoList[:i], t.TodoList[i+1:]...)
			break
		}
	}
}

func (t *TodoService) Rename(id int, title string) {
	t.Lock()
	defer t.Unlock()
	for i, todo := range t.TodoList {
		if todo.ID == id {
			t.TodoList[i].Title = title
			break
		}
	}
}

var (
	//go:embed templates/*
	templates embed.FS

	//go:embed assets/*
	assets embed.FS
)

func main() {
	indexTmpl := template.Must(template.ParseFS(templates, "templates/index.go.html"))

	app := &App{
		todoService: &TodoService{},
		m:           melody.New(),
		indexTmpl:   indexTmpl,
		assets:      assets,
	}

	http.HandleFunc("/", app.IndexHandler)
	http.HandleFunc("/add", app.AddHandler)
	http.HandleFunc("/toggle", app.ToggleHandler)
	http.HandleFunc("/delete", app.DeleteHandler)
	http.HandleFunc("/ws", app.WebSocketHandler)
	http.Handle("/assets/", app.AssetFileHandler())
	log.Println("Server running at http://localhost:8080")
	err := http.ListenAndServe("localhost:8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}

func (a *App) AssetFileHandler() http.Handler {
	return http.FileServer(http.FS(a.assets))
}

func (a *App) IndexHandler(w http.ResponseWriter, r *http.Request) {
	data := TemplateData{
		Timestamp: time.Now().Unix(),
		TodoList:  a.todoService.All(),
	}

	err := a.indexTmpl.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (a *App) AddHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	title := r.FormValue("title")
	a.todoService.Add(ToDo{Title: title})
	a.m.Broadcast([]byte("update"))
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) ToggleHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("toggle")
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sid := r.FormValue("id")
	id, err := strconv.Atoi(sid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.todoService.Toggle(id)
	a.m.Broadcast([]byte("update"))
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("delete")
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sid := r.FormValue("id")
	id, err := strconv.Atoi(sid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.todoService.Delete(id)
	a.m.Broadcast([]byte("update"))
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	a.m.HandleRequest(w, r)
}
