package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type DeezerArtistSearchResponse struct {
	Data []struct {
		ID            int64  `json:"id"`
		Name          string `json:"name"`
		PictureBig    string `json:"picture_big"`
		PictureMedium string `json:"picture_medium"`
	} `json:"data"`
}

type DeezerAlbumsResponse struct {
	Data []struct {
		ID          int64  `json:"id"`
		Title       string `json:"title"`
		CoverBig    string `json:"cover_big"`
		CoverMedium string `json:"cover_medium"`
		ReleaseDate string `json:"release_date"`
	} `json:"data"`
}

type DeezerTracksResponse struct {
	Data []struct {
		ID       int64  `json:"id"`
		Title    string `json:"title"`
		Duration int    `json:"duration"`
		Preview  string `json:"preview"`
	} `json:"data"`
}

type DeezerService struct {
	client *http.Client
}

func NewDeezerService() *DeezerService {
	return &DeezerService{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (s *DeezerService) SearchArtist(query string) (*DeezerArtistSearchResponse, error) {
	u := fmt.Sprintf("https://api.deezer.com/search/artist?q=%s&limit=5", url.QueryEscape(query))
	resp, err := s.client.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result DeezerArtistSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *DeezerService) GetArtistAlbums(artistID int64) (*DeezerAlbumsResponse, error) {
	u := fmt.Sprintf("https://api.deezer.com/artist/%d/albums?limit=10", artistID)
	resp, err := s.client.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result DeezerAlbumsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *DeezerService) GetAlbumTracks(albumID int64) (*DeezerTracksResponse, error) {
	u := fmt.Sprintf("https://api.deezer.com/album/%d/tracks?limit=50", albumID)
	resp, err := s.client.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result DeezerTracksResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}
