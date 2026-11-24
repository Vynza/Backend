package model

import "time"

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

type User struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	Role      Role      `json:"role"`
	AvatarURL string    `json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`
}

type Artist struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Bio       string    `json:"bio"`
	ImageURL  string    `json:"image_url"`
	CreatedAt time.Time `json:"created_at"`
}

type Album struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	ArtistID    int64     `json:"artist_id"`
	ArtistName  string    `json:"artist_name,omitempty"`
	CoverURL    string    `json:"cover_url"`
	ReleaseDate string    `json:"release_date"`
	CreatedAt   time.Time `json:"created_at"`
}

type Song struct {
	ID         int64     `json:"id"`
	Title      string    `json:"title"`
	ArtistID   int64     `json:"artist_id"`
	ArtistName string    `json:"artist_name,omitempty"`
	AlbumID    *int64    `json:"album_id,omitempty"`
	AlbumTitle string    `json:"album_title,omitempty"`
	Duration   int       `json:"duration"` // in seconds
	Genre      string    `json:"genre"`
	AudioURL   string    `json:"audio_url"`
	CoverURL   string    `json:"cover_url"`
	PlayCount  int64     `json:"play_count"`
	CreatedAt  time.Time `json:"created_at"`
}

type Playlist struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	CoverURL    string    `json:"cover_url"`
	SongCount   int       `json:"song_count,omitempty"`
	Songs       []Song    `json:"songs,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type AdminStats struct {
	TotalUsers     int64 `json:"total_users"`
	TotalSongs     int64 `json:"total_songs"`
	TotalArtists   int64 `json:"total_artists"`
	TotalAlbums    int64 `json:"total_albums"`
	TotalPlaylists int64 `json:"total_playlists"`
	TotalStreams   int64 `json:"total_streams"`
}
