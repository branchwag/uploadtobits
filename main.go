package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

var (
	printableColor = color.RGBA{0, 255, 0, 255} // Green for printable characters
	dotColor        = color.RGBA{0, 0, 0, 255}   // Black for dots (non-printable characters)
)

func createImageFromBinary(data []byte) (*image.RGBA, error) {
	width, height := 256, 256
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			index := y*width + x
			if index >= len(data) {
				break
			}

			char := data[index]
			var col color.Color

			if char >= 32 && char <= 126 { // Printable ASCII range
				col = printableColor
			} else { // Non-printable
				col = dotColor
			}

			img.Set(x, y, col)
		}
	}

	return img, nil
}

// Function to create a hex and ASCII representation of binary data
func createTextFromBinary(data []byte) (string, error) {
	var buf bytes.Buffer
	for i := 0; i < len(data); i += 16 {
		end := i + 16
		if end > len(data) {
			end = len(data)
		}

		// Print hex
		for j := i; j < end; j++ {
			buf.WriteString(fmt.Sprintf("%02x ", data[j]))
		}

		// Print ASCII
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

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, filepath.Join(".", "index.html"))
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
    err := r.ParseMultipartForm(10 << 20) // 10MB limit
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

    fileName := "uploaded_file" // Use a unique name for each file in a real application
    filePath := filepath.Join(".", fileName)

    out, err := os.Create(filePath)
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
	fmt.Printf("File saved to: %s\n", filePath) // Debug statement

    // Redirect to the visualization page with the file parameter
    http.Redirect(w, r, "/visualize?file="+fileName, http.StatusSeeOther)
}



func vizHandler(w http.ResponseWriter, r *http.Request) {
    fileName := r.URL.Query().Get("file")
    if fileName == "" {
        http.Error(w, "Missing 'file' query parameter", http.StatusBadRequest)
        return
    }

    filePath := filepath.Join(".", fileName)

    if _, err := os.Stat(filePath); os.IsNotExist(err) {
        http.Error(w, fmt.Sprintf("File not found: %v", err), http.StatusNotFound)
        return
    }

    data, err := os.ReadFile(filePath)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error reading file: %v", err), http.StatusInternalServerError)
        return
    }

    img, err := createImageFromBinary(data)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error generating image: %v", err), http.StatusInternalServerError)
        return
    }

    textOutput, err := createTextFromBinary(data)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error generating text output: %v", err), http.StatusInternalServerError)
        return
    }

    var imgBuf bytes.Buffer
    if err := png.Encode(&imgBuf, img); err != nil {
        http.Error(w, fmt.Sprintf("Error encoding image: %v", err), http.StatusInternalServerError)
        return
    }
    imgBase64 := "data:image/png;base64," + encodeToBase64(imgBuf.Bytes())

    html := fmt.Sprintf(`
    <!DOCTYPE html>
    <html>
    <head>
        <title>File Visualization</title>
    </head>
    <body>
        <h1>File Visualization</h1>
        <h2>Image Representation</h2>
        <img src="%s" alt="Image Representation"/>
        <h2>Text Representation</h2>
        <pre>%s</pre>
    </body>
    </html>
    `, imgBase64, textOutput)

	fmt.Printf("Visualizing file: %s\n", filePath) // Debug statement


    w.Header().Set("Content-Type", "text/html")
    w.Write([]byte(html))
}



func encodeToBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
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
