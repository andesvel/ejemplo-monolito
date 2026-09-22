package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"capas/services"
)

// TareaHandler agrupa los endpoints de tareas con su service inyectado.
type TareaHandler struct {
	service *services.TareaService
}

func NuevoTareaHandler(service *services.TareaService) *TareaHandler {
	return &TareaHandler{service: service}
}

// Registrar agrega las rutas de tareas al mux. Incluye la ruta anidada
// GET /usuarios/{id}/tareas: el recurso que se devuelve son tareas, así que
// lo atiende este handler aunque el "id" sea de usuario.
func (h *TareaHandler) Registrar(mux *http.ServeMux) {
	mux.HandleFunc("POST /tareas", h.Crear)
	mux.HandleFunc("GET /tareas", h.Listar)
	mux.HandleFunc("PATCH /tareas/{id}/estado", h.CambiarEstado)
	mux.HandleFunc("GET /usuarios/{id}/tareas", h.TareasDeUsuario)
}

// Crear: POST /tareas
func (h *TareaHandler) Crear(w http.ResponseWriter, r *http.Request) {
	var entrada struct {
		Titulo      string `json:"titulo"`
		Descripcion string `json:"descripcion"`
		UsuarioID   int64  `json:"usuario_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&entrada); err != nil {
		responderError(w, http.StatusBadRequest, "JSON invalido")
		return
	}

	tarea, err := h.service.Crear(entrada.Titulo, entrada.Descripcion, entrada.UsuarioID)
	if err != nil {
		responderSegunError(w, err)
		return
	}
	responderJSON(w, http.StatusCreated, tarea)
}

// Listar: GET /tareas?usuario_id=&estado=
// Solo traduce los parámetros de query a los tipos que espera el service;
// la validación del estado la hace el service, no aquí.
func (h *TareaHandler) Listar(w http.ResponseWriter, r *http.Request) {
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

	tareas, err := h.service.Listar(usuarioID, estado)
	if err != nil {
		responderSegunError(w, err)
		return
	}
	responderJSON(w, http.StatusOK, tareas)
}

// CambiarEstado: PATCH /tareas/{id}/estado
func (h *TareaHandler) CambiarEstado(w http.ResponseWriter, r *http.Request) {
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

	tarea, err := h.service.CambiarEstado(id, entrada.Estado)
	if err != nil {
		responderSegunError(w, err)
		return
	}
	responderJSON(w, http.StatusOK, tarea)
}

// TareasDeUsuario: GET /usuarios/{id}/tareas
func (h *TareaHandler) TareasDeUsuario(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		responderError(w, http.StatusBadRequest, "id invalido")
		return
	}

	tareas, err := h.service.ListarPorUsuario(id)
	if err != nil {
		responderSegunError(w, err)
		return
	}
	responderJSON(w, http.StatusOK, tareas)
}
