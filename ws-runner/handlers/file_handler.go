package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func CustomFileHandler(dir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		enableCors(&w)

		recursive := r.URL.Query().Get("recursive") == "true"
		requestPath := filepath.Clean(r.URL.Path)

		if requestPath == "." {
			requestPath = ""
		}

		fullPath := filepath.Join(dir, requestPath)

		if !strings.HasPrefix(fullPath, filepath.Clean(dir)) {
			sendErrorResponse(w, "Access denied", http.StatusForbidden, "Path outside allowed directory")
			return
		}

		stat, err := os.Stat(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				sendErrorResponse(w, "File or directory not found", http.StatusNotFound, err.Error())
			} else {
				sendErrorResponse(w, "Unable to access path", http.StatusInternalServerError, err.Error())
			}
			return
		}

		if stat.IsDir() {
			if recursive {
				handleRecursiveFileListing(w, r, fullPath, requestPath)
			} else {
				handleDirectoryListing(w, r, fullPath, requestPath)
			}
		} else {
			handleFileContent(w, r, fullPath, requestPath)
		}
	}
}

// ... (other file handler functions like handleRecursiveFileListing, handleDirectoryListing, etc.)
