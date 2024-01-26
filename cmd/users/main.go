package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"io"
	"net/http"
	"time"
)

type user struct {
	ID                uuid.UUID  `json:"id,omitempty" db:"id"`
	Name              string     `json:"name,omitempty" db:"name"`
	Email             string     `json:"email,omitempty" db:"email"`
	CreatedAt         *time.Time `json:"createdAt,omitempty" db:"created_at"`
	ApprovedForExamAt *time.Time `json:"approvedForExamAt,omitempty" db:"approved_for_exam_at"`
	ArchivedAt        *time.Time `json:"archivedAt,omitempty" db:"archived_at"`
}

var users = make([]user, 0)

func health(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "health\n")
}

func userList(w http.ResponseWriter, req *http.Request) {
	// language=SQL
	SQL := "select id, name, email, created_at, approved_for_exam_at, archived_at from user where archived_at is null"
	var users []user
	err := DB.Select(&users, SQL)
	if err != nil {
		fmt.Println("error reading users %+v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	RespondJSON(w, http.StatusOK, users)
}

func addUser(w http.ResponseWriter, req *http.Request) {
	var body user
	if err := ParseBody(req.Body, &body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	SQL := `insert into user (id, name, email, created_at) values ($1, $2, $3, $4)`
	_, err := DB.Queryx(SQL, uuid.New(), body.Name, body.Email, time.Now())
	if err != nil {
		fmt.Println("error writing users %+v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func approveForExam(w http.ResponseWriter, req *http.Request) {
	userID := chi.URLParam(req, "id")

	SQL := `update user set approved_for_exam_at = $1 where id = $2`
	_, err := DB.Queryx(SQL, time.Now(), userID)
	if err != nil {
		fmt.Println("error updating user approval status %+v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func deleteUser(w http.ResponseWriter, req *http.Request) {
	userID := chi.URLParam(req, "id")

	SQL := `update user set archived_at = now() where id = $1`
	_, err := DB.Queryx(SQL, userID)
	if err != nil {
		fmt.Println("error deleting user %+v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

var DB *sqlx.DB

func main() {
	// Set up PostgreSQL connection parameters
	psqlInfo := fmt.Sprintf("host=localhost port=5432 user=local password=local dbname=todo sslmode=disable")

	// Initialize the PostgreSQL database connection
	var err error
	DB, err = sqlx.Open("postgres", psqlInfo)
	if err != nil {
		fmt.Println("Unable to Connect to the Database: ", err)
		return
	}

	// Check if the database connection is successful
	err = DB.Ping()
	if err != nil {
		fmt.Println("Ping Panic: ", err)
		return
	}

	// Set up the Chi router
	router := chi.NewRouter()

	// Define routes
	router.Route("/", func(r chi.Router) {
		r.Get("/health", health)

		// userRouters(r)
		r.Get("/users", userList)
		r.Post("/add-User", addUser)
		r.Put("/approved-User", approveForExam)
		r.Delete("/delete-User", deleteUser)

		userRouters(r)
	})

	// Start the server on port 8080
	fmt.Println("Starting server at port 8080")
	http.ListenAndServe(":8080", router)
}

func userRouters(r chi.Router) chi.Router {
	return r.Route("/user", func(userRouter chi.Router) {
		userRouter.Get("/", userList)
		userRouter.Post("/", addUser)
		userRouter.Route("/{id}", func(userIDRouter chi.Router) {
			userIDRouter.Put("/approve", approveForExam)
			userIDRouter.Delete("/", deleteUser)
		})
	})
}

// RespondJSON sends the rateMetricInterface as a JSON
func RespondJSON(w http.ResponseWriter, statusCode int, body interface{}) {
	w.WriteHeader(statusCode)
	if body != nil {
		if err := EncodeJSONBody(w, body); err != nil {
			fmt.Println(fmt.Errorf("failed to respond JSON with error: %+v", err))
		}
	}
}

// EncodeJSONBody writes the JSON body to response writer
func EncodeJSONBody(resp http.ResponseWriter, data interface{}) error {
	return json.NewEncoder(resp).Encode(data)
}

// ParseBody parses the values from io reader to a given interface
func ParseBody(body io.Reader, out interface{}) error {
	err := json.NewDecoder(body).Decode(out)
	if err != nil {
		return err
	}

	return nil
}
