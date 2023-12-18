package main

import (
	"testing"
)

func TestTodoService(t *testing.T) {
	// Create a new instance of TodoService
	service := &TodoService{}

	// Test adding a new todo
	t.Run("AddTodo", func(t *testing.T) {
		// Add a new todo
		service.Add(ToDo{
			Title: "Buy groceries",
			Done:  false,
		})

		// Verify that the todo was added
		if len(service.TodoList) != 1 {
			t.Errorf("Expected TodoList length to be 1, got %d", len(service.TodoList))
		}

		// Verify the added todo's properties
		if service.TodoList[0].ID != 0 {
			t.Errorf("Expected todo ID to be 1, got %d", service.TodoList[0].ID)
		}
		if service.TodoList[0].Title != "Buy groceries" {
			t.Errorf("Expected todo Text to be 'Buy groceries', got '%s'", service.TodoList[0].Title)
		}
		if service.TodoList[0].Done != false {
			t.Errorf("Expected todo Done to be false, got %t", service.TodoList[0].Done)
		}
	})

	// Add more test cases for other methods of TodoService...
}
