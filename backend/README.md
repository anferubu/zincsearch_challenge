# Notas del Backend

## Indexador
El indexador se encuentra en el directorio /indexer/.

Primero, generamos el binario:

```bash
$ go build -o indexer/indexer ./indexer
```

Luego, con el binario podemos indexar los correos (recuerde que ZincSearch debe estar corriendo):

```bash
$ ./indexer/indexer enron_mail_20110402/maildir/
```

---

## API REST
La API la ejecutamos con el siguiente comando:

```bash
$ go run api/main.go
```

La implementación actual solo contempla un endpoint:

- /api/emails

El cual recibe una serie de parámetros de consulta para filtrar los correos:

- query: permite buscar palabras en el asunto o cuerpo del correo.
- from: permite filtrar correos por el remitente.
- to: permite filtrar correos por el destinatario.
- dateTime: permite filtrar correos por la fecha de envío.

- sortBy: permite filtrar correos por alguno de sus campos (to, from, dateTime, ...).
- sortDir: permite indicar la dirección de ordenamiento (asc, desc).

- page: indica la página (por defecto, 1).
- pageSize: define el tamaño de la página (por defecto, 10).

Así, por ejemplo, podemos obtener los correos de mark.taylor@enron.com que contengan la palabra clave 'Schuyler' con la siguiente ruta:

```bash
http://localhost:3000/api/emails?query=Schuyler&from=mark.taylor@enron.com
```
