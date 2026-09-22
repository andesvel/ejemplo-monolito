// Package services: lógica de negocio. No conoce HTTP (ni codecs ni códigos
// de estado) y solo habla con la capa repository.
package services

import (
	"errors"
	"strings"

	"capas/models"
	"capas/repository"
)

// ErrorValidacion marca un error causado por datos que envió el cliente.
// Los handlers lo usan para responder 400 en vez de 500.
type ErrorValidacion struct{ Mensaje string }

func (e ErrorValidacion) Error() string { return e.Mensaje }

// Errores de negocio reutilizables.
var (
	ErrEmailDuplicado      = errors.New("el email ya esta registrado")
	ErrUsuarioNoEncontrado = errors.New("usuario no encontrado")
	ErrTareaNoEncontrada   = errors.New("tarea no encontrada")
)

// UsuarioService contiene las reglas de negocio de usuarios.
type UsuarioService struct {
	repo *repository.UsuarioRepository
}

func NuevoUsuarioService(repo *repository.UsuarioRepository) *UsuarioService {
	return &UsuarioService{repo: repo}
}

// Crear valida las reglas y delega la persistencia en el repository.
// BENEFICIO: esta función se puede probar sin levantar HTTP ni hacer
// requests: se crea el service con una base de datos de prueba y se llama
// directamente a Crear.
func (s *UsuarioService) Crear(nombre, email string) (models.Usuario, error) {
	usuario := models.Usuario{Nombre: nombre, Email: email}

	if strings.TrimSpace(usuario.Nombre) == "" {
		return usuario, ErrorValidacion{"el nombre es obligatorio"}
	}
	if !strings.Contains(usuario.Email, "@") {
		return usuario, ErrorValidacion{"email invalido"}
	}

	if err := s.repo.Crear(&usuario); err != nil {
		// La regla "email único" la aplica la BD; aquí la traducimos a un
		// error de negocio con significado para quien llame.
		if strings.Contains(err.Error(), "UNIQUE") {
			return usuario, ErrEmailDuplicado
		}
		return usuario, err
	}
	return usuario, nil
}

// Listar devuelve todos los usuarios.
func (s *UsuarioService) Listar() ([]models.Usuario, error) {
	return s.repo.Listar()
}

// Existe verifica que un usuario exista (lo usa TareaService).
func (s *UsuarioService) Existe(id int64) (bool, error) {
	return s.repo.Existe(id)
}
