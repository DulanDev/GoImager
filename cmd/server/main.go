package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/DulanHewage/GoImager/internal/handler"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
			log.Fatalf("Error loading .env file")
	}
	
	r := mux.NewRouter()

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("GoImager"))
	}).Methods("GET")

	r.HandleFunc("/resize", handler.ResizeHandler).Methods("POST")
	r.HandleFunc("/convert", handler.ConvertHandler).Methods("POST")
	r.HandleFunc("/optimize", handler.OptimizeHandler).Methods("POST")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Server is running on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}