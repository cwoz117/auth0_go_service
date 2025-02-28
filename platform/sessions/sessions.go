package sessions

import (
    "bytes"
    "context"
    "encoding/gob"
    "errors"
    "net/http"
    "time"

    "github.com/bradfitz/gomemcache/memcache"
    "github.com/google/uuid"
)

// MemcachedStore implements SessionStore using Memcached
type MemcachedStore struct {
    Client *memcache.Client
    TTL    time.Duration
}

// NewMemcachedStore initializes a Memcached-based session store
func NewMemcachedStore(address string, ttl time.Duration) *MemcachedStore {
    client := memcache.New(address)
    return &MemcachedStore{
        Client: client,
        TTL:    ttl,
    }
}

// Set stores a session value
func (m *MemcachedStore) Set(ctx context.Context, sessionID, key string, value interface{}) error {
    var buf bytes.Buffer
    enc := gob.NewEncoder(&buf)
    if err := enc.Encode(value); err != nil {
        return err
    }
    return m.Client.Set(&memcache.Item{
        Key:        sessionID + ":" + key,
        Value:      buf.Bytes(),
        Expiration: int32(m.TTL.Seconds()),
    })
}

// Get retrieves a session value
func (m *MemcachedStore) Get(ctx context.Context, sessionID, key string) (interface{}, error) {
    item, err := m.Client.Get(sessionID + ":" + key)
    if err != nil {
        return nil, err
    }
    var value interface{}
    dec := gob.NewDecoder(bytes.NewReader(item.Value))
    if err := dec.Decode(&value); err != nil {
        return nil, err
    }
    return value, nil
}

// Delete removes a session
func (m *MemcachedStore) Delete(ctx context.Context, sessionID string) error {
    return m.Client.Delete(sessionID)
}

// SessionManager manages session cookies
type SessionManager struct {
    Store     SessionStore
    CookieKey string
    TTL       time.Duration
}

// NewSession creates a new session
func (s *SessionManager) NewSession(w http.ResponseWriter) string {
    sessionID := uuid.NewString()
    http.SetCookie(w, &http.Cookie{
        Name:     s.CookieKey,
        Value:    sessionID,
        Path:     "/",
        HttpOnly: true,
        Secure:   true,
        Expires:  time.Now().Add(s.TTL),
    })
    return sessionID
}

// GetSession retrieves the session ID from a request
func (s *SessionManager) GetSession(r *http.Request) (string, error) {
    cookie, err := r.Cookie(s.CookieKey)
    if err != nil {
        return "", err
    }
    return cookie.Value, nil
}

// DestroySession deletes a session and removes the cookie
func (s *SessionManager) DestroySession(w http.ResponseWriter, r *http.Request) {
    sessionID, err := s.GetSession(r)
    if err == nil {
        s.Store.Delete(r.Context(), sessionID)
    }
    http.SetCookie(w, &http.Cookie{
        Name:     s.CookieKey,
        Value:    "",
        Path:     "/",
        HttpOnly: true,
        Secure:   true,
        Expires:  time.Now().Add(-time.Hour), // Expire immediately
    })
}

