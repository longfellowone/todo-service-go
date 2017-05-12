package controllers

import (
	"net/http"
	"github.com/golang/protobuf/proto"
	"todo-service-go/events"
	"todo-service-go/repositories"
	"todo-service-go/commands"
	"log"
	"github.com/gorilla/mux"
	"github.com/gocql/gocql"
	"github.com/nats-io/go-nats"
)

type TodoController struct {
	CommonController
	todoRepository *repositories.TodoRepository
}

func NewTodoController(natsSession *nats.Conn, todoRepository *repositories.TodoRepository) *TodoController {
	return &TodoController{
		CommonController{natsSession},
		todoRepository,
	}
}

func (c *TodoController) Index(w http.ResponseWriter, req *http.Request) {
	todos := c.todoRepository.GetTodos()

	c.SendJSON(
		w,
		req,
		todos,
		http.StatusOK,
	)
}

func (c *TodoController) RemoveTodo(w http.ResponseWriter, req *http.Request)  {
	vars := mux.Vars(req)
	id := vars["todo_id"]

	uuid, err := gocql.ParseUUID(id)
	if err != nil {
		log.Println(err)
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	event := events.TodoRemoved{
		Id: uuid.String(),
	}

	data, err := proto.Marshal(&event)
	if err != nil {
		http.Error(w, "Unable to add todo", http.StatusBadRequest)
		return
	}

	c.natsSession.Publish("todos.remove", data)

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"ok": true}`))
}

func (c *TodoController) GetTodo(w http.ResponseWriter, req *http.Request)  {
	vars := mux.Vars(req)
	id := vars["todo_id"]

	uuid, err := gocql.ParseUUID(id)
	if err != nil {
		log.Println(err)
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	todo := c.todoRepository.GetTodo(uuid)
	if todo != nil {
		c.SendJSON(
			w,
			req,
			todo,
			http.StatusOK,
		)
	} else {
		http.Error(w, "Todo not found", http.StatusNotFound)
	}
}

func (c *TodoController) AddTodo(w http.ResponseWriter, req *http.Request)  {
	defer req.Body.Close()

	command := &commands.AddTodo{}
	err := c.DecodeAndValidate(req, command)
	if err != nil {
		log.Println(err)
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	event := events.TodoAdded{
		Title: command.Title,
		Completed: command.Completed,
	}

	data, err := proto.Marshal(&event)
	if err != nil {
		http.Error(w, "Unable to add todo", http.StatusBadRequest)
		return
	}

	c.natsSession.Publish("todos.new", data)

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"ok": true}`))
}
