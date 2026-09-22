# monolito-tipos-demo

Repositorio **educativo**: la misma aplicación pequeña (gestor de tareas de
equipo) implementada de 3 formas distintas para comparar en vivo
organizaciones de código dentro de un monolito. La tecnología es idéntica en
las tres versiones (Go con `net/http` de la librería estándar, SQLite vía
`modernc.org/sqlite`, SQL crudo, sin ORM); la única variable es **cómo se
organiza el código**. La carpeta `04-distribuido-antipatron` agrega, sin
código, el escenario que se debe evitar: separar procesos sin separar datos.

## Cómo correr cada versión

Requisitos: Go 1.22+ (el ruteo usa patrones `PATCH /tareas/{id}/estado`).

Cada versión usa el mismo `schema.sql` (ubicado en la raíz del repo) y crea
su propio `demo.db` al arrancar. Correr **desde la carpeta de cada versión**
(porque buscan el schema en `../schema.sql`):

```bash
# 01-tradicional — todo en un archivo (puerto 8081)
cd 01-tradicional && go run main.go

# 02-capas — capas horizontales handlers/services/repository/models (puerto 8082)
cd 02-capas && go run main.go

# 03-modular — features verticales por dominio (puerto 8083)
cd 03-modular && go run main.go
```

Los tres exponen exactamente los mismos endpoints:

```
POST   /usuarios                  {"nombre": "...", "email": "..."}
GET    /usuarios
POST   /tareas                    {"titulo": "...", "descripcion": "...", "usuario_id": 1}
GET    /tareas?usuario_id=1&estado=pendiente
PATCH  /tareas/{id}/estado        {"estado": "en_progreso"}
GET    /usuarios/{id}/tareas
```

## Comparación

| | 01-tradicional | 02-capas | 03-modular | 04-distribuido-antipatron |
|---|---|---|---|---|
| **Estructura** | 1 archivo (`main.go`) con 6 handlers + helpers | 4 paquetes horizontales: `models/`, `repository/` (2 repos + conexión), `services/` (2 services), `handlers/` (2 handlers) + `main.go` que conecta todo | 2 paquetes verticales: `features/usuarios/` y `features/tareas/`, cada uno con su model+SQL+reglas+HTTP en un solo archivo | 2 servicios desplegados por separado + 1 archivo SQLite compartido (solo documento, sin código) |
| **Organización** | Por *endpoint*: cada función HTTP es un mundo autónomo con SQL y reglas adentro | Por *tarea técnica*: "todo el SQL aquí", "toda la lógica allá"; un dominio (tareas) está repartido en 4 archivos de 4 carpetas distintas | Por *dominio*: lo que cambia junto vive junto; `tareas` depende de `usuarios` pero no al revés (frontera explícita en `usuarios.Existe`) | "Organización" solo a nivel de despliegue: el código sigue acoplado por el esquema de la BD |
| **Mantenibilidad** | Aceptable con 6 endpoints, decae rápido: cambiar la validación de estado obliga a buscarla dentro de 2 handlers; el listado acumula `if` de filtros con SQL concatenado a mano | Cambiar reglas o SQL tiene lugar único (service o repository); el costo es la dispersión: agregar algo a tareas implica saltar entre 4 archivos y entender cómo se conectan en `main.go` | Cambiar tareas = abrir 1 archivo; el riesgo es la duplicación deliberada entre features (helpers, errores, validación repetidos) para mantener fronteras limpias | Cualquier cambio de esquema en un servicio rompe al otro en runtime; hay que coordinar deploys para un "ALTER TABLE" |
| **Testing** | Solo end-to-end: no se puede probar una regla de negocio sin levantar HTTP, porque no existe como unidad separada | Cada capa se prueba sola: services sin HTTP, repositories sin handlers (el service recibe repos inyectados, fáciles de reemplazar en un test) | Igual que capas a nivel de feature: el service de tareas se prueba con la BD sola; en la práctica, un test de tareas también depende del código de usuarios | "Testing" entre servicios: un test de tareas necesita que el archivo compartido esté en el estado que otro servicio dejó; sin contratos, los tests son frágiles |
| **Acoplamiento** | Máximo y oculto: el driver de BD, las reglas y el formato HTTP conviven dentro de cada función; 6 copias del patrón SELECT-Scan-responder | Bajo entre capas (interfaces pequeñas: repos y services inyectados), alto entre dominios: `tareas` y `usuarios` se tocan a través de 4 capas a la vez | Bajo y visible entre features (un punto: `usuarios.Existe`), algo de duplicación interna como precio | Acoplamiento por esquema de BD escondido bajo una apariencia de servicios independientes: lo peor de ambos mundos |

## Cómo agregar un campo nuevo a Tarea (ejemplo: `prioridad`)

Este es el experimento más útil para la demo en vivo. Supongamos agregar el
campo `prioridad` a las tareas (columna en `schema.sql` + campo en el JSON de
entrada y salida). Esto hay que tocar en cada versión:

**01-tradicional — 1 archivo, pero en 4 lugares dentro de él.**
`main.go` es un solo archivo, y en él hay que editar: el `INSERT` de
`crearTarea`, el `SELECT` + `Scan` de `listarTareas` (y los de
`tareasDeUsuario` y `cambiarEstadoTarea` si el campo aparece en esas
respuestas), y decidir a mano dónde entra el dato en los `map[string]any` de
respuesta. Fácil de *empezar* (todo está ahí), fácil de *olvidar un lugar*:
no hay nada que te avise que `tareasDeUsuario` quedó sin el campo nuevo.

**02-capas — 4 archivos, uno por capa.**
`models/tarea.go` (el campo y quizá una constante/validación),
`repository/tarea_repository.go` (INSERT, SELECT y Scan en todos los métodos
que tocan la tabla), `services/tarea_service.go` (solo si el campo tiene
regla de negocio, ej. prioridad por defecto) y
`handlers/tarea_handler.go` (el struct de entrada). El compilador ayuda:
mientras el struct de `models` no tenga el campo, los demás archivos fallan
al compilar. Cada cambio tiene un lugar canónico, pero la misma feature vive
repartida en 4 carpetas.

**03-modular — 1 archivo.**
`features/tareas/tareas.go`: el campo en el struct, el INSERT, el SELECT/Scan
y (si aplica) la regla de negocio están todos juntos. Un solo `diff`, cero
saltos entre carpetas. Esta es la comparación estrella de la demo: lo que en
capas son 4 archivos en 4 carpetas, aquí es una carpeta con un archivo.

En las tres versiones, además, `schema.sql` raíz (compartido) gana la
columna nueva — eso es intencional: el esquema es el único punto común que
el ejercicio fija para poder comparar el resto.
