// Package handlers: capa HTTP. Estos archivos SOLO traducen entre
// HTTP (JSON, códigos de estado, parámetros de URL) y los services.
// No hay SQL ni reglas de negocio aquí.
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"capas/services"
)

// UsuarioHandler agrupa los endpoints de usuarios con su service inyectado.
type UsuarioHandler struct {
	service *services.UsuarioService
}

func NuevoUsuarioHandler(service *services.UsuarioService) *UsuarioHandler {
	return &UsuarioHandler{service: service}
}

// Registrar agrega las rutas de usuarios al mux.
func (h *UsuarioHandler) Registrar(mux *http.ServeMux) {
	mux.HandleFunc("POST /usuarios", h.Crear)
	mux.HandleFunc("GET /usuarios", h.Listar)
}

// Crear: POST /usuarios — parsea la entrada, llama al service, escribe la respuesta.
func (h *UsuarioHandler) Crear(w http.ResponseWriter, r *http.Request) {
	var entrada struct {
		Nombre string `json:"nombre"`
		Email  string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&entrada); err != nil {
		responderError(w, http.StatusBadRequest, "JSON invalido")
		return
	}

	usuario, err := h.service.Crear(entrada.Nombre, entrada.Email)
	if err != nil {
		responderSegunError(w, err)
		return
	}
	responderJSON(w, http.StatusCreated, usuario)
}

// Listar: GET /usuarios
func (h *UsuarioHandler) Listar(w http.ResponseWriter, r *http.Request) {
	usuarios, err := h.service.Listar()
	if err != nil {
		responderSegunError(w, err)
		return
	}
	responderJSON(w, http.StatusOK, usuarios)
}

// responderJSON escribe cualquier dato como respuesta JSON.
func responderJSON(w http.ResponseWriter, codigo int, datos any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(codigo)
	json.NewEncoder(w).Encode(datos)
}

// responderError escribe {"error": "..."} con el código indicado.
func responderError(w http.ResponseWriter, codigo int, mensaje string) {
	responderJSON(w, codigo, map[string]string{"error": mensaje})
}

// responderSegunError traduce errores de negocio a códigos HTTP.
// BENEFICIO: el service devuelve errores con significado (ErrEmailDuplicado,
// ErrorValidacion...) y este único punto los convierte a HTTP; si mañana
// cambia el criterio de códigos, se ajusta solo aquí.
func responderSegunError(w http.ResponseWriter, err error) {
	var validacion services.ErrorValidacion
	switch {
	case errors.As(err, &validacion):
		responderError(w, http.StatusBadRequest, validacion.Mensaje)
	case errors.Is(err, services.ErrEmailDuplicado):
		responderError(w, http.StatusConflict, err.Error())
	case errors.Is(err, services.ErrUsuarioNoEncontrado),
		errors.Is(err, services.ErrTareaNoEncontrada):
		responderError(w, http.StatusNotFound, err.Error())
	default:
		responderError(w, http.StatusInternalServerError, err.Error())
	}
}
