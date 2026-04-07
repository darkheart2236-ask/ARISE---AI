package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	r := mux.NewRouter()

	// Serve CSS and JS from the "static" folder
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Routes
	r.HandleFunc("/", indexHandler).Methods("GET")

	port := os.Getenv("PORT")
	if port == "" { port = "8080" }

	fmt.Printf("ARISE AI launched on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "templates/index.html")
}
