package main

import (
	"fmt"
	"github.com/gorilla/context"
	"github.com/gorilla/sessions"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

const (
	sessionName = "sicuro-auth"
)

var (
	SessionSecret string
	SessionStore  *sessions.CookieStore
	Templates     *template.Template
	Upgrader      websocket.Upgrader
	AppDIR, port  string
)

func init() {
	err := godotenv.Load(".env")

	if err != nil {
		log.Fatalf("Error loading .env file")
	}
	AppDIR = filepath.Join(os.Getenv("ROOT_DIR"), "app")
	port = os.Getenv("PORT")
	SessionSecret = os.Getenv("SESSION_SECRET")
	SessionStore = sessions.NewCookieStore([]byte(SessionSecret))
	Templates = template.Must(template.ParseFiles(fetchTemplates()...))
	Upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
}

func main() {
	setupGithubOAuth()
	registerRoutes()

	fmt.Printf("Starting server on port: %s\n", port)
	err := http.ListenAndServe(fmt.Sprintf(":%s", port), context.ClearHandler(http.DefaultServeMux))
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
