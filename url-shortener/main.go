package main

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type SortMode int

const (
	SortDefault SortMode = iota
	SortClicksDescending
	SortClicksAscending
)

type ExpiredAtTime struct {
	time.Time
}

func (eat *ExpiredAtTime) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	if s == "" {
		return nil
	}

	t, err := time.Parse(time.DateTime, s)
	if err != nil {
		return err
	}
	eat.Time = t
	return nil
}

func (eat ExpiredAtTime) MarshalJSON() ([]byte, error) {
	fmt.Println(eat, eat.IsZero())
	if eat.IsZero() {
		return []byte("null"), nil
	} else {
		return []byte(fmt.Sprintf("\"%s\"", eat.Format(time.DateTime))), nil
	}
}

type ShortURL struct {
	ClickCounter uint64        `json:"clicks"`
	ShortURL     string        `json:"short_url"`
	RedirectURL  string        `json:"url"`
	ExpiredAt    ExpiredAtTime `json:"expired_at,omitempty"`
}

type URLShortener struct {
	shortURLs map[string]ShortURL
}

func NewURLShortener() *URLShortener {
	return &URLShortener{
		shortURLs: make(map[string]ShortURL),
	}
}

func (us *URLShortener) ShortenURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type is not application/json", http.StatusUnsupportedMediaType)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1048576)

	var s ShortURL
	err := json.NewDecoder(r.Body).Decode(&s)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var maxBytesError *http.MaxBytesError

		switch {
		case errors.As(err, &syntaxError):
			msg := fmt.Sprintf("Request body contains badly-formed JSON (at %d)", syntaxError.Offset)
			http.Error(w, msg, http.StatusBadRequest)

		case errors.Is(err, io.ErrUnexpectedEOF):
			http.Error(w, "Request body contains badly-formed JSON", http.StatusBadRequest)

		case errors.As(err, &unmarshalTypeError):
			msg := fmt.Sprintf("Request body contains an invalid value for the %q field (at %d)", unmarshalTypeError.Field, unmarshalTypeError.Offset)
			http.Error(w, msg, http.StatusBadRequest)

		case errors.Is(err, io.EOF):
			http.Error(w, "Request body must not be empty", http.StatusBadRequest)

		case errors.As(err, &maxBytesError):
			msg := fmt.Sprintf("Request body must not be larger than %.2f MB", float64(maxBytesError.Limit)/float64(1048576))
			http.Error(w, msg, http.StatusRequestEntityTooLarge)

		default:
			log.Print(err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	now := time.Now()
	s.ClickCounter = 0
	for {
		shortURL := us.GenerateShortURL(s.RedirectURL, strconv.Itoa(int(now.Unix())))
		s.ShortURL = shortURL
		if v, ok := us.shortURLs[shortURL]; ok {
			// Reuse the key on collision and when the shortened URL is expired
			if !v.ExpiredAt.IsZero() && now.After(v.ExpiredAt.Time) {
				us.shortURLs[shortURL] = s
				break
			}
		} else {
			us.shortURLs[shortURL] = s
			break
		}
	}

	err = json.NewEncoder(w).Encode(s)
	if err != nil {
		log.Print(err)
	}
}

// GenerateShortURL accept url to be shortened with suffix to make returned short url unique
func (us *URLShortener) GenerateShortURL(longURL string, suffix string) string {
	hash := sha1.New()
	hash.Write([]byte(longURL))
	hash.Write([]byte(suffix))
	url := base64.URLEncoding.EncodeToString(hash.Sum(nil))
	return url[:6]
}

func (us *URLShortener) RedirectShortURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	now := time.Now()
	if r.URL.Path == "/" {
		data := make([]ShortURL, 0)
		for _, v := range us.shortURLs {
			if v.ExpiredAt.IsZero() || v.ExpiredAt.After(now) {
				data = append(data, v)
			}
		}

		sortMode := SortDefault
		sort := r.URL.Query().Get("sort")
		switch sort {
		case "clicks":
			sortMode = SortClicksAscending
		case "-clicks":
			sortMode = SortClicksDescending
		}
		SortShortURL(data, sortMode)

		err := json.NewEncoder(w).Encode(data)
		if err != nil {
			log.Print(err)
		}
	} else {
		shortURL := strings.TrimPrefix(r.URL.Path, "/")
		if v, ok := us.shortURLs[shortURL]; ok {
			if v.ExpiredAt.IsZero() || v.ExpiredAt.After(now) {
				atomic.AddUint64(&v.ClickCounter, 1)
				us.shortURLs[shortURL] = v
				http.Redirect(w, r, v.RedirectURL, http.StatusSeeOther)
				return
			}
		}

		http.NotFound(w, r)
	}
}

func main() {
	urlShortener := NewURLShortener()

	http.HandleFunc("/shorten", urlShortener.ShortenURL)
	http.HandleFunc("/", urlShortener.RedirectShortURL)

	fmt.Println("Starting server at :8080")
	http.ListenAndServe(":8080", nil)
}

func SortShortURL(data []ShortURL, mode SortMode) []ShortURL {
	var res []ShortURL
	switch mode {
	case SortClicksAscending:
		sort.Slice(data, func(i, j int) bool {
			return data[i].ClickCounter < data[j].ClickCounter
		})
	case SortClicksDescending:
		sort.Slice(data, func(i, j int) bool {
			return data[i].ClickCounter > data[j].ClickCounter
		})
	default:
		res = data
	}

	return res
}
