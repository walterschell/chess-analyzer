package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"sync"
	"text/template"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/walterschell/chess-analyzer/chessanalysis"
)

const DefaultPort = 8080

//go:embed assets/static/*
//go:embed assets/static/pieces/*
//go:embed assets/templates/*
var assets embed.FS
var static fs.FS
var templates fs.FS

func init() {
	var err error
	static, err = fs.Sub(assets, "assets/static")
	if err != nil {
		panic(fmt.Sprintf("Failed to load static files: %v", err))
	}
	templates, err = fs.Sub(assets, "assets/templates")
	if err != nil {
		panic(fmt.Sprintf("Failed to load templates: %v", err))
	}
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

type Message struct {
	Type  string `json:"type"`
	PGN   string `json:"pgn,omitempty"`
	Text  string `json:"text,omitempty"`
	Depth int    `json:"depth,omitempty"`
}

func NewApplication() *Application {
	templateParser := template.New("")
	templateParser.Delims("[[", "]]")

	app := &Application{
		router:    mux.NewRouter(),
		templates: template.Must(templateParser.ParseFS(templates, "*.html.gotmpl")),
		clients:   make(map[*Client]interface{}),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}

	app.router.NotFoundHandler = stdoutLogger(http.HandlerFunc(notFoundHandler))
	app.router.Use(stdoutLogger)

	// Create a custom file server that sets the correct content type for PGN files
	fileServer := http.FileServer(http.FS(static))
	app.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("[Static] Serving file: %s\n", r.URL.Path)

		// Set content type for PGN files before serving
		if strings.HasSuffix(r.URL.Path, ".pgn") {
			w.Header().Set("Content-Type", "text/plain")
		}

		// Let the standard file server handle the request
		fileServer.ServeHTTP(w, r)
	})))

	app.router.HandleFunc("/", app.indexHandler)
	app.router.HandleFunc("/ws", app.wsHandler)

	return app
}

func (app *Application) indexHandler(w http.ResponseWriter, r *http.Request) {
	templateVars := struct {
		Title string
	}{
		Title: "Chess Game Analyzer",
	}

	err := app.templates.ExecuteTemplate(w, "index.html.gotmpl", templateVars)
	if err != nil {
		fmt.Printf("Error rendering template: %v\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (app *Application) wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := app.upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	fmt.Printf("New websocket connection from %s\n", conn.RemoteAddr())
	client := &Client{
		conn:        conn,
		application: app,
	}
	app.clientsLock.Lock()
	app.clients[client] = nil
	app.clientsLock.Unlock()

	go func() {
		for {
			_, messageBytes, err := client.conn.ReadMessage()
			if err != nil {
				fmt.Printf("Error reading message: %v\n", err)
				app.clientsLock.Lock()
				delete(app.clients, client)
				app.clientsLock.Unlock()
				client.conn.Close()
				return
			}

			var message Message
			if err := json.Unmarshal(messageBytes, &message); err != nil {
				fmt.Printf("Error parsing message: %v\n", err)
				continue
			}

			if message.Type == "analyze" {
				// Use default depth of 5 if not specified
				depth := message.Depth
				if depth <= 0 {
					depth = 5
				}
				if depth > 30 {
					depth = 30
				}

				// Start streaming analysis
				movesChan, errChan := chessanalysis.AnalyzeChessGameStreaming(message.PGN, chessanalysis.WithDepth(depth))

				// Process moves as they come in
				go func() {
					for move := range movesChan {
						if move == nil {
							continue
						}

						// Convert analysis to JSON
						analysisJSON, err := json.Marshal(move)
						if err != nil {
							fmt.Printf("Error marshaling analysis: %v\n", err)
							continue
						}

						// Send analysis to client
						response := Message{
							Type: "analysis",
							Text: string(analysisJSON),
						}
						if err := client.conn.WriteJSON(response); err != nil {
							fmt.Printf("Error sending analysis: %v\n", err)
							return
						}
					}

					// Check for any errors from the analysis
					if err := <-errChan; err != nil {
						response := Message{
							Type: "analysis",
							Text: fmt.Sprintf("Analysis error: %v", err),
						}
						client.conn.WriteJSON(response)
					}
				}()
			}
		}
	}()
}

func (app *Application) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	app.router.ServeHTTP(w, r)
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "File Not Found", http.StatusNotFound)
}

func stdoutLogger(next http.Handler) http.Handler {
	return handlers.LoggingHandler(os.Stdout, next)
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

	http.ListenAndServe(fmt.Sprintf(":%d", port), app)
}
