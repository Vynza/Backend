package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"vynza-backend/internal/config"
	"vynza-backend/internal/middleware"
	"vynza-backend/internal/model"
	"vynza-backend/internal/repository"
	"vynza-backend/internal/service"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	cfg  *config.Config
	repo *repository.DB
}

func New(cfg *config.Config, repo *repository.DB) *Handler {
	_ = os.MkdirAll(cfg.UploadAudio, 0755)
	_ = os.MkdirAll(cfg.UploadCovers, 0755)
	return &Handler{cfg: cfg, repo: repo}
}

func sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func sendError(w http.ResponseWriter, status int, msg string) {
	sendJSON(w, status, map[string]string{"error": msg})
}

// Auth Handlers
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if req.Username == "" || req.Email == "" || len(req.Password) < 6 {
		sendError(w, http.StatusBadRequest, "Validation error: email, username and password (min 6 chars) required")
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	user := &model.User{
		Username:  req.Username,
		Email:     req.Email,
		Password:  string(hashedPassword),
		Role:      model.RoleUser,
		AvatarURL: "https://images.unsplash.com/photo-1535713875002-d1d0cf377fde?w=200",
	}

	if err := h.repo.CreateUser(user); err != nil {
		sendError(w, http.StatusConflict, "Username or email already exists")
		return
	}

	token, err := h.generateToken(user)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to issue token")
		return
	}

	sendJSON(w, http.StatusCreated, map[string]interface{}{
		"token": token,
		"user":  user,
	})
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	user, err := h.repo.GetUserByEmail(req.Email)
	if err != nil {
		sendError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		sendError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	token, err := h.generateToken(user)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to issue token")
		return
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"token": token,
		"user":  user,
	})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*middleware.Claims)
	if !ok {
		sendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	user, err := h.repo.GetUserByID(claims.UserID)
	if err != nil {
		sendError(w, http.StatusNotFound, "User not found")
		return
	}

	sendJSON(w, http.StatusOK, user)
}

func (h *Handler) generateToken(user *model.User) (string, error) {
	claims := middleware.Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(72 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.cfg.JWTSecret))
}

// Catalog Handlers
func (h *Handler) GetSongs(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	genre := r.URL.Query().Get("genre")
	artistID, _ := strconv.ParseInt(r.URL.Query().Get("artist_id"), 10, 64)

	songs, err := h.repo.GetSongs(search, genre, artistID)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to fetch songs")
		return
	}
	sendJSON(w, http.StatusOK, songs)
}

func (h *Handler) GetSongByID(w http.ResponseWriter, r *http.Request, id int64) {
	song, err := h.repo.GetSongByID(id)
	if err != nil {
		sendError(w, http.StatusNotFound, "Song not found")
		return
	}
	sendJSON(w, http.StatusOK, song)
}

func (h *Handler) StreamSong(w http.ResponseWriter, r *http.Request, id int64) {
	song, err := h.repo.GetSongByID(id)
	if err != nil {
		sendError(w, http.StatusNotFound, "Song not found")
		return
	}

	h.repo.IncrementPlayCount(id)

	// If remote URL (HTTP/HTTPS), redirect or stream
	if strings.HasPrefix(song.AudioURL, "http://") || strings.HasPrefix(song.AudioURL, "https://") {
		http.Redirect(w, r, song.AudioURL, http.StatusTemporaryRedirect)
		return
	}

	// Local file streaming with HTTP Range support
	filePath := filepath.Join(h.cfg.UploadAudio, filepath.Base(song.AudioURL))
	file, err := os.Open(filePath)
	if err != nil {
		sendError(w, http.StatusNotFound, "Audio file not found on disk")
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Could not read audio file")
		return
	}

	http.ServeContent(w, r, stat.Name(), stat.ModTime(), file)
}

func (h *Handler) GetArtists(w http.ResponseWriter, r *http.Request) {
	artists, err := h.repo.GetArtists()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to fetch artists")
		return
	}
	sendJSON(w, http.StatusOK, artists)
}

func (h *Handler) GetAlbums(w http.ResponseWriter, r *http.Request) {
	albums, err := h.repo.GetAlbums()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to fetch albums")
		return
	}
	sendJSON(w, http.StatusOK, albums)
}

// Playlists Handlers
func (h *Handler) GetPlaylists(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value(middleware.UserContextKey).(*middleware.Claims)
	playlists, err := h.repo.GetUserPlaylists(claims.UserID)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to fetch playlists")
		return
	}
	sendJSON(w, http.StatusOK, playlists)
}

func (h *Handler) CreatePlaylist(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value(middleware.UserContextKey).(*middleware.Claims)
	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		CoverURL    string `json:"cover_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid payload")
		return
	}
	if req.Title == "" {
		sendError(w, http.StatusBadRequest, "Title is required")
		return
	}

	p := &model.Playlist{
		UserID:      claims.UserID,
		Title:       req.Title,
		Description: req.Description,
		CoverURL:    req.CoverURL,
	}
	if p.CoverURL == "" {
		p.CoverURL = "https://images.unsplash.com/photo-1514525253161-7a46d19cd819?w=500"
	}

	if err := h.repo.CreatePlaylist(p); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to create playlist")
		return
	}
	sendJSON(w, http.StatusCreated, p)
}

func (h *Handler) GetPlaylistDetails(w http.ResponseWriter, r *http.Request, id int64) {
	p, err := h.repo.GetPlaylistDetails(id)
	if err != nil {
		sendError(w, http.StatusNotFound, "Playlist not found")
		return
	}
	sendJSON(w, http.StatusOK, p)
}

func (h *Handler) AddSongToPlaylist(w http.ResponseWriter, r *http.Request, playlistID int64) {
	var req struct {
		SongID int64 `json:"song_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.SongID == 0 {
		sendError(w, http.StatusBadRequest, "Valid song_id is required")
		return
	}

	if err := h.repo.AddSongToPlaylist(playlistID, req.SongID); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to add song to playlist")
		return
	}
	sendJSON(w, http.StatusOK, map[string]string{"message": "Song added to playlist"})
}

func (h *Handler) RemoveSongFromPlaylist(w http.ResponseWriter, r *http.Request, playlistID, songID int64) {
	if err := h.repo.RemoveSongFromPlaylist(playlistID, songID); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to remove song")
		return
	}
	sendJSON(w, http.StatusOK, map[string]string{"message": "Song removed from playlist"})
}

// Favorites Handlers
func (h *Handler) GetFavorites(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value(middleware.UserContextKey).(*middleware.Claims)
	favs, err := h.repo.GetUserFavorites(claims.UserID)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to fetch favorites")
		return
	}
	sendJSON(w, http.StatusOK, favs)
}

func (h *Handler) AddFavorite(w http.ResponseWriter, r *http.Request, songID int64) {
	claims := r.Context().Value(middleware.UserContextKey).(*middleware.Claims)
	if err := h.repo.AddFavorite(claims.UserID, songID); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to add favorite")
		return
	}
	sendJSON(w, http.StatusOK, map[string]string{"message": "Song added to favorites"})
}

func (h *Handler) RemoveFavorite(w http.ResponseWriter, r *http.Request, songID int64) {
	claims := r.Context().Value(middleware.UserContextKey).(*middleware.Claims)
	if err := h.repo.RemoveFavorite(claims.UserID, songID); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to remove favorite")
		return
	}
	sendJSON(w, http.StatusOK, map[string]string{"message": "Song removed from favorites"})
}

// Admin Handlers
func (h *Handler) GetAdminStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.repo.GetAdminStats()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to calculate statistics")
		return
	}
	sendJSON(w, http.StatusOK, stats)
}

func (h *Handler) GetAdminUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.repo.GetAllUsers()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to fetch users")
		return
	}
	sendJSON(w, http.StatusOK, users)
}

func (h *Handler) AdminCreateSong(w http.ResponseWriter, r *http.Request) {
	// Support JSON or multipart form
	contentType := r.Header.Get("Content-Type")

	var song model.Song

	if strings.HasPrefix(contentType, "multipart/form-data") {
		_ = r.ParseMultipartForm(32 << 20) // 32MB max memory

		song.Title = r.FormValue("title")
		artistID, _ := strconv.ParseInt(r.FormValue("artist_id"), 10, 64)
		song.ArtistID = artistID
		albumIDStr := r.FormValue("album_id")
		if albumIDStr != "" {
			albumID, _ := strconv.ParseInt(albumIDStr, 10, 64)
			song.AlbumID = &albumID
		}
		duration, _ := strconv.Atoi(r.FormValue("duration"))
		song.Duration = duration
		song.Genre = r.FormValue("genre")
		song.CoverURL = r.FormValue("cover_url")
		song.AudioURL = r.FormValue("audio_url")

		// Handle uploaded audio file if present
		file, handler, err := r.FormFile("audio")
		if err == nil {
			defer file.Close()
			filename := strconv.FormatInt(time.Now().UnixNano(), 10) + "_" + filepath.Base(handler.Filename)
			dstPath := filepath.Join(h.cfg.UploadAudio, filename)
			dst, err := os.Create(dstPath)
			if err == nil {
				defer dst.Close()
				_, _ = io.Copy(dst, file)
				song.AudioURL = "/uploads/audio/" + filename
			}
		}

		// Handle uploaded cover image if present
		coverFile, coverHandler, err := r.FormFile("cover")
		if err == nil {
			defer coverFile.Close()
			cfilename := strconv.FormatInt(time.Now().UnixNano(), 10) + "_" + filepath.Base(coverHandler.Filename)
			cdstPath := filepath.Join(h.cfg.UploadCovers, cfilename)
			cdst, err := os.Create(cdstPath)
			if err == nil {
				defer cdst.Close()
				_, _ = io.Copy(cdst, coverFile)
				song.CoverURL = "/uploads/covers/" + cfilename
			}
		}
	} else {
		if err := json.NewDecoder(r.Body).Decode(&song); err != nil {
			sendError(w, http.StatusBadRequest, "Invalid payload")
			return
		}
	}

	if song.Title == "" || song.ArtistID == 0 || song.AudioURL == "" {
		sendError(w, http.StatusBadRequest, "Title, artist_id, and audio_url/file are required")
		return
	}

	if song.CoverURL == "" {
		song.CoverURL = "https://images.unsplash.com/photo-1614613535308-eb5fbd3d2c17?w=500"
	}
	if song.Duration == 0 {
		song.Duration = 180
	}

	if err := h.repo.CreateSong(&song); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to create song")
		return
	}
	sendJSON(w, http.StatusCreated, song)
}

func (h *Handler) AdminDeleteSong(w http.ResponseWriter, r *http.Request, id int64) {
	if err := h.repo.DeleteSong(id); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to delete song")
		return
	}
	sendJSON(w, http.StatusOK, map[string]string{"message": "Song deleted successfully"})
}

func (h *Handler) AdminCreateArtist(w http.ResponseWriter, r *http.Request) {
	var a model.Artist
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil || a.Name == "" {
		sendError(w, http.StatusBadRequest, "Artist name is required")
		return
	}
	if a.ImageURL == "" {
		a.ImageURL = "https://images.unsplash.com/photo-1511671782779-c97d3d27a1d4?w=500"
	}
	if err := h.repo.CreateArtist(&a); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to create artist")
		return
	}
	sendJSON(w, http.StatusCreated, a)
}

func (h *Handler) AdminCreateAlbum(w http.ResponseWriter, r *http.Request) {
	var al model.Album
	if err := json.NewDecoder(r.Body).Decode(&al); err != nil || al.Title == "" || al.ArtistID == 0 {
		sendError(w, http.StatusBadRequest, "Album title and artist_id are required")
		return
	}
	if al.CoverURL == "" {
		al.CoverURL = "https://images.unsplash.com/photo-1614613535308-eb5fbd3d2c17?w=500"
	}
	if err := h.repo.CreateAlbum(&al); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to create album")
		return
	}
	sendJSON(w, http.StatusCreated, al)
}

func (h *Handler) AdminImportArtist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		ArtistName string `json:"artist_name"`
		Genre      string `json:"genre"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.ArtistName) == "" {
		sendError(w, http.StatusBadRequest, "artist_name is required")
		return
	}

	if req.Genre == "" {
		req.Genre = "Pop / Rock"
	}

	dz := service.NewDeezerService()
	searchRes, err := dz.SearchArtist(req.ArtistName)
	if err != nil || len(searchRes.Data) == 0 {
		sendError(w, http.StatusNotFound, "Artist not found on Deezer catalog")
		return
	}

	dzArtist := searchRes.Data[0]
	artist := &model.Artist{
		Name:     dzArtist.Name,
		Bio:      "Official catalog imported via Deezer Music API.",
		ImageURL: dzArtist.PictureBig,
	}
	if artist.ImageURL == "" {
		artist.ImageURL = dzArtist.PictureMedium
	}
	if err := h.repo.GetOrCreateArtist(artist); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to save artist")
		return
	}

	albumsRes, err := dz.GetArtistAlbums(dzArtist.ID)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to fetch artist albums")
		return
	}

	importedAlbums := 0
	importedSongs := 0

	// Import up to 3 albums to keep import fast and responsive
	limit := 3
	if len(albumsRes.Data) < limit {
		limit = len(albumsRes.Data)
	}

	for i := 0; i < limit; i++ {
		dzAlbum := albumsRes.Data[i]
		album := &model.Album{
			Title:       dzAlbum.Title,
			ArtistID:    artist.ID,
			CoverURL:    dzAlbum.CoverBig,
			ReleaseDate: dzAlbum.ReleaseDate,
		}
		if album.CoverURL == "" {
			album.CoverURL = dzAlbum.CoverMedium
		}
		if err := h.repo.GetOrCreateAlbum(album); err != nil {
			continue
		}
		importedAlbums++

		tracksRes, err := dz.GetAlbumTracks(dzAlbum.ID)
		if err != nil {
			continue
		}
		for _, t := range tracksRes.Data {
			if t.Preview == "" {
				continue
			}
			albumID := album.ID
			song := &model.Song{
				Title:    t.Title,
				ArtistID: artist.ID,
				AlbumID:  &albumID,
				Duration: t.Duration,
				Genre:    req.Genre,
				AudioURL: t.Preview,
				CoverURL: album.CoverURL,
			}
			if err := h.repo.CreateSong(song); err == nil {
				importedSongs++
			}
		}
	}

	sendJSON(w, http.StatusCreated, map[string]interface{}{
		"message":         "Import completed successfully",
		"artist":          artist,
		"imported_albums": importedAlbums,
		"imported_songs":  importedSongs,
	})
}
