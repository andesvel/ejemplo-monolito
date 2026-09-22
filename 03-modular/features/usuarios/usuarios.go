// Package usuarios: feature completa de usuarios.
//
// En este archivo conviven, para un único dominio de negocio:
//   - el model      (struct Usuario)
//   - el repository (funciones SQL)
//   - el service    (reglas de negocio)
//   - el handler    (HTTP)
//
// Esa es la idea de la organización modular (vertical): lo que cambia junto,
// vive junto. Si el equipo decidiera extraer "usuarios" como microservicio,
// TODO lo necesario ya está dentro de este paquete: se copia la carpeta,
// se le agrega un main.go con su propio servidor/puerto y listo — ningún
// otro archivo del monolito tendría que tocarse, porque las demás features
// no conocen los detalles internos de esta (solo la función exportada Existe).
package usuarios

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

// ---------------------------------------------------------------------------
// Model
// ---------------------------------------------------------------------------

type Usuario struct {
	ID     int64  `json:"id"`
	Nombre string `json:"nombre"`
	Email  string `json:"email"`
}

// ---------------------------------------------------------------------------
// Errores de negocio
// ---------------------------------------------------------------------------

var (
	ErrEmailDuplicado      = errors.New("el email ya esta registrado")
	ErrUsuarioNoEncontrado = errors.New("usuario no encontrado")
)

// ErrorValidacion marca un error por datos del cliente (respuesta 400).
type ErrorValidacion struct{ Mensaje string }

func (e ErrorValidacion) Error() string { return e.Mensaje }

// ---------------------------------------------------------------------------
// Repository (SQL)
// ---------------------------------------------------------------------------

func crearUsuarioDB(db *sql.DB, usuario *Usuario) error {
	resultado, err := db.Exec(
		"INSERT INTO usuarios (nombre, email) VALUES (?, ?)",
		usuario.Nombre, usuario.Email,
	)
	if err != nil {
		return err
	}
	id, err := resultado.LastInsertId()
	if err != nil {
		return err
	}
	usuario.ID = id
	return nil
}

func listarUsuariosDB(db *sql.DB) ([]Usuario, error) {
	filas, err := db.Query("SELECT id, nombre, email FROM usuarios ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer filas.Close()

	usuarios := []Usuario{}
	for filas.Next() {
		var u Usuario
		if err := filas.Scan(&u.ID, &u.Nombre, &u.Email); err != nil {
			return nil, err
		}
		usuarios = append(usuarios, u)
	}
	return usuarios, filas.Err()
}

// Existe es la única función que la feature tareas necesita de esta feature:
// la interfaz entre features es mínima a propósito.
func Existe(db *sql.DB, id int64) (bool, error) {
	var n int
	if err := db.QueryRow("SELECT COUNT(1) FROM usuarios WHERE id = ?", id).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}

// ---------------------------------------------------------------------------
// Service (reglas de negocio)
// ---------------------------------------------------------------------------

func validarUsuario(usuario *Usuario) error {
	if strings.TrimSpace(usuario.Nombre) == "" {
		return ErrorValidacion{"el nombre es obligatorio"}
	}
	if !strings.Contains(usuario.Email, "@") {
		return ErrorValidacion{"email invalido"}
	}
	return nil
}

func crearUsuario(db *sql.DB, nombre, email string) (Usuario, error) {
	usuario := Usuario{Nombre: nombre, Email: email}
	if err := validarUsuario(&usuario); err != nil {
		return usuario, err
	}
	if err := crearUsuarioDB(db, &usuario); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return usuario, ErrEmailDuplicado
		}
		return usuario, err
	}
	return usuario, nil
}

// ---------------------------------------------------------------------------
// Handler (HTTP)
// ---------------------------------------------------------------------------

func responderJSON(w http.ResponseWriter, codigo int, datos any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(codigo)
	json.NewEncoder(w).Encode(datos)
}

func responderError(w http.ResponseWriter, codigo int, mensaje string) {
	responderJSON(w, codigo, map[string]string{"error": mensaje})
}

func responderSegunError(w http.ResponseWriter, err error) {
	var validacion ErrorValidacion
	switch {
	case errors.As(err, &validacion):
		responderError(w, http.StatusBadRequest, validacion.Mensaje)
	case errors.Is(err, ErrEmailDuplicado):
		responderError(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrUsuarioNoEncontrado):
		responderError(w, http.StatusNotFound, err.Error())
	default:
		responderError(w, http.StatusInternalServerError, err.Error())
	}
}

// RegistrarRutas agrega los endpoints de esta feature al mux de main.go.
func RegistrarRutas(mux *http.ServeMux, db *sql.DB) {
	mux.HandleFunc("POST /usuarios", func(w http.ResponseWriter, r *http.Request) {
		var entrada struct {
			Nombre string `json:"nombre"`
			Email  string `json:"email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&entrada); err != nil {
			responderError(w, http.StatusBadRequest, "JSON invalido")
			return
		}
		usuario, err := crearUsuario(db, entrada.Nombre, entrada.Email)
		if err != nil {
			responderSegunError(w, err)
			return
		}
		responderJSON(w, http.StatusCreated, usuario)
	})

	mux.HandleFunc("GET /usuarios", func(w http.ResponseWriter, r *http.Request) {
		usuarios, err := listarUsuariosDB(db)
		if err != nil {
			responderSegunError(w, err)
			return
		}
		responderJSON(w, http.StatusOK, usuarios)
	})
}
