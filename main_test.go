package main

import (
	"errors"
	"testing"
)

type TodoServiceMemory struct {
	todos  map[int]ToDo
	nextID int
	TodoCrud
}

func NewTodoServiceMemory() *TodoServiceMemory {
	return &TodoServiceMemory{
		todos:  make(map[int]ToDo),
		nextID: 1,
	}
}

func (t *TodoServiceMemory) All() []ToDo {
	var todos []ToDo
	for _, todo := range t.todos {
		todos = append(todos, todo)
	}
	return todos
}

func (t *TodoServiceMemory) Find(id int) (ToDo, error) {
	todo, exists := t.todos[id]
	if !exists {
		return ToDo{}, errors.New("todo not found")
	}
	return todo, nil
}

func (t *TodoServiceMemory) Add(todo ToDo) {
	todo.ID = t.nextID
	t.nextID++
	t.todos[todo.ID] = todo
}

func (t *TodoServiceMemory) Toggle(id int) {
	todo, exists := t.todos[id]
	if exists {
		todo.Done = !todo.Done
		t.todos[id] = todo
	}
}

func (t *TodoServiceMemory) Delete(id int) {
	delete(t.todos, id)
}

func (t *TodoServiceMemory) Rename(id int, title string) {
	todo, exists := t.todos[id]
	if exists {
		todo.Title = title
		t.todos[id] = todo
	}
}

func (t *TodoServiceMemory) Clear() {
	t.todos = make(map[int]ToDo)
	t.nextID = 1
}

func (t *TodoServiceMemory) ClearCompleted() {
	for id, todo := range t.todos {
		if todo.Done {
			delete(t.todos, id)
		}
	}
}

func (t *TodoServiceMemory) OpenTodos() []ToDo {
	var todos []ToDo
	for _, todo := range t.todos {
		if !todo.Done {
			todos = append(todos, todo)
		}
	}
	return todos
}

func (t *TodoServiceMemory) CompletedTodos() []ToDo {
	var todos []ToDo
	for _, todo := range t.todos {
		if todo.Done {
			todos = append(todos, todo)
		}
	}
	return todos
}

func TestTodoService(t *testing.T) {
	// Create a new instance of TodoService
	service := NewTodoServiceMemory()

	// Test adding a new todo
	t.Run("AddTodo", func(t *testing.T) {
		// Add a new todo
		service.Add(ToDo{
			Title: "Buy groceries",
			Done:  false,
		})

		// Verify that the todo was added
		if len(service.All()) != 1 {
			t.Errorf("Expected TodoList length to be 1, got %d", len(service.All()))
		}

		// Verify the added todo's properties
		if service.All()[0].ID != 1 {
			t.Errorf("Expected todo ID to be 1, got %d", service.All()[0].ID)
		}
		if service.All()[0].Title != "Buy groceries" {
			t.Errorf("Expected todo Text to be 'Buy groceries', got '%s'", service.All()[0].Title)
		}
		if service.All()[0].Done != false {
			t.Errorf("Expected todo Done to be false, got %t", service.All()[0].Done)
		}
	})

	// Add more test cases for other methods of TodoService...
}
