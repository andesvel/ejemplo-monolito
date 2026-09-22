// 01-tradicional: TODO en un solo archivo.
//
// Este es el "mal ejemplo" deliberado de la demo: ruteo, validaciones,
// lógica de negocio y SQL viven mezclados dentro de las mismas funciones.
// Funciona, pero cualquier cambio obliga a tocar este único archivo gigante.
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	_ "modernc.org/sqlite" // driver SQLite sin cgo
)

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("sqlite", "demo.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	schema, err := os.ReadFile("../schema.sql")
	if err != nil {
		log.Fatal(err)
	}
	if _, err := db.Exec(string(schema)); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /usuarios", crearUsuario)
	mux.HandleFunc("GET /usuarios", listarUsuarios)
	mux.HandleFunc("POST /tareas", crearTarea)
	mux.HandleFunc("GET /tareas", listarTareas)
	mux.HandleFunc("PATCH /tareas/{id}/estado", cambiarEstadoTarea)
	mux.HandleFunc("GET /usuarios/{id}/tareas", tareasDeUsuario)

	log.Println("01-tradicional escuchando en http://localhost:8081")
	log.Fatal(http.ListenAndServe(":8081", mux))
}

// Helper mínimo compartido por todos los handlers.
func responderJSON(w http.ResponseWriter, codigo int, datos any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(codigo)
	json.NewEncoder(w).Encode(datos)
}

func responderError(w http.ResponseWriter, codigo int, mensaje string) {
	responderJSON(w, codigo, map[string]string{"error": mensaje})
}

// ---------------------------------------------------------------------------
// Usuarios
// ---------------------------------------------------------------------------

// PROBLEMA: esta función hace 4 trabajos a la vez: decodifica JSON, valida
// reglas de negocio (email con @, obligatoriedad), ejecuta SQL crudo y arma
// la respuesta HTTP. No se puede probar la validación ni la persistencia sin
// levantar un servidor HTTP completo.
func crearUsuario(w http.ResponseWriter, r *http.Request) {
	var entrada struct {
		Nombre string `json:"nombre"`
		Email  string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&entrada); err != nil {
		responderError(w, http.StatusBadRequest, "JSON invalido")
		return
	}

	// Regla de negocio incrustada en el handler...
	if strings.TrimSpace(entrada.Nombre) == "" {
		responderError(w, http.StatusBadRequest, "el nombre es obligatorio")
		return
	}
	if !strings.Contains(entrada.Email, "@") {
		responderError(w, http.StatusBadRequest, "email invalido")
		return
	}

	// ...y SQL crudo también. PROBLEMA: si mañana cambiamos de motor de base
	// de datos (por ejemplo a Postgres), hay que reescribir este bloque aquí
	// y en TODAS las demás funciones de este archivo. No hay capa de acceso
	// a datos: la BD está cocida dentro de cada handler.
	resultado, err := db.Exec(
		"INSERT INTO usuarios (nombre, email) VALUES (?, ?)",
		entrada.Nombre, entrada.Email,
	)
	if err != nil {
		// La unicidad del email la detectamos por el mensaje de error de la BD:
		// la regla vive en el schema y el handler tiene que interpretarla.
		if strings.Contains(err.Error(), "UNIQUE") {
			responderError(w, http.StatusConflict, "el email ya esta registrado")
			return
		}
		responderError(w, http.StatusInternalServerError, err.Error())
		return
	}
	id, _ := resultado.LastInsertId()

	responderJSON(w, http.StatusCreated, map[string]any{
		"id": id, "nombre": entrada.Nombre, "email": entrada.Email,
	})
}

// PROBLEMA: la consulta SQL está pegada a la respuesta HTTP. Si dos vistas
// distintas (web, API móvil, reporte) quisieran listar usuarios, cada una
// copiaría esta misma consulta.
func listarUsuarios(w http.ResponseWriter, r *http.Request) {
	filas, err := db.Query("SELECT id, nombre, email FROM usuarios ORDER BY id")
	if err != nil {
		responderError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer filas.Close()

	usuarios := []map[string]any{}
	for filas.Next() {
		var id int64
		var nombre, email string
		if err := filas.Scan(&id, &nombre, &email); err != nil {
			responderError(w, http.StatusInternalServerError, err.Error())
			return
		}
		usuarios = append(usuarios, map[string]any{"id": id, "nombre": nombre, "email": email})
	}
	responderJSON(w, http.StatusOK, usuarios)
}

// ---------------------------------------------------------------------------
// Tareas
// ---------------------------------------------------------------------------

func crearTarea(w http.ResponseWriter, r *http.Request) {
	var entrada struct {
		Titulo      string `json:"titulo"`
		Descripcion string `json:"descripcion"`
		UsuarioID   int64  `json:"usuario_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&entrada); err != nil {
		responderError(w, http.StatusBadRequest, "JSON invalido")
		return
	}

	// Regla de negocio incrustada: el titulo es obligatorio.
	if strings.TrimSpace(entrada.Titulo) == "" {
		responderError(w, http.StatusBadRequest, "el titulo es obligatorio")
		return
	}

	// Regla de negocio incrustada (otra vez): el usuario debe existir.
	// PROBLEMA: esta misma verificación se repite en crearTarea y podría
	// necesitarse en más lados; al estar en el handler, cualquier handler
	// nuevo debe reescribirla (y alguno se olvidará).
	var existe int
	if err := db.QueryRow("SELECT COUNT(1) FROM usuarios WHERE id = ?", entrada.UsuarioID).Scan(&existe); err != nil {
		responderError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if existe == 0 {
		responderError(w, http.StatusBadRequest, fmt.Sprintf("el usuario %d no existe", entrada.UsuarioID))
		return
	}

	// SQL crudo de nuevo, dentro del handler.
	resultado, err := db.Exec(
		"INSERT INTO tareas (titulo, descripcion, estado, usuario_id) VALUES (?, ?, 'pendiente', ?)",
		entrada.Titulo, entrada.Descripcion, entrada.UsuarioID,
	)
	if err != nil {
		responderError(w, http.StatusInternalServerError, err.Error())
		return
	}
	id, _ := resultado.LastInsertId()

	// Para responder releemos la fila: otra consulta SQL incrustada.
	var creada, creadoEn string
	if err := db.QueryRow(
		"SELECT titulo, created_at FROM tareas WHERE id = ?", id,
	).Scan(&creada, &creadoEn); err != nil {
		responderError(w, http.StatusInternalServerError, err.Error())
		return
	}

	responderJSON(w, http.StatusCreated, map[string]any{
		"id": id, "titulo": creada, "descripcion": entrada.Descripcion,
		"estado": "pendiente", "usuario_id": entrada.UsuarioID, "created_at": creadoEn,
	})
}

// PROBLEMA: los filtros de consulta (?usuario_id= y ?estado=) se resuelven
// concatenando SQL a mano según los parámetros. Cada filtro nuevo (por fecha,
// por texto, etc.) agrega otro `if` y otra variante de la misma consulta.
func listarTareas(w http.ResponseWriter, r *http.Request) {
	consulta := "SELECT id, titulo, descripcion, estado, usuario_id, created_at FROM tareas WHERE 1=1"
	var args []any

	if v := r.URL.Query().Get("usuario_id"); v != "" {
		consulta += " AND usuario_id = ?"
		args = append(args, v)
	}
	if v := r.URL.Query().Get("estado"); v != "" {
		// Validación del estado incrustada en el handler (copiada de crearTarea).
		if v != "pendiente" && v != "en_progreso" && v != "completada" {
			responderError(w, http.StatusBadRequest, "estado invalido")
			return
		}
		consulta += " AND estado = ?"
		args = append(args, v)
	}
	consulta += " ORDER BY id"

	filas, err := db.Query(consulta, args...)
	if err != nil {
		responderError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer filas.Close()

	tareas := []map[string]any{}
	for filas.Next() {
		var id, usuarioID int64
		var titulo, descripcion, estado, creadoEn string
		if err := filas.Scan(&id, &titulo, &descripcion, &estado, &usuarioID, &creadoEn); err != nil {
			responderError(w, http.StatusInternalServerError, err.Error())
			return
		}
		tareas = append(tareas, map[string]any{
			"id": id, "titulo": titulo, "descripcion": descripcion,
			"estado": estado, "usuario_id": usuarioID, "created_at": creadoEn,
		})
	}
	responderJSON(w, http.StatusOK, tareas)
}

// PROBLEMA: la transición de estados ("se puede pasar de pendiente a
// en_progreso, no a cualquier cosa") es lógica de negocio pero está escrita
// dentro del handler HTTP, mezclada con parseo de URL y SQL.
func cambiarEstadoTarea(w http.ResponseWriter, r *http.Request) {
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

	if entrada.Estado != "pendiente" && entrada.Estado != "en_progreso" && entrada.Estado != "completada" {
		responderError(w, http.StatusBadRequest, "estado invalido")
		return
	}

	resultado, err := db.Exec("UPDATE tareas SET estado = ? WHERE id = ?", entrada.Estado, id)
	if err != nil {
		responderError(w, http.StatusInternalServerError, err.Error())
		return
	}
	afectadas, _ := resultado.RowsAffected()
	if afectadas == 0 {
		responderError(w, http.StatusNotFound, "tarea no encontrada")
		return
	}

	var titulo, creadoEn string
	var usuarioID int64
	db.QueryRow("SELECT titulo, usuario_id, created_at FROM tareas WHERE id = ?", id).
		Scan(&titulo, &usuarioID, &creadoEn)

	responderJSON(w, http.StatusOK, map[string]any{
		"id": id, "titulo": titulo, "estado": entrada.Estado,
		"usuario_id": usuarioID, "created_at": creadoEn,
	})
}

func tareasDeUsuario(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		responderError(w, http.StatusBadRequest, "id invalido")
		return
	}

	// SQL incrustado, cuarta copia del patrón SELECT-Scan-responder.
	filas, err := db.Query(
		"SELECT id, titulo, descripcion, estado, usuario_id, created_at FROM tareas WHERE usuario_id = ? ORDER BY id",
		id,
	)
	if err != nil {
		responderError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer filas.Close()

	tareas := []map[string]any{}
	for filas.Next() {
		var tID, usuarioID int64
		var titulo, descripcion, estado, creadoEn string
		if err := filas.Scan(&tID, &titulo, &descripcion, &estado, &usuarioID, &creadoEn); err != nil {
			responderError(w, http.StatusInternalServerError, err.Error())
			return
		}
		tareas = append(tareas, map[string]any{
			"id": tID, "titulo": titulo, "descripcion": descripcion,
			"estado": estado, "usuario_id": usuarioID, "created_at": creadoEn,
		})
	}
	responderJSON(w, http.StatusOK, tareas)
}
