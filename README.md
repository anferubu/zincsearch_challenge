# ZincSearch Challenge

Proyecto de Indexación y Visualización de Correos Electrónicos

Este repositorio contiene la solución a un desafío de aprendizaje, cuyo objetivo es construir un sistema que permita indexar, analizar y visualizar datos de correos electrónicos. A continuación, se describe el propósito de cada parte del proyecto y las tecnologías utilizadas.

---

## Contenido del Proyecto

### 1. **Indexación de la Base de Datos**
  - Uso de la base de datos de correos de Enron Corp (disponible [aquí](http://www.cs.cmu.edu/~enron/enron_mail_20110402.tgz)).
  - Indexación de los correos utilizando la herramienta [ZincSearch](https://zincsearch.com/).

### 2. **Profiling**
  - Análisis de rendimiento del programa de indexación utilizando herramientas de diagnóstico de Go.
  - Generación de gráficos de profiling para identificar áreas de mejora en la aplicación.

### 3. **Visualización de Datos**
  - Implementación de una API en Go y una interfaz simple en Vue3 para buscar y visualizar los contenidos indexados.

### 4. **Optimización (Opcional)**
   - Uso del análisis de profiling para optimizar el rendimiento del programa.
   - Documentación de las mejoras aplicadas y su impacto.

### 5. **Despliegue (Opcional)**
   - Implementación del sistema en un entorno de producción utilizando AWS o LocalStack.
   - Automatización del despliegue con Terraform.

---

## Tecnologías Utilizadas

Las siguientes tecnologías son esenciales para el desarrollo del proyecto:

- **Base de Datos**: ZincSearch
- **Backend**: Go
- **API Router**: [chi](https://github.com/go-chi/chi)
- **Frontend**: Vue 3
- **Estilos CSS**: Tailwind CSS

> Nota: No se deben usar otras librerías externas en el backend.

---

## Cómo Ejecutar el Proyecto

1. **Preparar el Entorno**
  - Asegúrate de tener Go instalado ([descargar Go](https://go.dev/)).
  - Instala ZincSearch siguiendo la [documentación oficial](https://zincsearch.com/).
  - Configura las variables de entorno necesarias en un archivo `.env`.

  ```bash
  ZINC_HOST=host
  ZINC_USER=user
  ZINC_PASSWORD=password
  ZINC_INDEX=index_name
  ```

2. **Indexar los Correos**
  - Descarga la base de datos desde [este enlace](http://www.cs.cmu.edu/~enron/enron_mail_20110402.tgz).
  - Asegurarse de que ZincSearch se esté ejecutando:

  ```bash
  ./zincsearch.exe
  ```

  - Ejecuta el indexador con el comando:

  ```bash
  ./indexer enron_mail_20110402
  ```

3. **Iniciar el Servidor**
  - Ejecuta la API:

  ```bash
  go run backend/api/main.go
  ```

  - Ejecuta el frontend:

  ```bash
  npm run dev
  ```

   - Accede a la interfaz en `http://localhost:3000`.

4. **Opcional: Despliegue**
   - Sigue los tutoriales de [Terraform](https://developer.hashicorp.com/terraform/tutorials/aws-get-started) para desplegar la solución.


## Resultado final

![Resultado final](graphic_interface.gif)
