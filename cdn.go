package main

import (
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/cors"
)

// WrappedWriter - Wrapper for http.ResponseWriter
type WrappedWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader - Write the header
func (w *WrappedWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
	w.statusCode = statusCode
}

// Middleware - Middleware type
type Middleware func(http.Handler) http.Handler

// CreateStack - Create a stack of middlewares
func CreateStack(middlewares ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}

// RequestLoggerMiddleware - Log all requests
func RequestLoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &WrappedWriter{w, http.StatusOK}
		next.ServeHTTP(wrapped, r)

		cfConnectingIP := r.Header.Get("CF-Connecting-IP")
		forwardedFor := r.Header.Get("X-Forwarded-For")
		if cfConnectingIP != "" {
			r.RemoteAddr = cfConnectingIP
		} else if forwardedFor != "" {
			r.RemoteAddr = forwardedFor
		}

		log.Printf("%s %d %s %s %s", r.RemoteAddr, wrapped.statusCode, r.Method, r.URL.Path, time.Since(start))
	})
}

var uploadKey string = os.Getenv("UPLOAD_KEY")

// CDNServer - The web server
type CDNServer struct {
	Address  string
	UsingUDS bool
}

// NewCDNServer - Create a new API server
func NewCDNServer(address string, usingUDS bool) *CDNServer {
	return &CDNServer{
		Address:  address,
		UsingUDS: usingUDS,
	}
}

// Setup - Setup the cdn server
func (s *CDNServer) Setup() http.Handler {
	router := http.NewServeMux()

	router.Handle("/", http.FileServer(http.Dir("/cdn")))
	router.HandleFunc("GET /upload", UploadPageHandler)
	router.HandleFunc("POST /upload", UploadHandler)

	return CreateStack(
		RequestLoggerMiddleware,
		cors.AllowAll().Handler,
	)(router)
}

// Run - Start the web server
func (s *CDNServer) Run() error {
	server := http.Server{
		Addr:    s.Address,
		Handler: s.Setup(),
	}

	if s.UsingUDS {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-c
			os.Remove(s.Address)
			os.Exit(1)
		}()

		if _, err := os.Stat(s.Address); err == nil {
			log.Printf("Removing existing socket file %s", s.Address)
			if err := os.Remove(s.Address); err != nil {
				return err
			}
		}

		socket, err := net.Listen("unix", s.Address)
		if err != nil {
			return err
		}

		log.Printf("WebServer listening on %s", s.Address)
		return server.Serve(socket)
	} else {
		log.Printf("WebServer listening on %s", s.Address)
		return server.ListenAndServe()
	}
}

// UploadPageHandler - Upload page handler
func UploadPageHandler(w http.ResponseWriter, r *http.Request) {
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

// UploadHandler - Upload handler
func UploadHandler(w http.ResponseWriter, r *http.Request) {
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
