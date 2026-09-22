# 04 — Distribuido (antipatrón): dos servicios, UNA base de datos

Este documento **no tiene código** a propósito: describe lo que pasaría si
tomamos la app de la carpeta `02-capas` (o `03-modular`) y la "separamos"
solo a medias — dos procesos desplegados por separado, pero ambos leyendo y
escribiendo **directamente sobre el mismo archivo SQLite compartido**.

## Diagrama

```
     ┌─────────────────────────┐          ┌─────────────────────────┐
      │   servicio-usuarios     │          │    servicio-tareas      │
      │   (puerto 9001)         │          │    (puerto 9002)        │
      │                         │          │                         │
      │  handlers/              │          │  handlers/              │
      │  services/              │          │  services/              │
      │  repository/  ──────────────┐  ┌───────────  repository/       │
      └─────────────────────────┘  │  │   └─────────────────────────┘
                                   │  │
                              ┌────▼──▼─────┐
                              │  demo.db    │        ← EL MISMO archivo
                              │ (SQLite)    │           en el MISMO disco,
                              └─────────────┘           compartido por
                                    ▲                    ambos procesos
                                    │
                        SQL crudo contra el schema ajeno
                (tareas hace INSERT/JOIN sobre la tabla usuarios)
```

Qué está "distribuido" aquí: **los procesos**. Qué no: **los datos**.
Cada servicio tiene su propio binario, su propio puerto, su propio ciclo de
despliegue… pero los dos abren `demo.db` con una ruta de archivo. El único
acoplamiento real (el esquema de la base) nunca se rompió.

## Tres problemas concretos que esto causa en la práctica

1. **Contención y bloqueo de escrituras sobre un único archivo.**
   SQLite permite un solo escritor a la vez. Con dos servicios independientes
   escribiendo (usuarios creando cuentas, tareas actualizando estados), las
   transacciones empiezan a chocar: errores `database is locked`, esperas y
   reintentos que no se pueden ajustar por servicio, y peor rendimiento que
   ejecutando todo en un solo proceso. El lock es global al archivo: los dos
   servicios compiten por el mismo candado aunque trabajen sobre tablas distintas.

2. **Acoplamiento por esquema, sin contrato: cualquier `ALTER TABLE` rompe el
   otro servicio.**
   Si `servicio-usuarios` agrega una columna `NOT NULL` o renombra un campo,
   `servicio-tareas` — que hace sus propios `INSERT`/`SELECT` sobre la tabla
   ajena — se rompe en runtime, no en compilación. Y nadie lo avisa: no hay
   API con contrato, ni migraciones coordinadas, solo "los dos conocen el
   mismo schema.sql". Es el acoplamiento del monolito tradicional, pero ahora
   cruzando límites de red y despliegue, donde es mucho más caro de detectar.

3. **Imposible escalar ni desplegar de forma independiente.**
   La promesa de separar en servicios es escalar `servicio-tareas` (que
   recibe más carga) sin tocar el resto. Aquí no se puede: el archivo SQLite
   vive en el disco de una máquina concreta, así que ambos servicios tienen
   que correr en ese mismo host; no hay réplicas, no hay segundo pod, no hay
   deploy por separado sin corte, y un backup/actualización del archivo
   afecta a los dos. Además, si el host se cae, se caen "los dos
   microservicios": la separación compró complejidad operativa sin comprar
   ninguna de las garantías de la distribución real (base de datos servidor
   tipo Postgres con acceso por red, o comunicación por API entre servicios).
