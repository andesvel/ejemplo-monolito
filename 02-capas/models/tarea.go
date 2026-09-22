package models

// Tarea es la representación de una tarea en la API (JSON) y en memoria.
type Tarea struct {
	ID          int64  `json:"id"`
	Titulo      string `json:"titulo"`
	Descripcion string `json:"descripcion"`
	Estado      string `json:"estado"`
	UsuarioID   int64  `json:"usuario_id"`
	CreatedAt   string `json:"created_at"`
}

// Los estados posibles, definidos una única vez para todo el programa.
const (
	EstadoPendiente  = "pendiente"
	EstadoEnProgreso = "en_progreso"
	EstadoCompletada = "completada"
)

// EstadoValido verifica que un estado pertenezca a la lista permitida.
// BENEFICIO: al estar en models, tanto el service como el handler reutilizan
// esta validación en vez de repetir la lista de estados en cada archivo.
func EstadoValido(estado string) bool {
	switch estado {
	case EstadoPendiente, EstadoEnProgreso, EstadoCompletada:
		return true
	}
	return false
}
