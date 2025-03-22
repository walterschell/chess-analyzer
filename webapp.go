package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"sync"
	"text/template"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

const DefaultPort = 8080

//go:embed assets
var assets embed.FS
var static fs.FS
var templates fs.FS

func init() {
	static, _ = fs.Sub(assets, "assets/static")
	templates, _ = fs.Sub(assets, "assets/templates")
}

func stdoutLogger(next http.Handler) http.Handler {
	return handlers.LoggingHandler(os.Stdout, next)
}

type Client struct {
	conn        *websocket.Conn
	application *Application
}

type Application struct {
	router      *mux.Router
	templates   *template.Template
	clients     map[*Client]interface{}
	clientsLock sync.RWMutex
	upgrader    websocket.Upgrader
}

func NewApplication() *Application {
	templateParser := template.New("")
	templateParser.Delims("[[", "]]")
	result := Application{
		router:    mux.NewRouter(),
		templates: template.Must(templateParser.ParseFS(templates, "*.html.gotmpl")),
		clients:   make(map[*Client]interface{}),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
	result.router.NotFoundHandler = stdoutLogger(http.HandlerFunc(notFoundHandler))
	result.router.Use(stdoutLogger)

	result.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.FS(static))))
	result.router.HandleFunc("/", result.indexHandler)
	result.router.HandleFunc("/ws", result.wsHandler)
	return &result
}

func (app *Application) indexHandler(w http.ResponseWriter, r *http.Request) {
	templateVars := struct {
		Title string
	}{
		Title: "Hello, World!",
	}

	err := app.templates.ExecuteTemplate(w, "index.html.gotmpl", templateVars)
	if err != nil {
		fmt.Printf("Error rendering template: %v\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

}

func (application *Application) wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := application.upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	fmt.Printf("New websocket connection from %s\n", conn.RemoteAddr())
	client := &Client{
		conn:        conn,
		application: application,
	}
	application.clientsLock.Lock()
	application.clients[client] = nil
	application.clientsLock.Unlock()
	client.conn.WriteMessage(websocket.TextMessage, []byte("Welcome!"))
	go func() {
		for {
			_, messageJson, err := client.conn.ReadMessage()
			if err != nil {
				fmt.Printf("Error reading message: %v\n", err)
				application.clientsLock.Lock()
				delete(application.clients, client)
				application.clientsLock.Unlock()
				client.conn.Close()
				return
			}
			fmt.Printf("Received message: %s\n", messageJson)
			var message struct {
				Message string `json:"message"`
			}
			err = json.Unmarshal(messageJson, &message)
			if err != nil {
				fmt.Printf("Error parsing message: %v\n", err)
				continue
			}
			application.broadcast(message.Message)
		}
	}()
}

func (app *Application) broadcast(message string) {
	fmt.Printf("Broadcasting message: %s\n", message)
	app.clientsLock.RLock()
	defer app.clientsLock.RUnlock()
	for client := range app.clients {
		client.conn.WriteMessage(websocket.TextMessage, []byte(message))
	}
}

func (app *Application) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	app.router.ServeHTTP(w, r)
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "File Not Found", http.StatusNotFound)
}

func main() {
	var port uint
	flag.UintVar(&port, "port", DefaultPort, "Port to listen on")
	flag.Parse()
	if port == 0 || port > 65535 {
		fmt.Println("Invalid port number")
		os.Exit(1)
	}
	fmt.Printf("Starting server on :%d\n", port)
	app := NewApplication()

	// Remove this and replace with your own logic
	go func() {
		for {
			app.broadcast("Hello, World!")
			time.Sleep(5 * time.Second)
		}
	}()

	http.ListenAndServe(fmt.Sprintf(":%d", port), app)
}
