package services

import (
	"database/sql"
	"strings"

	"capas/models"
	"capas/repository"
)

// TareaService contiene las reglas de negocio de tareas.
// Depende del repositorio de tareas y del de usuarios (necesario para
// validar que el usuario exista antes de crear una tarea).
type TareaService struct {
	repoTareas   *repository.TareaRepository
	repoUsuarios *repository.UsuarioRepository
}

func NuevoTareaService(repoTareas *repository.TareaRepository, repoUsuarios *repository.UsuarioRepository) *TareaService {
	return &TareaService{repoTareas: repoTareas, repoUsuarios: repoUsuarios}
}

// Crear aplica las reglas de creación: titulo obligatorio, usuario existente
// y estado inicial "pendiente".
func (s *TareaService) Crear(titulo, descripcion string, usuarioID int64) (models.Tarea, error) {
	tarea := models.Tarea{
		Titulo:      titulo,
		Descripcion: descripcion,
		Estado:      models.EstadoPendiente,
		UsuarioID:   usuarioID,
	}

	if strings.TrimSpace(tarea.Titulo) == "" {
		return tarea, ErrorValidacion{"el titulo es obligatorio"}
	}

	existe, err := s.repoUsuarios.Existe(tarea.UsuarioID)
	if err != nil {
		return tarea, err
	}
	if !existe {
		return tarea, ErrUsuarioNoEncontrado
	}

	if err := s.repoTareas.Crear(&tarea); err != nil {
		return tarea, err
	}
	return s.repoTareas.ObtenerPorID(tarea.ID)
}

// Listar valida los filtros opcionales y delega en el repository.
func (s *TareaService) Listar(usuarioID *int64, estado *string) ([]models.Tarea, error) {
	if estado != nil && !models.EstadoValido(*estado) {
		return nil, ErrorValidacion{"estado invalido"}
	}
	return s.repoTareas.Listar(repository.FiltrosTarea{UsuarioID: usuarioID, Estado: estado})
}

// ListarPorUsuario devuelve las tareas de un usuario, verificando que exista.
func (s *TareaService) ListarPorUsuario(usuarioID int64) ([]models.Tarea, error) {
	existe, err := s.repoUsuarios.Existe(usuarioID)
	if err != nil {
		return nil, err
	}
	if !existe {
		return nil, ErrUsuarioNoEncontrado
	}
	return s.repoTareas.ListarPorUsuario(usuarioID)
}

// CambiarEstado valida el nuevo estado y actualiza la tarea.
func (s *TareaService) CambiarEstado(id int64, estado string) (models.Tarea, error) {
	var tarea models.Tarea

	if !models.EstadoValido(estado) {
		return tarea, ErrorValidacion{"estado invalido"}
	}

	existe, err := s.repoTareas.CambiarEstado(id, estado)
	if err != nil {
		return tarea, err
	}
	if !existe {
		return tarea, ErrTareaNoEncontrada
	}

	tarea, err = s.repoTareas.ObtenerPorID(id)
	if err == sql.ErrNoRows {
		return tarea, ErrTareaNoEncontrada
	}
	return tarea, err
}
