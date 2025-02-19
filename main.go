package main

import (
	"embed"
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"golang.org/x/net/websocket"
)

//go:embed web/templates/* README.md
var content embed.FS

// renderMarkdown converts markdown content to HTML with GitHub-like styling
func renderMarkdown(md []byte) template.HTML {
	// Create markdown parser with extensions
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs
	p := parser.NewWithExtensions(extensions)

	// Parse markdown
	doc := p.Parse(md)

	// Create HTML renderer with options
	opts := html.RendererOptions{
		Flags: html.CommonFlags | html.HrefTargetBlank,
	}
	renderer := html.NewRenderer(opts)

	// Render to HTML
	html := markdown.Render(doc, renderer)
	return template.HTML(html)
}

// Server maintains the set of active websocket clients and handles incoming webhooks
type Server struct {
	clients        map[*websocket.Conn]bool
	broadcast      chan []byte
	register       chan *websocket.Conn
	unregister     chan *websocket.Conn
	forwardAddress string
	smtpUrl        string
	fromEmail      string
}

func newServer(forwardAddress, smtpUrl, fromEmail string) *Server {
	return &Server{
		broadcast:      make(chan []byte),
		register:       make(chan *websocket.Conn),
		unregister:     make(chan *websocket.Conn),
		clients:        make(map[*websocket.Conn]bool),
		forwardAddress: forwardAddress,
		smtpUrl:        smtpUrl,
		fromEmail:      fromEmail,
	}
}

func (s *Server) run() {
	for {
		select {
		case client := <-s.register:
			s.clients[client] = true
		case client := <-s.unregister:
			if _, ok := s.clients[client]; ok {
				delete(s.clients, client)
				client.Close()
			}
		case message := <-s.broadcast:
			for client := range s.clients {
				err := websocket.Message.Send(client, string(message))
				if err != nil {
					log.Printf("Error sending message to client: %v", err)
					client.Close()
					delete(s.clients, client)
				}
			}
		}
	}
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	log.Printf("[Request] /")

	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Read README content
	readmeBytes, err := content.ReadFile("README.md")
	if err != nil {
		log.Printf("Error reading README: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFS(content, "web/templates/index.html")
	if err != nil {
		log.Printf("Error parsing template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := struct {
		ForwardAddress string
		SmtpUrl        string
		FromEmail      string
		ReadmeContent  template.HTML
	}{
		ForwardAddress: s.forwardAddress,
		SmtpUrl:        s.smtpUrl,
		FromEmail:      s.fromEmail,
		ReadmeContent:  renderMarkdown(readmeBytes),
	}

	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

	log.Printf("[Done] /")
}

func (s *Server) handleWebSocket(ws *websocket.Conn) {
	log.Printf("[Register] /ws")

	s.register <- ws

	defer func() {
		s.unregister <- ws
		log.Printf("[Unregister] /ws")
	}()

	// Keep the connection alive
	for {
		var msg string
		if err := websocket.Message.Receive(ws, &msg); err != nil {
			break
		}
	}

	log.Printf("[Done] /ws")
}

func (s *Server) handleEmails(w http.ResponseWriter, r *http.Request) {
	log.Printf("[Request] /emails")

	// Read the raw data from the request body
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	log.Printf("Email:\n```\n%s\n```", string(data))

	// Parse the email payload using inline struct
	var email struct {
		Headers struct {
			Subject string `json:"subject"`
		} `json:"headers"`
	}
	if err := json.Unmarshal(data, &email); err != nil {
		log.Printf("Error parsing email JSON: %v", err)
		http.Error(w, "Error parsing email", http.StatusBadRequest)
		return
	}

	// Broadcast the raw data directly
	s.broadcast <- data

	// TODO: Send reply email using SMTP
	log.Printf("Would send reply email using SMTP URL: %s", s.smtpUrl)

	// Create and send response
	response := struct {
		Status  string `json:"status"`
		Subject string `json:"subject"`
	}{
		Status:  "ok",
		Subject: email.Headers.Subject,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}

	log.Printf("[Done] /emails")
}

func main() {
	// Load configuration from environment variables
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	forwardAddress := os.Getenv("CLOUDMAILIN_FORWARD_ADDRESS")
	smtpUrl := os.Getenv("CLOUDMAILIN_SMTP_URL")
	fromEmail := os.Getenv("FROM_EMAIL")
	// Create and start the server
	server := newServer(forwardAddress, smtpUrl, fromEmail)
	go server.run()

	// Create new servemux
	mux := http.NewServeMux()

	// Set up routes
	mux.HandleFunc("GET /", server.handleHome)
	mux.Handle("GET /ws", websocket.Handler(server.handleWebSocket))
	mux.HandleFunc("POST /emails", server.handleEmails)

	// Start the server
	log.Printf("Starting server on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
