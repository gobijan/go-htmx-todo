package main

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/olahol/melody"
)

type App struct {
	todoService TodoCrud
	m           *melody.Melody
	indexTmpl   *template.Template
	renameTmpl  *template.Template
	assets      embed.FS
}

type ToDo struct {
	ID    int
	Title string
	Done  bool
}

type TodoCrud interface {
	All() []ToDo
	Find(id int) (ToDo, error)
	Add(todo ToDo)
	Toggle(id int)
	Delete(id int)
	Rename(id int, title string)
	Clear()
	ClearCompleted()
	OpenTodos() []ToDo
	CompletedTodos() []ToDo
}

type TodoService struct {
	TodoCrud
	sync.Mutex
	TodoList []ToDo
	ToDoID   int
}

func (t *TodoService) OpenTodos() []ToDo {
	t.Lock()
	defer t.Unlock()
	openTodos := []ToDo{}
	for _, todo := range t.TodoList {
		if !todo.Done {
			openTodos = append(openTodos, todo)
		}
	}
	return openTodos
}

func (t *TodoService) CompletedTodos() []ToDo {
	t.Lock()
	defer t.Unlock()
	completedTodos := []ToDo{}
	for _, todo := range t.TodoList {
		if todo.Done {
			completedTodos = append(completedTodos, todo)
		}
	}
	return completedTodos
}

func (t *TodoService) All() []ToDo {
	t.Lock()
	defer t.Unlock()
	// make a copy of the list
	todoList := make([]ToDo, len(t.TodoList))
	copy(todoList, t.TodoList)
	// reverse the copy
	slices.Reverse(todoList)
	return todoList
}

func (t *TodoService) Find(id int) (todo ToDo, err error) {
	t.Lock()
	defer t.Unlock()
	for _, todo := range t.TodoList {
		if todo.ID == id {
			return todo, nil
		}
	}
	return todo, fmt.Errorf("todo with id %d not found", id)
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

func (t *TodoService) Clear() {
	t.Lock()
	defer t.Unlock()
	t.TodoList = []ToDo{}
}

func (t *TodoService) ClearCompleted() {
	t.Lock()
	defer t.Unlock()
	var incomplete []ToDo
	for _, todo := range t.TodoList {
		if !todo.Done {
			incomplete = append(incomplete, todo)
		}
	}
	t.TodoList = incomplete
}

var (
	//go:embed templates/*
	templates embed.FS

	//go:embed assets/*
	assets embed.FS
)

func main() {
	indexTmpl := template.Must(template.ParseFS(templates, "templates/application.go.html", "templates/index.go.html"))
	renameTmpl := template.Must(template.ParseFS(templates, "templates/application.go.html", "templates/rename.go.html"))

	app := &App{
		todoService: &TodoService{},
		m:           melody.New(),
		indexTmpl:   indexTmpl,
		renameTmpl:  renameTmpl,
		assets:      assets,
	}

	http.HandleFunc("/", app.IndexHandler)
	http.HandleFunc("/add", app.AddHandler)
	http.HandleFunc("/toggle", app.ToggleHandler)
	http.HandleFunc("/delete", app.DeleteHandler)
	http.HandleFunc("/showrename", app.ShowRenameHandler)
	http.HandleFunc("/rename", app.RenameHandler)
	http.HandleFunc("/clear", app.ClearHandler)
	http.HandleFunc("/clearcompleted", app.ClearCompletedHandler)
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
	completedTodos := a.todoService.CompletedTodos()
	completedTodosCount := len(completedTodos)

	data := struct {
		Timestamp           int64
		TodoList            []ToDo
		CompletedTodosCount int
	}{
		Timestamp:           time.Now().Unix(),
		TodoList:            a.todoService.All(),
		CompletedTodosCount: completedTodosCount,
	}

	// Execute the template with the data
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

func (a *App) ShowRenameHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("show rename")

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	iid, err := strconv.Atoi(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get Todo by id
	todo, err := a.todoService.Find(iid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Anonymously embed the id in the template data
	data := struct {
		Timestamp int64
		ID        string
		Todo      ToDo
	}{
		Timestamp: time.Now().Unix(),
		ID:        id,
		Todo:      todo,
	}

	err = a.renameTmpl.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (a *App) RenameHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("rename")
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
	title := r.FormValue("title")
	a.todoService.Rename(id, title)
	a.m.Broadcast([]byte("update"))
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) ClearHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("clear")
	a.todoService.Clear()
	a.m.Broadcast([]byte("update"))
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) ClearCompletedHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("clear completed")
	a.todoService.ClearCompleted()
	a.m.Broadcast([]byte("update"))
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	if err := a.m.HandleRequest(w, r); err != nil {
		log.Println("WebSocketHandler error:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
