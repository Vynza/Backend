-- ============================================================
-- VYNZA Music Platform - Supabase PostgreSQL Schema & Seed
-- ============================================================

-- 1. Drop existing tables if needed (Clean setup)
DROP TABLE IF EXISTS listening_history CASCADE;
DROP TABLE IF EXISTS favorites CASCADE;
DROP TABLE IF EXISTS playlist_songs CASCADE;
DROP TABLE IF EXISTS playlists CASCADE;
DROP TABLE IF EXISTS songs CASCADE;
DROP TABLE IF EXISTS albums CASCADE;
DROP TABLE IF EXISTS artists CASCADE;
DROP TABLE IF EXISTS users CASCADE;

-- 2. Users Table
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(100) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    avatar_url TEXT DEFAULT 'https://images.unsplash.com/photo-1535713875002-d1d0cf377fde?w=200',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 3. Artists Table
CREATE TABLE artists (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    bio TEXT DEFAULT '',
    image_url TEXT DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 4. Albums Table
CREATE TABLE albums (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    artist_id BIGINT NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
    cover_url TEXT DEFAULT '',
    release_date VARCHAR(50) DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 5. Songs Table
CREATE TABLE songs (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    artist_id BIGINT NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
    album_id BIGINT REFERENCES albums(id) ON DELETE SET NULL,
    duration INTEGER NOT NULL DEFAULT 0,
    genre VARCHAR(100) DEFAULT 'Music',
    audio_url TEXT NOT NULL,
    cover_url TEXT DEFAULT '',
    play_count BIGINT DEFAULT 0,
    lyrics TEXT DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 6. Playlists Table
CREATE TABLE playlists (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT DEFAULT '',
    cover_url TEXT DEFAULT 'https://images.unsplash.com/photo-1514525253161-7a46d19cd819?w=500',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 7. Playlist Songs (Many-to-Many)
CREATE TABLE playlist_songs (
    playlist_id BIGINT NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,
    song_id BIGINT NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    added_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (playlist_id, song_id)
);

-- 8. Favorites Table
CREATE TABLE favorites (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    song_id BIGINT NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, song_id)
);

-- 9. Listening History Table
CREATE TABLE listening_history (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    song_id BIGINT NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    played_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- INITIAL SEED DATA (Admin, Listener, Artists, Albums, Songs)
-- ============================================================

-- Password Hash: Bcrypt for 'Admin12345!' and 'User12345!'
INSERT INTO users (username, email, password, role, avatar_url) VALUES 
('admin', 'admin@vynza.com', '$2a$10$wE43r/w/q9y3yD9MsqXGv.9eD3o0q6Z1K6l1fVqPjK1oW2y8gVvKe', 'admin', 'https://images.unsplash.com/photo-1534528741775-53994a69daeb?w=200'),
('listener', 'listener@vynza.com', '$2a$10$wE43r/w/q9y3yD9MsqXGv.9eD3o0q6Z1K6l1fVqPjK1oW2y8gVvKe', 'user', 'https://images.unsplash.com/photo-1535713875002-d1d0cf377fde?w=200')
ON CONFLICT (email) DO NOTHING;

-- Seed Artists
INSERT INTO artists (id, name, bio, image_url) VALUES 
(1, 'CyberPulse', 'Electronic synthwave pioneer crafting futuristic soundscapes and cyber beats.', 'https://images.unsplash.com/photo-1511671782779-c97d3d27a1d4?w=500'),
(2, 'Aurora Nights', 'Indie dream-pop duo exploring emotional melodies and starry acoustic waves.', 'https://images.unsplash.com/photo-1493225457124-a3eb161ffa5f?w=500'),
(3, 'Urban Echo', 'Modern lo-fi and R&B producer from Jakarta crafting chill vibes.', 'https://images.unsplash.com/photo-1470225620780-dba8ba36b745?w=500')
ON CONFLICT (name) DO NOTHING;

-- Seed Albums
INSERT INTO albums (id, title, artist_id, cover_url, release_date) VALUES 
(1, 'Neon Horizon', 1, 'https://images.unsplash.com/photo-1614613535308-eb5fbd3d2c17?w=500', '2026-01-15'),
(2, 'Starlight Memories', 2, 'https://images.unsplash.com/photo-1518709268805-4e9042af9f23?w=500', '2026-02-10'),
(3, 'Midnight Coffee', 3, 'https://images.unsplash.com/photo-1501386761578-eac5c94b800a?w=500', '2026-03-01')
ON CONFLICT (id) DO NOTHING;

-- Seed Songs
INSERT INTO songs (id, title, artist_id, album_id, duration, genre, audio_url, cover_url, play_count) VALUES 
(1, 'Electric Sunset', 1, 1, 185, 'Synthwave', 'https://www.soundhelix.com/examples/mp3/SoundHelix-Song-1.mp3', 'https://images.unsplash.com/photo-1614613535308-eb5fbd3d2c17?w=500', 1420),
(2, 'Cybernetic Highway', 1, 1, 210, 'Electronic', 'https://www.soundhelix.com/examples/mp3/SoundHelix-Song-2.mp3', 'https://images.unsplash.com/photo-1618005182384-a83a8bd57fbe?w=500', 890),
(3, 'Whispering Stars', 2, 2, 195, 'Dream Pop', 'https://www.soundhelix.com/examples/mp3/SoundHelix-Song-3.mp3', 'https://images.unsplash.com/photo-1518709268805-4e9042af9f23?w=500', 2310),
(4, 'Raindrops on Asphalt', 3, 3, 160, 'Lo-Fi', 'https://www.soundhelix.com/examples/mp3/SoundHelix-Song-4.mp3', 'https://images.unsplash.com/photo-1501386761578-eac5c94b800a?w=500', 3150)
ON CONFLICT (id) DO NOTHING;

-- Reset Sequences to highest IDs
SELECT setval('users_id_seq', (SELECT MAX(id) FROM users));
SELECT setval('artists_id_seq', (SELECT MAX(id) FROM artists));
SELECT setval('albums_id_seq', (SELECT MAX(id) FROM albums));
SELECT setval('songs_id_seq', (SELECT MAX(id) FROM songs));
