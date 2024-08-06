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
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type App struct {
	todoService TodoCrud
	m           *melody.Melody
	indexTmpl   *template.Template
	renameTmpl  *template.Template
	assets      embed.FS
	db          *gorm.DB
}

type ToDo struct {
	gorm.Model
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

type TodoServiceDB struct {
	TodoCrud
	db *gorm.DB
}

func (t *TodoServiceDB) All() []ToDo {
	var todos []ToDo
	t.db.Order("id desc").Find(&todos)
	return todos
}

func (t *TodoServiceDB) Find(id int) (ToDo, error) {
	var todo ToDo
	result := t.db.First(&todo, id)
	if result.Error != nil {
		return ToDo{}, result.Error
	}
	return todo, nil
}

func (t *TodoServiceDB) Add(todo ToDo) {
	t.db.Create(&todo)
}

func (t *TodoServiceDB) Toggle(id int) {
	var todo ToDo
	t.db.First(&todo, id)
	todo.Done = !todo.Done
	t.db.Save(&todo)
}

func (t *TodoServiceDB) Delete(id int) {
	t.db.Delete(&ToDo{}, id)
}

func (t *TodoServiceDB) Rename(id int, title string) {
	var todo ToDo
	t.db.First(&todo, id)
	todo.Title = title
	t.db.Save(&todo)
}

func (t *TodoServiceDB) Clear() {
	t.db.Exec("DELETE FROM to_dos")
}

func (t *TodoServiceDB) ClearCompleted() {
	t.db.Where("done = ?", true).Delete(&ToDo{})
}

func (t *TodoServiceDB) OpenTodos() []ToDo {
	var todos []ToDo
	t.db.Where("done = ?", false).Find(&todos)
	return todos
}

func (t *TodoServiceDB) CompletedTodos() []ToDo {
	var todos []ToDo
	t.db.Where("done = ?", true).Find(&todos)
	return todos
}

var (
	//go:embed templates/*
	templates embed.FS

	//go:embed assets/*
	assets embed.FS
)

func main() {
	indexTmpl := template.Must(template.ParseFS(templates, "templates/application.go.html", "templates/index.go.html"))
	renameTmpl := template.Must(template.ParseFS(templates, "templates/rename.go.html"))

	db, err := gorm.Open(sqlite.Open("database.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	err = db.AutoMigrate(&ToDo{})
	if err != nil {
		panic("failed to migrate database")
	}

	app := &App{
		todoService: &TodoServiceDB{db: db},
		m:           melody.New(),
		indexTmpl:   indexTmpl,
		renameTmpl:  renameTmpl,
		assets:      assets,
		db:          db,
	}

	http.HandleFunc("/", app.IndexHandler)
	http.HandleFunc("POST /add", app.AddHandler)
	http.HandleFunc("PATCH /toggle", app.ToggleHandler)
	http.HandleFunc("DELETE /delete", app.DeleteHandler)
	http.HandleFunc("GET /showrename", app.ShowRenameHandler)
	http.HandleFunc("PATCH /rename", app.RenameHandler)
	http.HandleFunc("POST /clear", app.ClearHandler)
	http.HandleFunc("POST /clearcompleted", app.ClearCompletedHandler)
	http.HandleFunc("/ws", app.WebSocketHandler)
	http.Handle("/assets/", app.AssetFileHandler())
	log.Println("Server running at http://localhost:8080")
	err = http.ListenAndServe("localhost:8080", nil)
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
