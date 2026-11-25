package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"vynza-backend/internal/config"
	"vynza-backend/internal/handler"
	"vynza-backend/internal/middleware"
	"vynza-backend/internal/model"
	"vynza-backend/internal/repository"
)

func main() {
	cfg := config.Load()

	db, err := repository.NewDB(cfg.DBDriver, cfg.DBUrl)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	h := handler.New(cfg, db)

	mux := http.NewServeMux()

	// Static uploads directory serving
	fsUploads := http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads")))
	mux.Handle("/uploads/", fsUploads)

	// API Router
	mux.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","app":"VYNZA Backend API","version":"1.0.0"}`))
	})

	// Auth routes
	mux.HandleFunc("/api/v1/auth/register", h.Register)
	mux.HandleFunc("/api/v1/auth/login", h.Login)
	mux.Handle("/api/v1/auth/me", middleware.Auth(cfg.JWTSecret, model.RoleUser)(http.HandlerFunc(h.Me)))

	// Catalog routes (public)
	mux.HandleFunc("/api/v1/artists", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.GetArtists(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/albums", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.GetAlbums(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/songs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.GetSongs(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/songs/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		// expected: ["api", "v1", "songs", "{id}"] or ["api", "v1", "songs", "{id}", "stream"]
		if len(parts) >= 4 {
			id, err := strconv.ParseInt(parts[3], 10, 64)
			if err != nil {
				http.Error(w, `{"error":"invalid song id"}`, http.StatusBadRequest)
				return
			}

			if len(parts) == 4 {
				if r.Method == http.MethodGet {
					h.GetSongByID(w, r, id)
				}
				return
			}

			if len(parts) == 5 && parts[4] == "stream" {
				if r.Method == http.MethodGet {
					h.StreamSong(w, r, id)
				}
				return
			}
		}
		http.NotFound(w, r)
	})

	// User Playlists & Favorites (protected by User auth)
	userAuth := middleware.Auth(cfg.JWTSecret, model.RoleUser)

	mux.Handle("/api/v1/playlists", userAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.GetPlaylists(w, r)
		} else if r.Method == http.MethodPost {
			h.CreatePlaylist(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})))

	mux.Handle("/api/v1/playlists/", userAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) >= 4 {
			pID, err := strconv.ParseInt(parts[3], 10, 64)
			if err != nil {
				http.Error(w, `{"error":"invalid playlist id"}`, http.StatusBadRequest)
				return
			}

			if len(parts) == 4 && r.Method == http.MethodGet {
				h.GetPlaylistDetails(w, r, pID)
				return
			}

			if len(parts) == 5 && parts[4] == "songs" && r.Method == http.MethodPost {
				h.AddSongToPlaylist(w, r, pID)
				return
			}

			if len(parts) == 6 && parts[4] == "songs" && r.Method == http.MethodDelete {
				sID, _ := strconv.ParseInt(parts[5], 10, 64)
				h.RemoveSongFromPlaylist(w, r, pID, sID)
				return
			}
		}
		http.NotFound(w, r)
	})))

	mux.Handle("/api/v1/favorites", userAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.GetFavorites(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})))

	mux.Handle("/api/v1/favorites/", userAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) == 4 {
			sID, err := strconv.ParseInt(parts[3], 10, 64)
			if err != nil {
				http.Error(w, `{"error":"invalid song id"}`, http.StatusBadRequest)
				return
			}
			if r.Method == http.MethodPost {
				h.AddFavorite(w, r, sID)
				return
			} else if r.Method == http.MethodDelete {
				h.RemoveFavorite(w, r, sID)
				return
			}
		}
		http.NotFound(w, r)
	})))

	// Admin routes (protected by Admin auth)
	adminAuth := middleware.Auth(cfg.JWTSecret, model.RoleAdmin)

	mux.Handle("/api/v1/admin/stats", adminAuth(http.HandlerFunc(h.GetAdminStats)))
	mux.Handle("/api/v1/admin/users", adminAuth(http.HandlerFunc(h.GetAdminUsers)))
	mux.Handle("/api/v1/admin/import", adminAuth(http.HandlerFunc(h.AdminImportArtist)))

	mux.Handle("/api/v1/admin/artists", adminAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.AdminCreateArtist(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})))

	mux.Handle("/api/v1/admin/albums", adminAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.AdminCreateAlbum(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})))

	mux.Handle("/api/v1/admin/songs", adminAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.AdminCreateSong(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})))

	mux.Handle("/api/v1/admin/songs/", adminAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) == 5 && parts[3] == "songs" {
			id, err := strconv.ParseInt(parts[4], 10, 64)
			if err != nil {
				http.Error(w, `{"error":"invalid song id"}`, http.StatusBadRequest)
				return
			}
			if r.Method == http.MethodDelete {
				h.AdminDeleteSong(w, r, id)
				return
			}
		}
		http.NotFound(w, r)
	})))

	handlerWithCORS := middleware.CORS(mux)

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("[VYNZA BACKEND] Server listening on %s (PID: %d)", addr, 8080)
	if err := http.ListenAndServe(addr, handlerWithCORS); err != nil {
		log.Fatalf("Server terminated: %v", err)
	}
}
