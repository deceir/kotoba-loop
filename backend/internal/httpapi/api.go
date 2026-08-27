package httpapi

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"kotoba-loop/backend/internal/auth"
)

type API struct {
	DB             *sql.DB
	Secret, Origin string
}
type authRequest struct{ Email, Password string }
type ctxHandler func(http.ResponseWriter, *http.Request, int64)

func (a *API) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { write(w, 200, map[string]string{"status": "ok"}) })
	m.HandleFunc("POST /api/auth/register", a.register)
	m.HandleFunc("POST /api/auth/login", a.login)
	m.Handle("GET /api/decks", a.require(a.decks))
	m.Handle("POST /api/decks/{id}/add", a.require(a.addDeck))
	m.Handle("GET /api/reviews/next", a.require(a.nextReview))
	m.Handle("POST /api/reviews/{id}", a.require(a.review))
	return a.cors(m)
}
func (a *API) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", a.Origin)
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (a *API) require(next ctxHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		id, err := auth.Parse(h, a.Secret)
		if err != nil {
			problem(w, 401, "Please sign in again")
			return
		}
		next(w, r, id)
	})
}
func (a *API) register(w http.ResponseWriter, r *http.Request) {
	var v authRequest
	if decode(w, r, &v) != nil {
		return
	}
	v.Email = strings.ToLower(strings.TrimSpace(v.Email))
	if !strings.Contains(v.Email, "@") {
		problem(w, 400, "Enter a valid email")
		return
	}
	hash, err := auth.Hash(v.Password)
	if err != nil {
		problem(w, 400, err.Error())
		return
	}
	res, err := a.DB.Exec("INSERT INTO users(email,password_hash) VALUES (?,?)", v.Email, hash)
	if err != nil {
		problem(w, 409, "An account with that email already exists")
		return
	}
	id, _ := res.LastInsertId()
	token, _ := auth.Sign(id, a.Secret)
	write(w, 201, map[string]any{"token": token, "email": v.Email})
}
func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var v authRequest
	if decode(w, r, &v) != nil {
		return
	}
	var id int64
	var hash, email string
	err := a.DB.QueryRow("SELECT id,password_hash,email FROM users WHERE email=?", strings.ToLower(strings.TrimSpace(v.Email))).Scan(&id, &hash, &email)
	if err != nil || !auth.Verify(hash, v.Password) {
		problem(w, 401, "Email or password is incorrect")
		return
	}
	token, _ := auth.Sign(id, a.Secret)
	write(w, 200, map[string]any{"token": token, "email": email})
}
func (a *API) decks(w http.ResponseWriter, r *http.Request, userID int64) {
	rows, err := a.DB.Query(`SELECT d.id,d.name,d.description,COUNT(wd.word_id),SUM(CASE WHEN uw.user_id IS NOT NULL THEN 1 ELSE 0 END) FROM decks d LEFT JOIN word_decks wd ON wd.deck_id=d.id LEFT JOIN user_words uw ON uw.word_id=wd.word_id AND uw.user_id=? GROUP BY d.id ORDER BY d.id`, userID)
	if err != nil {
		server(w, err)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, total, added int64
		var n, d string
		if rows.Scan(&id, &n, &d, &total, &added) == nil {
			out = append(out, map[string]any{"id": id, "name": n, "description": d, "wordCount": total, "addedCount": added})
		}
	}
	write(w, 200, out)
}
func (a *API) addDeck(w http.ResponseWriter, r *http.Request, userID int64) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		problem(w, 400, "Invalid deck")
		return
	}
	res, err := a.DB.Exec(`INSERT IGNORE INTO user_words(user_id,word_id) SELECT ?,word_id FROM word_decks WHERE deck_id=?`, userID, id)
	if err != nil {
		server(w, err)
		return
	}
	n, _ := res.RowsAffected()
	write(w, 200, map[string]any{"added": n})
}
func (a *API) nextReview(w http.ResponseWriter, r *http.Request, userID int64) {
	var v struct {
		ID       int64  `json:"id"`
		English  string `json:"english"`
		Japanese string `json:"japanese"`
		Reading  string `json:"reading"`
		Deck     string `json:"deck"`
	}
	err := a.DB.QueryRow(`SELECT w.id,w.english,w.japanese,w.reading,MIN(d.name) FROM user_words uw JOIN words w ON w.id=uw.word_id JOIN word_decks wd ON wd.word_id=w.id JOIN decks d ON d.id=wd.deck_id WHERE uw.user_id=? AND uw.due_at<=NOW() GROUP BY w.id,w.english,w.japanese,w.reading,uw.due_at ORDER BY uw.due_at,w.id LIMIT 1`, userID).Scan(&v.ID, &v.English, &v.Japanese, &v.Reading, &v.Deck)
	if err == sql.ErrNoRows {
		w.WriteHeader(204)
		return
	}
	if err != nil {
		server(w, err)
		return
	}
	write(w, 200, v)
}
func (a *API) review(w http.ResponseWriter, r *http.Request, userID int64) {
	wordID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		problem(w, 400, "Invalid word")
		return
	}
	var v struct {
		Correct bool `json:"correct"`
	}
	if decode(w, r, &v) != nil {
		return
	}
	var reps, interval, lapses int
	var ease float64
	err = a.DB.QueryRow("SELECT repetitions,interval_days,ease_factor,lapses FROM user_words WHERE user_id=? AND word_id=?", userID, wordID).Scan(&reps, &interval, &ease, &lapses)
	if err != nil {
		problem(w, 404, "Word not found")
		return
	}
	if v.Correct {
		reps++
		if reps == 1 {
			interval = 1
		} else if reps == 2 {
			interval = 3
		} else {
			interval = int(float64(interval) * ease)
		}
		ease += 0.05
	} else {
		reps = 0
		interval = 0
		lapses++
		ease -= 0.2
		if ease < 1.3 {
			ease = 1.3
		}
	}
	due := time.Now().Add(10 * time.Minute)
	if interval > 0 {
		due = time.Now().Add(time.Duration(interval) * 24 * time.Hour)
	}
	_, err = a.DB.Exec("UPDATE user_words SET repetitions=?,interval_days=?,ease_factor=?,lapses=?,due_at=? WHERE user_id=? AND word_id=?", reps, interval, ease, lapses, due, userID, wordID)
	if err != nil {
		server(w, err)
		return
	}
	write(w, 200, map[string]any{"nextDue": due, "intervalDays": interval})
}
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	err := json.NewDecoder(r.Body).Decode(v)
	if err != nil {
		problem(w, 400, "Invalid request")
	}
	return err
}
func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func problem(w http.ResponseWriter, status int, msg string) {
	write(w, status, map[string]string{"error": msg})
}
func server(w http.ResponseWriter, err error) {
	log.Print(err)
	problem(w, 500, "Something went wrong")
}
