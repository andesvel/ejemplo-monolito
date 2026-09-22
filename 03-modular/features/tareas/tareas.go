// Package tareas: feature completa de tareas (model + repository + service
// + handler en un solo lugar).
//
// DEPENDENCIAS ENTRE FEATURES: este paquete importa a features/usuarios
// (necesita verificar que el usuario_id exista) pero usuarios NO importa
// a tareas. La dirección de la dependencia es única y acíclica.
//
// Por qué esta organización facilita extraer un microservicio si algún día
// hiciera falta: todo lo que "tareas" necesita para funcionar vive DENTRO
// de este paquete (su SQL, sus reglas, su HTTP) y sus dependencias externas
// están acotadas a un punto visible (la llamada a usuarios.Existe). Para
// separarlo: se reemplaza esa única función por una llamada HTTP al servicio
// de usuarios y se mueve esta carpeta a un repositorio propio con un main.go.
// En la versión por capas, en cambio, un mismo dominio está repartido en 4
// carpetas distintas y extraerlo significa desenterrar y reagrupar código
// que convive con los demás dominios en cada capa.
package tareas

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"monolito/features/usuarios"
)

// ---------------------------------------------------------------------------
// Model
// ---------------------------------------------------------------------------

type Tarea struct {
	ID          int64  `json:"id"`
	Titulo      string `json:"titulo"`
	Descripcion string `json:"descripcion"`
	Estado      string `json:"estado"`
	UsuarioID   int64  `json:"usuario_id"`
	CreatedAt   string `json:"created_at"`
}

// Estados posibles de una tarea (una sola definición dentro de la feature).
const (
	EstadoPendiente  = "pendiente"
	EstadoEnProgreso = "en_progreso"
	EstadoCompletada = "completada"
)

// ---------------------------------------------------------------------------
// Errores de negocio
// ---------------------------------------------------------------------------

var (
	ErrTareaNoEncontrada = errors.New("tarea no encontrada")
)

// ErrorValidacion marca un error por datos del cliente (respuesta 400).
// (Se repite a propósito en cada feature: los paquetes no comparten código
// entre sí para mantener la frontera limpia y la extracción trivial.)
type ErrorValidacion struct{ Mensaje string }

func (e ErrorValidacion) Error() string { return e.Mensaje }

// ---------------------------------------------------------------------------
// Repository (SQL)
// ---------------------------------------------------------------------------

func crearTareaDB(db *sql.DB, tarea *Tarea) error {
	resultado, err := db.Exec(
		"INSERT INTO tareas (titulo, descripcion, estado, usuario_id) VALUES (?, ?, ?, ?)",
		tarea.Titulo, tarea.Descripcion, tarea.Estado, tarea.UsuarioID,
	)
	if err != nil {
		return err
	}
	id, err := resultado.LastInsertId()
	if err != nil {
		return err
	}
	tarea.ID = id
	return nil
}

func obtenerTareaPorID(db *sql.DB, id int64) (Tarea, error) {
	var t Tarea
	err := db.QueryRow(
		"SELECT id, titulo, descripcion, estado, usuario_id, created_at FROM tareas WHERE id = ?", id,
	).Scan(&t.ID, &t.Titulo, &t.Descripcion, &t.Estado, &t.UsuarioID, &t.CreatedAt)
	return t, err
}

// filtrosTarea agrupa los filtros opcionales del listado.
type filtrosTarea struct {
	usuarioID *int64
	estado    *string
}

func listarTareasDB(db *sql.DB, filtros filtrosTarea) ([]Tarea, error) {
	consulta := "SELECT id, titulo, descripcion, estado, usuario_id, created_at FROM tareas WHERE 1=1"
	var args []any

	if filtros.usuarioID != nil {
		consulta += " AND usuario_id = ?"
		args = append(args, *filtros.usuarioID)
	}
	if filtros.estado != nil {
		consulta += " AND estado = ?"
		args = append(args, *filtros.estado)
	}
	consulta += " ORDER BY id"

	filas, err := db.Query(consulta, args...)
	if err != nil {
		return nil, err
	}
	defer filas.Close()

	tareas := []Tarea{}
	for filas.Next() {
		var t Tarea
		if err := filas.Scan(&t.ID, &t.Titulo, &t.Descripcion, &t.Estado, &t.UsuarioID, &t.CreatedAt); err != nil {
			return nil, err
		}
		tareas = append(tareas, t)
	}
	return tareas, filas.Err()
}

func cambiarEstadoDB(db *sql.DB, id int64, estado string) (bool, error) {
	resultado, err := db.Exec("UPDATE tareas SET estado = ? WHERE id = ?", estado, id)
	if err != nil {
		return false, err
	}
	afectadas, err := resultado.RowsAffected()
	if err != nil {
		return false, err
	}
	return afectadas > 0, nil
}

// ---------------------------------------------------------------------------
// Service (reglas de negocio)
// ---------------------------------------------------------------------------

func estadoValido(estado string) bool {
	switch estado {
	case EstadoPendiente, EstadoEnProgreso, EstadoCompletada:
		return true
	}
	return false
}

func crearTarea(db *sql.DB, titulo, descripcion string, usuarioID int64) (Tarea, error) {
	tarea := Tarea{
		Titulo:      titulo,
		Descripcion: descripcion,
		Estado:      EstadoPendiente,
		UsuarioID:   usuarioID,
	}

	if strings.TrimSpace(tarea.Titulo) == "" {
		return tarea, ErrorValidacion{"el titulo es obligatorio"}
	}

	// ÚNICO punto de acoplamiento con la feature usuarios: verificar que
	// exista el usuario dueño de la tarea.
	existe, err := usuarios.Existe(db, tarea.UsuarioID)
	if err != nil {
		return tarea, err
	}
	if !existe {
		return tarea, usuarios.ErrUsuarioNoEncontrado
	}

	if err := crearTareaDB(db, &tarea); err != nil {
		return tarea, err
	}
	return obtenerTareaPorID(db, tarea.ID)
}

func listarTareas(db *sql.DB, usuarioID *int64, estado *string) ([]Tarea, error) {
	if estado != nil && !estadoValido(*estado) {
		return nil, ErrorValidacion{"estado invalido"}
	}
	return listarTareasDB(db, filtrosTarea{usuarioID: usuarioID, estado: estado})
}

func listarTareasDeUsuario(db *sql.DB, usuarioID int64) ([]Tarea, error) {
	existe, err := usuarios.Existe(db, usuarioID)
	if err != nil {
		return nil, err
	}
	if !existe {
		return nil, usuarios.ErrUsuarioNoEncontrado
	}
	return listarTareasDB(db, filtrosTarea{usuarioID: &usuarioID})
}

func cambiarEstado(db *sql.DB, id int64, estado string) (Tarea, error) {
	var tarea Tarea

	if !estadoValido(estado) {
		return tarea, ErrorValidacion{"estado invalido"}
	}

	existe, err := cambiarEstadoDB(db, id, estado)
	if err != nil {
		return tarea, err
	}
	if !existe {
		return tarea, ErrTareaNoEncontrada
	}
	return obtenerTareaPorID(db, id)
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
	case errors.Is(err, usuarios.ErrUsuarioNoEncontrado):
		responderError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrTareaNoEncontrada):
		responderError(w, http.StatusNotFound, err.Error())
	default:
		responderError(w, http.StatusInternalServerError, err.Error())
	}
}

// RegistrarRutas agrega los endpoints de esta feature al mux de main.go.
func RegistrarRutas(mux *http.ServeMux, db *sql.DB) {
	mux.HandleFunc("POST /tareas", func(w http.ResponseWriter, r *http.Request) {
		var entrada struct {
			Titulo      string `json:"titulo"`
			Descripcion string `json:"descripcion"`
			UsuarioID   int64  `json:"usuario_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&entrada); err != nil {
			responderError(w, http.StatusBadRequest, "JSON invalido")
			return
		}
		tarea, err := crearTarea(db, entrada.Titulo, entrada.Descripcion, entrada.UsuarioID)
		if err != nil {
			responderSegunError(w, err)
			return
		}
		responderJSON(w, http.StatusCreated, tarea)
	})

	mux.HandleFunc("GET /tareas", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		var usuarioID *int64
		if v := query.Get("usuario_id"); v != "" {
			id, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				responderError(w, http.StatusBadRequest, "usuario_id invalido")
				return
			}
			usuarioID = &id
		}

		var estado *string
		if v := query.Get("estado"); v != "" {
			estado = &v
		}

		tareas, err := listarTareas(db, usuarioID, estado)
		if err != nil {
			responderSegunError(w, err)
			return
		}
		responderJSON(w, http.StatusOK, tareas)
	})

	mux.HandleFunc("PATCH /tareas/{id}/estado", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			responderError(w, http.StatusBadRequest, "id invalido")
			return
		}

		var entrada struct {
			Estado string `json:"estado"`
		}
		if err := json.NewDecoder(r.Body).Decode(&entrada); err != nil {
			responderError(w, http.StatusBadRequest, "JSON invalido")
			return
		}

		tarea, err := cambiarEstado(db, id, entrada.Estado)
		if err != nil {
			responderSegunError(w, err)
			return
		}
		responderJSON(w, http.StatusOK, tarea)
	})

	mux.HandleFunc("GET /usuarios/{id}/tareas", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			responderError(w, http.StatusBadRequest, "id invalido")
			return
		}

		tareas, err := listarTareasDeUsuario(db, id)
		if err != nil {
			responderSegunError(w, err)
			return
		}
		responderJSON(w, http.StatusOK, tareas)
	})
}
