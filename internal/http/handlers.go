package http

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/parse"
	"github.com/rs/zerolog/log"
)

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	// TODO(robherley): cute landing page
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("✂️\n"))
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Healthy\n"))
}

func FileHandler(cfg *config.Config, database *db.DB, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fileID := strings.TrimPrefix(r.URL.Path, "/f/")

		file := db.File{}
		if err := database.First(&file, "id = ?", fileID).Error; err != nil {
			log.Error().Err(err).Msg("unable to lookup file")
			http.NotFound(w, r)
			return
		}

		if file.Private {
			log.Warn().Msg("attempted to access private file")
			http.NotFound(w, r)
		}

		if shouldSendRaw(r) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write(file.Content)
			return
		}

		out, err := parse.File(&file)
		if err != nil {
			log.Error().Err(err).Msg("unable to parse file")
			http.Error(w, "unable to parse file", http.StatusInternalServerError)
			return
		}

		tmpl.ExecuteTemplate(w, "file.go.html", map[string]interface{}{
			"FileID":    file.ID,
			"FileSize":  humanize.Bytes(uint64(file.Size)),
			"CreatedAt": humanize.Time(file.CreatedAt),
			"FileType":  out.FileType,
			"HTML":      out.HTML,
			"CSS":       out.CSS,
		})
	}
}

func shouldSendRaw(r *http.Request) bool {
	if isCurl := strings.Contains(r.Header.Get("user-agent"), "curl"); isCurl {
		return true
	}

	if _, hasRawParam := r.URL.Query()["r"]; hasRawParam {
		return true
	}

	return false
}