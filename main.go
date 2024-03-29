package main

import (
	"io"
	"log"
	"net/http"
	"os"
)

// -------------- Globals --------------
var uploadKey string

// -------------- Main --------------
func main() {
	ip := os.Getenv("IP_ADDRESS")
	if ip == "" {
		ip = "0.0.0.0"
	}
	port := os.Getenv("REST_PORT")
	if port == "" {
		port = "3004"
	}
	uploadKey = os.Getenv("UPLOAD_KEY")

	router := http.NewServeMux()

	router.Handle("/", http.FileServer(http.Dir("./static")))
	router.HandleFunc("GET /upload", uploadPageHandler)
	router.HandleFunc("POST /upload", uploadHandler)

	server := http.Server{
		Addr:    ip + ":" + port,
		Handler: router,
	}
	log.Fatal(server.ListenAndServe())
}

// -------------- Handlers --------------
// Upload page handler
func uploadPageHandler(w http.ResponseWriter, r *http.Request) {
	html := `
	<!DOCTYPE html>
	<html>
		<head>
			<title>Upload</title>
		</head>
		<body>
			<form action="/upload" method="post" enctype="multipart/form-data">
				<label for="upload_key">Upload Key:</label>
				<input type="text" name="upload_key" required/>
				<br>
				<label for="upload_path">Upload Path:</label>
				<input type="text" name="upload_path" />
				<br>
				<input type="file" name="file" required/>
				<br>
				<input type="submit" value="Upload" />
			</form>
		</body>
	</html>
	`
	w.Write([]byte(html))
}

// Upload handler
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("upload_key") != uploadKey {
		http.Error(w, "Invalid upload key", http.StatusUnauthorized)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	path := r.FormValue("upload_path")
	for len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}
	if _, err := os.Stat("/cdn/" + path); os.IsNotExist(err) {
		os.MkdirAll("/cdn/"+path, os.ModePerm)
	}

	dst, err := os.Create("/cdn/" + path + "/" + handler.Filename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("File uploaded successfully"))
}
