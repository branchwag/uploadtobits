package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, filepath.Join(".", "index.html"))
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20) //10mb limit
	if err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Unable to get file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	out, err := os.Create(filepath.Join(".", "uploaded_file"))
	if err != nil {
		http.Error(w, "Unable to create file", http.StatusInternalServerError)
		return
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		http.Error(w, "Unable to save file", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte("<p>File uploaded!<p>"))
}

func VisualizeBinary(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	for i := 0; i < len(data); i += 16 {
		end := i + 16
		if end > len(data) {
			end = len(data)
		}

		//print hex
		for j := i; j < end; j++ {
			buf.WriteString(fmt.Sprintf("%02x ", data[j]))
		}

		//print ascii
		buf.WriteString(" | ")
		for j := i; j < end; j++ {
			if data[j] >= 32 && data[j] <= 126 {
				buf.WriteString(fmt.Sprintf("%c", data[j]))
			} else {
				buf.WriteString(".")
			}
		}
		buf.WriteString("\n")
	}

	return buf.String(), nil
}

func vizHandler(w http.ResponseWriter, r *http.Request) {
	filepath :=  "./adicon.jpg" // later use r.URL.Query().Get("file")
	if filepath == "" {
		http.Error(w, "Missing 'file' query parameter", http.StatusBadRequest)
		return
	}

	output, err := VisualizeBinary(filepath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error visualizing binary file: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text-plain")
	w.Write([]byte(output))
}

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/visualize", vizHandler)
	
	fmt.Println("Starting server on :4242")

	if err := http.ListenAndServe(":4242", nil); err != nil {
		log.Fatalf("Could not start server: %s\n", err.Error())
	}
}
