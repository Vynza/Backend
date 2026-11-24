package repository

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"vynza-backend/internal/model"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
}

func NewDB(dbPath string) (*DB, error) {
	dir := filepath.Dir(dbPath)
	if dir != "." && dir != "" {
		_ = os.MkdirAll(dir, 0755)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(1) // SQLite single-writer recommendation

	if err := initSchema(db); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return &DB{db}, nil
}

func initSchema(db *sql.DB) error {
	schema := `
	PRAGMA foreign_keys = ON;

	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		email TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL,
		role TEXT NOT NULL DEFAULT 'user',
		avatar_url TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS artists (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		bio TEXT DEFAULT '',
		image_url TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS albums (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		artist_id INTEGER NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
		cover_url TEXT DEFAULT '',
		release_date TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS songs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		artist_id INTEGER NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
		album_id INTEGER REFERENCES albums(id) ON DELETE SET NULL,
		duration INTEGER NOT NULL DEFAULT 0,
		genre TEXT DEFAULT '',
		audio_url TEXT NOT NULL,
		cover_url TEXT DEFAULT '',
		play_count INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS playlists (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		title TEXT NOT NULL,
		description TEXT DEFAULT '',
		cover_url TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS playlist_songs (
		playlist_id INTEGER NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,
		song_id INTEGER NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
		added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (playlist_id, song_id)
	);

	CREATE TABLE IF NOT EXISTS favorites (
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		song_id INTEGER NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (user_id, song_id)
	);

	CREATE TABLE IF NOT EXISTS listening_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		song_id INTEGER NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
		played_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return err
	}

	seedInitialData(db)
	return nil
}

func seedInitialData(db *sql.DB) {
	// Seed Admin if not exists
	var userCount int
	_ = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	if userCount == 0 {
		hashPassword := func(pwd string) string {
			bytes, _ := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
			return string(bytes)
		}

		adminHash := hashPassword("Admin12345!")
		userHash := hashPassword("User12345!")

		_, _ = db.Exec(`
			INSERT INTO users (username, email, password, role, avatar_url) VALUES 
			('admin', 'admin@vynza.com', ?, 'admin', 'https://images.unsplash.com/photo-1534528741775-53994a69daeb?w=200'),
			('listener', 'listener@vynza.com', ?, 'user', 'https://images.unsplash.com/photo-1535713875002-d1d0cf377fde?w=200')
		`, adminHash, userHash)

		// Seed Artists
		_, _ = db.Exec(`
			INSERT INTO artists (id, name, bio, image_url) VALUES 
			(1, 'CyberPulse', 'Electronic synthwave pioneer crafting futuristic soundscapes.', 'https://images.unsplash.com/photo-1511671782779-c97d3d27a1d4?w=500'),
			(2, 'Aurora Nights', 'Indie dream-pop duo exploring emotional melodies.', 'https://images.unsplash.com/photo-1493225457124-a3eb161ffa5f?w=500'),
			(3, 'Urban Echo', 'Modern lo-fi and R&B producer from Jakarta.', 'https://images.unsplash.com/photo-1470225620780-dba8ba36b745?w=500')
		`)

		// Seed Albums
		_, _ = db.Exec(`
			INSERT INTO albums (id, title, artist_id, cover_url, release_date) VALUES 
			(1, 'Neon Horizon', 1, 'https://images.unsplash.com/photo-1614613535308-eb5fbd3d2c17?w=500', '2026-01-15'),
			(2, 'Starlight Memories', 2, 'https://images.unsplash.com/photo-1518709268805-4e9042af9f23?w=500', '2026-02-10'),
			(3, 'Midnight Coffee', 3, 'https://images.unsplash.com/photo-1501386761578-eac5c94b800a?w=500', '2026-03-01')
		`)

		// Seed sample audio tracks (using free public Creative Commons / synthetic test audio)
		_, _ = db.Exec(`
			INSERT INTO songs (id, title, artist_id, album_id, duration, genre, audio_url, cover_url, play_count) VALUES 
			(1, 'Electric Sunset', 1, 1, 185, 'Synthwave', 'https://www.soundhelix.com/examples/mp3/SoundHelix-Song-1.mp3', 'https://images.unsplash.com/photo-1614613535308-eb5fbd3d2c17?w=500', 1420),
			(2, 'Cybernetic Highway', 1, 1, 210, 'Electronic', 'https://www.soundhelix.com/examples/mp3/SoundHelix-Song-2.mp3', 'https://images.unsplash.com/photo-1618005182384-a83a8bd57fbe?w=500', 890),
			(3, 'Whispering Stars', 2, 2, 195, 'Dream Pop', 'https://www.soundhelix.com/examples/mp3/SoundHelix-Song-3.mp3', 'https://images.unsplash.com/photo-1518709268805-4e9042af9f23?w=500', 2310),
			(4, 'Raindrops on Asphalt', 3, 3, 160, 'Lo-Fi', 'https://www.soundhelix.com/examples/mp3/SoundHelix-Song-4.mp3', 'https://images.unsplash.com/photo-1501386761578-eac5c94b800a?w=500', 3150)
		`)

		// Seed Playlist
		_, _ = db.Exec(`
			INSERT INTO playlists (id, user_id, title, description, cover_url) VALUES 
			(1, 2, 'Late Night Vibes', 'Chilled beats to relax, code and feel every beat.', 'https://images.unsplash.com/photo-1514525253161-7a46d19cd819?w=500')
		`)

		_, _ = db.Exec(`
			INSERT INTO playlist_songs (playlist_id, song_id) VALUES 
			(1, 1), (1, 4)
		`)

		// Seed Favorites
		_, _ = db.Exec(`
			INSERT INTO favorites (user_id, song_id) VALUES (2, 1), (2, 3)
		`)

		log.Println("[INFO] Initial seed data created successfully (Users, Artists, Albums, Songs, Playlist)")
	}
}

// User Methods
func (db *DB) CreateUser(u *model.User) error {
	res, err := db.Exec(
		"INSERT INTO users (username, email, password, role, avatar_url) VALUES (?, ?, ?, ?, ?)",
		u.Username, u.Email, u.Password, u.Role, u.AvatarURL,
	)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	u.ID = id
	return nil
}

func (db *DB) GetUserByEmail(email string) (*model.User, error) {
	row := db.QueryRow("SELECT id, username, email, password, role, avatar_url, created_at FROM users WHERE email = ?", email)
	u := &model.User{}
	var createdAt string
	err := row.Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.Role, &u.AvatarURL, &createdAt)
	if err != nil {
		return nil, err
	}
	u.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	return u, nil
}

func (db *DB) GetUserByID(id int64) (*model.User, error) {
	row := db.QueryRow("SELECT id, username, email, password, role, avatar_url, created_at FROM users WHERE id = ?", id)
	u := &model.User{}
	var createdAt string
	err := row.Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.Role, &u.AvatarURL, &createdAt)
	if err != nil {
		return nil, err
	}
	u.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	return u, nil
}

func (db *DB) GetAllUsers() ([]model.User, error) {
	rows, err := db.Query("SELECT id, username, email, role, avatar_url, created_at FROM users ORDER BY id DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		var createdAt string
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.AvatarURL, &createdAt); err == nil {
			u.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
			users = append(users, u)
		}
	}
	return users, nil
}

// Songs Catalog
func (db *DB) GetSongs(search, genre string, artistID int64) ([]model.Song, error) {
	query := `
		SELECT s.id, s.title, s.artist_id, a.name, s.album_id, COALESCE(al.title, ''), 
		       s.duration, s.genre, s.audio_url, s.cover_url, s.play_count, s.created_at
		FROM songs s
		JOIN artists a ON s.artist_id = a.id
		LEFT JOIN albums al ON s.album_id = al.id
		WHERE 1=1
	`
	var args []interface{}
	if search != "" {
		query += " AND (s.title LIKE ? OR a.name LIKE ?)"
		pattern := "%" + search + "%"
		args = append(args, pattern, pattern)
	}
	if genre != "" {
		query += " AND s.genre = ?"
		args = append(args, genre)
	}
	if artistID > 0 {
		query += " AND s.artist_id = ?"
		args = append(args, artistID)
	}
	query += " ORDER BY s.id DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var songs []model.Song
	for rows.Next() {
		var s model.Song
		var albumTitle string
		var createdAt string
		err := rows.Scan(
			&s.ID, &s.Title, &s.ArtistID, &s.ArtistName, &s.AlbumID, &albumTitle,
			&s.Duration, &s.Genre, &s.AudioURL, &s.CoverURL, &s.PlayCount, &createdAt,
		)
		if err == nil {
			s.AlbumTitle = albumTitle
			s.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
			songs = append(songs, s)
		}
	}
	return songs, nil
}

func (db *DB) GetSongByID(id int64) (*model.Song, error) {
	query := `
		SELECT s.id, s.title, s.artist_id, a.name, s.album_id, COALESCE(al.title, ''), 
		       s.duration, s.genre, s.audio_url, s.cover_url, s.play_count, s.created_at
		FROM songs s
		JOIN artists a ON s.artist_id = a.id
		LEFT JOIN albums al ON s.album_id = al.id
		WHERE s.id = ?
	`
	row := db.QueryRow(query, id)
	var s model.Song
	var albumTitle string
	var createdAt string
	err := row.Scan(
		&s.ID, &s.Title, &s.ArtistID, &s.ArtistName, &s.AlbumID, &albumTitle,
		&s.Duration, &s.Genre, &s.AudioURL, &s.CoverURL, &s.PlayCount, &createdAt,
	)
	if err != nil {
		return nil, err
	}
	s.AlbumTitle = albumTitle
	s.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	return &s, nil
}

func (db *DB) IncrementPlayCount(id int64) {
	_, _ = db.Exec("UPDATE songs SET play_count = play_count + 1 WHERE id = ?", id)
}

func (db *DB) CreateSong(s *model.Song) error {
	res, err := db.Exec(
		"INSERT INTO songs (title, artist_id, album_id, duration, genre, audio_url, cover_url) VALUES (?, ?, ?, ?, ?, ?, ?)",
		s.Title, s.ArtistID, s.AlbumID, s.Duration, s.Genre, s.AudioURL, s.CoverURL,
	)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	s.ID = id
	return nil
}

func (db *DB) DeleteSong(id int64) error {
	_, err := db.Exec("DELETE FROM songs WHERE id = ?", id)
	return err
}

// Artists
func (db *DB) GetArtists() ([]model.Artist, error) {
	rows, err := db.Query("SELECT id, name, bio, image_url, created_at FROM artists ORDER BY name ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artists []model.Artist
	for rows.Next() {
		var a model.Artist
		var createdAt string
		if err := rows.Scan(&a.ID, &a.Name, &a.Bio, &a.ImageURL, &createdAt); err == nil {
			a.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
			artists = append(artists, a)
		}
	}
	return artists, nil
}

func (db *DB) CreateArtist(a *model.Artist) error {
	res, err := db.Exec("INSERT INTO artists (name, bio, image_url) VALUES (?, ?, ?)", a.Name, a.Bio, a.ImageURL)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	a.ID = id
	return nil
}

// Albums
func (db *DB) GetAlbums() ([]model.Album, error) {
	query := `
		SELECT al.id, al.title, al.artist_id, a.name, al.cover_url, al.release_date, al.created_at
		FROM albums al
		JOIN artists a ON al.artist_id = a.id
		ORDER BY al.id DESC
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var albums []model.Album
	for rows.Next() {
		var al model.Album
		var createdAt string
		if err := rows.Scan(&al.ID, &al.Title, &al.ArtistID, &al.ArtistName, &al.CoverURL, &al.ReleaseDate, &createdAt); err == nil {
			al.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
			albums = append(albums, al)
		}
	}
	return albums, nil
}

func (db *DB) CreateAlbum(al *model.Album) error {
	res, err := db.Exec("INSERT INTO albums (title, artist_id, cover_url, release_date) VALUES (?, ?, ?, ?)", al.Title, al.ArtistID, al.CoverURL, al.ReleaseDate)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	al.ID = id
	return nil
}

// Playlists
func (db *DB) GetUserPlaylists(userID int64) ([]model.Playlist, error) {
	query := `
		SELECT p.id, p.user_id, p.title, p.description, p.cover_url, p.created_at, COUNT(ps.song_id)
		FROM playlists p
		LEFT JOIN playlist_songs ps ON p.id = ps.playlist_id
		WHERE p.user_id = ?
		GROUP BY p.id
		ORDER BY p.id DESC
	`
	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.Playlist
	for rows.Next() {
		var p model.Playlist
		var createdAt string
		if err := rows.Scan(&p.ID, &p.UserID, &p.Title, &p.Description, &p.CoverURL, &createdAt, &p.SongCount); err == nil {
			p.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
			list = append(list, p)
		}
	}
	return list, nil
}

func (db *DB) CreatePlaylist(p *model.Playlist) error {
	res, err := db.Exec("INSERT INTO playlists (user_id, title, description, cover_url) VALUES (?, ?, ?, ?)", p.UserID, p.Title, p.Description, p.CoverURL)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	p.ID = id
	return nil
}

func (db *DB) AddSongToPlaylist(playlistID, songID int64) error {
	_, err := db.Exec("INSERT OR IGNORE INTO playlist_songs (playlist_id, song_id) VALUES (?, ?)", playlistID, songID)
	return err
}

func (db *DB) RemoveSongFromPlaylist(playlistID, songID int64) error {
	_, err := db.Exec("DELETE FROM playlist_songs WHERE playlist_id = ? AND song_id = ?", playlistID, songID)
	return err
}

func (db *DB) GetPlaylistDetails(playlistID int64) (*model.Playlist, error) {
	row := db.QueryRow("SELECT id, user_id, title, description, cover_url, created_at FROM playlists WHERE id = ?", playlistID)
	var p model.Playlist
	var createdAt string
	if err := row.Scan(&p.ID, &p.UserID, &p.Title, &p.Description, &p.CoverURL, &createdAt); err != nil {
		return nil, err
	}
	p.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)

	songsQuery := `
		SELECT s.id, s.title, s.artist_id, a.name, s.album_id, COALESCE(al.title, ''), 
		       s.duration, s.genre, s.audio_url, s.cover_url, s.play_count, s.created_at
		FROM playlist_songs ps
		JOIN songs s ON ps.song_id = s.id
		JOIN artists a ON s.artist_id = a.id
		LEFT JOIN albums al ON s.album_id = al.id
		WHERE ps.playlist_id = ?
		ORDER BY ps.added_at ASC
	`
	rows, err := db.Query(songsQuery, playlistID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var s model.Song
			var albumTitle string
			var scDate string
			if err := rows.Scan(&s.ID, &s.Title, &s.ArtistID, &s.ArtistName, &s.AlbumID, &albumTitle, &s.Duration, &s.Genre, &s.AudioURL, &s.CoverURL, &s.PlayCount, &scDate); err == nil {
				s.AlbumTitle = albumTitle
				s.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", scDate)
				p.Songs = append(p.Songs, s)
			}
		}
	}
	p.SongCount = len(p.Songs)
	return &p, nil
}

// Favorites
func (db *DB) GetUserFavorites(userID int64) ([]model.Song, error) {
	query := `
		SELECT s.id, s.title, s.artist_id, a.name, s.album_id, COALESCE(al.title, ''), 
		       s.duration, s.genre, s.audio_url, s.cover_url, s.play_count, s.created_at
		FROM favorites f
		JOIN songs s ON f.song_id = s.id
		JOIN artists a ON s.artist_id = a.id
		LEFT JOIN albums al ON s.album_id = al.id
		WHERE f.user_id = ?
		ORDER BY f.created_at DESC
	`
	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var songs []model.Song
	for rows.Next() {
		var s model.Song
		var albumTitle string
		var scDate string
		if err := rows.Scan(&s.ID, &s.Title, &s.ArtistID, &s.ArtistName, &s.AlbumID, &albumTitle, &s.Duration, &s.Genre, &s.AudioURL, &s.CoverURL, &s.PlayCount, &scDate); err == nil {
			s.AlbumTitle = albumTitle
			s.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", scDate)
			songs = append(songs, s)
		}
	}
	return songs, nil
}

func (db *DB) AddFavorite(userID, songID int64) error {
	_, err := db.Exec("INSERT OR IGNORE INTO favorites (user_id, song_id) VALUES (?, ?)", userID, songID)
	return err
}

func (db *DB) RemoveFavorite(userID, songID int64) error {
	_, err := db.Exec("DELETE FROM favorites WHERE user_id = ? AND song_id = ?", userID, songID)
	return err
}

// Admin Stats
func (db *DB) GetAdminStats() (*model.AdminStats, error) {
	stats := &model.AdminStats{}
	_ = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&stats.TotalUsers)
	_ = db.QueryRow("SELECT COUNT(*) FROM songs").Scan(&stats.TotalSongs)
	_ = db.QueryRow("SELECT COUNT(*) FROM artists").Scan(&stats.TotalArtists)
	_ = db.QueryRow("SELECT COUNT(*) FROM albums").Scan(&stats.TotalAlbums)
	_ = db.QueryRow("SELECT COUNT(*) FROM playlists").Scan(&stats.TotalPlaylists)
	_ = db.QueryRow("SELECT COALESCE(SUM(play_count), 0) FROM songs").Scan(&stats.TotalStreams)
	return stats, nil
}
