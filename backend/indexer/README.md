# Análisis y optimización del indexador
Para el análisis de rendimiento de la aplicación, se utilizó el profiling de Go, que incluyó mediciones del uso de CPU, goroutines y memoria. Este enfoque permitió identificar los cuellos de botella en el procesamiento y la asignación de recursos.

A través de varias iteraciones de prueba y ajuste, se identificaron áreas específicas que necesitaban optimización. Cada iteración del análisis proporcionó información clave para refinar el rendimiento, logrando finalmente un procesamiento más eficiente y satisfactorio.

No se recolectaron datos de cada iteración, pero por ejemplo, en una de las iteraciones, se obtuvieron los siguientes perfiles:

1. **CPU**
```bash
Duration: 591.13s, Total samples = 1.32hrs (805.96%)
Showing nodes accounting for 4675.30s, 98.13% of 4764.27s total
Dropped 653 nodes (cum <= 23.82s)
Showing top 10 nodes out of 37
      flat  flat%   sum%        cum   cum%
  4670.77s 98.04% 98.04%   4675.69s 98.14%  runtime.cgocall
     1.37s 0.029% 98.07%     88.01s  1.85%  bufio.(*Scanner).Scan
     0.66s 0.014% 98.08%   4640.03s 97.39%  main.parseEmail
     0.53s 0.011% 98.09%     97.94s  2.06%  path/filepath.walk
     0.43s 0.009% 98.10%   4394.43s 92.24%  syscall.CreateFile
     0.42s 0.0088% 98.11%   4432.03s 93.03%  os.openFileNolog
     0.31s 0.0065% 98.12%   4432.46s 93.04%  os.OpenFile
     0.28s 0.0059% 98.12%   4641.65s 97.43%  main.processFiles
     0.27s 0.0057% 98.13%   4675.96s 98.15%  runtime.syscall_syscalln
     0.26s 0.0055% 98.13%   4396.16s 92.27%  syscall.Open
```

Se observa que **runtime.cgocall** (llamadas a código C) es el proceso que más consume CPU. Así mismo, puede ser buena idea revisar la función **main.parseEmail** porque también consume mucha CPU.

**syscall.CreateFile** y **syscall.Open** son llamadas al sistema para abrir y crear archivos. Su uso intensivo se explica porque son más de 517.000 archivos a procesar. **path/filepath.walk** realiza el recorrido de archivos/directorios. Una opción para optimizar este punto sería cambiar la estructura de archivos de enron_mail para que sea más simple y fácil de acceder.

2. **Goroutine**
```bash
Showing nodes accounting for 2, 100% of 2 total
      flat  flat%   sum%        cum   cum%
         1 50.00% 50.00%          1 50.00%  runtime.gopark
         1 50.00%   100%          1 50.00%  runtime.goroutineProfileWithLabels
         0     0%   100%          1 50.00%  main.main
         0     0%   100%          1 50.00%  runtime.main
         0     0%   100%          1 50.00%  runtime.pprof_goroutineProfileWithLabels
         0     0%   100%          1 50.00%  runtime/pprof.(*Profile).WriteTo
         0     0%   100%          1 50.00%  runtime/pprof.profileWriter
         0     0%   100%          1 50.00%  runtime/pprof.writeGoroutine
         0     0%   100%          1 50.00%  runtime/pprof.writeRuntimeProfile
         0     0%   100%          1 50.00%  time.Sleep
```

En este caso se recomienda revisar el manejo de goroutines y bloqueos (uso de canales, mutexes, etc.) para asegurarse de que las goroutines no están siendo bloqueadas innecesariamente.

3. **Memory**
```bash
Showing nodes accounting for 5.72MB, 100% of 5.72MB total
Showing top 10 nodes out of 15
      flat  flat%   sum%        cum   cum%
       4MB 69.92% 69.92%        4MB 69.92%  bytes.growSlice
    1.72MB 30.08%   100%     1.72MB 30.08%  runtime/pprof.StartCPUProfile
         0     0%   100%        4MB 69.92%  bytes.(*Buffer).Write
         0     0%   100%        4MB 69.92%  bytes.(*Buffer).grow
         0     0%   100%        4MB 69.92%  encoding/json.(*encodeState).marshal
         0     0%   100%        4MB 69.92%  encoding/json.(*encodeState).reflectValue
         0     0%   100%        4MB 69.92%  encoding/json.Marshal
         0     0%   100%        4MB 69.92%  encoding/json.arrayEncoder.encode
         0     0%   100%        4MB 69.92%  encoding/json.sliceEncoder.encode
         0     0%   100%        4MB 69.92%  encoding/json.stringEncoder
```

**bytes.growSlice** es la función responsable de expandir los tamaños de los slices en Go. Esto puede indicar que la aplicación está haciendo un uso intensivo de slices, lo que lleva a la reasignación y expansión de memoria de manera frecuente. Esto concuerda con la manipulación de grandes cantidades de datos. Optimizar el uso de slices (por ejemplo, preasignando capacidad o utilizando otras estructuras de datos más adecuadas) podría reducir la carga de memoria.

**runtime/pprof.StartCPUProfile** es la función que genera el perfil de CPU y ocupa un espacio considerable en memoria, lo cual es esperado durante la recolección de datos del perfil.

La serialización de datos **json.Marshal** está usando una porción significativa de memoria. Podría ser útil revisar si se puede optimizar el proceso de serialización JSON.

A partir de lo anterior, observamos los siguientes puntos de mejora:

- **Optimizar la manipulación de archivos y las operaciones de E/S**: considerar el uso de técnicas como el almacenamiento en caché o la lectura/ escritura en lotes.
- **Mejorar el procesamiento de correos electrónicos**: investigar la optimización de la función `parseEmail`, posiblemente paralelizando el trabajo o mejorando el algoritmo.
- **Revisar la concurrencia de las goroutines**: asegurarse de que las goroutines no estén bloqueadas innecesariamente, y optimiza el uso de canales o mutexes.
- **Optimizar el uso de memoria**: si las expansiones de slices son una parte crítica, investiga el preajuste de la capacidad de los slices o el uso de estructuras de datos alternativas.

Se decidió profundizar en los bloqueos y cuellos de botella de las concurrencias. Al revisar el código nuevamente, se observó que en la implementación se acumulan emails hasta llegar a 10.000 antes de enviarlos. Si los workers procesan archivos a diferentes velocidades, podrían crearse puntos de espera donde los lotes se acumulan.

Además, solo hay un goroutine enviando lotes a ZincSearch. Si este worker se bloquea o se ralentiza, todos los lotes procesados se acumularán en el canal batchChan.

Se realizaron los siguientes cambios para mejorar ese aspecto:

- Múltiples workers para enviar batches a ZincSearch (5 por defecto).
- Buffer más grande para el canal de batches (workers*2).

Después de implementar estos cambios, se notó una mejora importante en el tiempo total de ejecución, pasando de +9 min a ~32 seg.

1. **CPU**
```bash
Duration: 31.95s, Total samples = 115.37s (361.08%)
Showing nodes accounting for 103.71s, 89.89% of 115.37s total
Dropped 451 nodes (cum <= 0.58s)
Showing top 10 nodes out of 114
      flat  flat%   sum%        cum   cum%
    90.33s 78.30% 78.30%     90.72s 78.63%  runtime.cgocall
     2.82s  2.44% 80.74%      3.08s  2.67%  encoding/json.appendString[go.shape.string]
     2.68s  2.32% 83.06%      2.68s  2.32%  runtime.memmove
     1.57s  1.36% 84.42%      1.57s  1.36%  runtime.memclrNoHeapPointers
     1.52s  1.32% 85.74%      1.53s  1.33%  runtime.stdcall0
     1.27s  1.10% 86.84%      1.30s  1.13%  strings.TrimSpace
     1.01s  0.88% 87.72%      4.10s  3.55%  runtime.mallocgc
     1.01s  0.88% 88.59%      1.01s  0.88%  runtime.stdcall2
     0.89s  0.77% 89.36%      0.97s  0.84%  runtime.findObject
     0.61s  0.53% 89.89%      0.61s  0.53%  runtime.nextFreeFast (inline)
```

Disminuyó el uso de **runtime.cgocall** (98.04% -> 78.30%), aunque sigue siendo un cuello de botella importante.

Aunque no se hizo ningún cambio en **main.parseEmail**, esta ya no aparece como un nodo destacado. Ahora aparece **encoding/json.appendString[go.shape.string]**, lo que indica que ahora una parte del tiempo se dedica a operaciones relacionadas con JSON. Es posible que las mejoras en otros aspectos de la aplicación hayan resaltado el impacto de la serialización/deserialización de JSON.

Así mismo, **runtime.memmove** y **runtime.mallocgc** ahora aparecen como nodos significativos. Esto sugiere que el manejo de memoria es más visible tras las optimizaciones.

2. **Goroutine**
```bash
Showing nodes accounting for 2, 100% of 2 total
      flat  flat%   sum%        cum   cum%
         1 50.00% 50.00%          1 50.00%  runtime.gopark
         1 50.00%   100%          1 50.00%  runtime.goroutineProfileWithLabels
         0     0%   100%          1 50.00%  main.main
         0     0%   100%          1 50.00%  runtime.main
         0     0%   100%          1 50.00%  runtime.pprof_goroutineProfileWithLabels
         0     0%   100%          1 50.00%  runtime/pprof.(*Profile).WriteTo
         0     0%   100%          1 50.00%  runtime/pprof.profileWriter
         0     0%   100%          1 50.00%  runtime/pprof.writeGoroutine
         0     0%   100%          1 50.00%  runtime/pprof.writeRuntimeProfile
         0     0%   100%          1 50.00%  time.Sleep
```

Se se observan grandes cambios con respecto al perfil pasado.

3. **Memory**
```bash
Showing nodes accounting for 169.16MB, 99.40% of 170.17MB total
Dropped 11 nodes (cum <= 0.85MB)
Showing top 10 nodes out of 14
      flat  flat%   sum%        cum   cum%
     168MB 98.72% 98.72%      168MB 98.72%  bytes.growSlice
    1.16MB  0.68% 99.40%     1.16MB  0.68%  runtime/pprof.StartCPUProfile
         0     0% 99.40%      168MB 98.72%  bytes.(*Buffer).Write
         0     0% 99.40%      168MB 98.72%  bytes.(*Buffer).grow
         0     0% 99.40%      168MB 98.72%  encoding/json.(*encodeState).marshal
         0     0% 99.40%      168MB 98.72%  encoding/json.(*encodeState).reflectValue
         0     0% 99.40%      168MB 98.72%  encoding/json.Marshal
         0     0% 99.40%      168MB 98.72%  encoding/json.arrayEncoder.encode
         0     0% 99.40%      168MB 98.72%  encoding/json.sliceEncoder.encode
         0     0% 99.40%      168MB 98.72%  encoding/json.stringEncoder
```

Aunque el tiempo de ejecución se redujo significativamente, el uso de memoria aumentó dramáticamente (5.72 MB -> 170.17 MB), debido a que se están usando más estructuras en memoria para optimizar el tiempo de CPU.

También se observa un aumento en **bytes.growSlice**, lo que indica que las operaciones relacionadas con slices dominan aún más el uso de memoria. Esto sugiere que la aplicación probablemente esté trabajando con slices más grandes o que las operaciones en slices han aumentado debido a las optimizaciones.

El siguiente paso puede ser revisar las operaciones que implican slices y buffers. Podemos considerar preasignar slices con capacidad suficiente para evitar expansiones frecuentes.

Con el procesamiento de datos en paralelo, podemos evaluar si el uso de buffers intermadios puede ser optimizado o minimizado.

En la última optimización, se inicializa el strings.Builder (bodyBuffer) con una capacidad inicial aproximada (4 KB) con el objetivo de disminuir el uso de **bytes.growSlice**. Al hacerlo, vemos una mejora en el uso de memoria de dicha función (168 MB -> 152 MB, aproximadamente un 10% menos).

1. **Memory**
```bash
Showing nodes accounting for 153.72MB, 98.99% of 155.29MB total
Dropped 16 nodes (cum <= 0.78MB)
Showing top 10 nodes out of 15
      flat  flat%   sum%        cum   cum%
     152MB 97.88% 97.88%      152MB 97.88%  bytes.growSlice
    1.72MB  1.11% 98.99%     1.72MB  1.11%  runtime/pprof.StartCPUProfile
         0     0% 98.99%      152MB 97.88%  bytes.(*Buffer).Write
         0     0% 98.99%      152MB 97.88%  bytes.(*Buffer).grow
         0     0% 98.99%      152MB 97.88%  encoding/json.(*encodeState).marshal
         0     0% 98.99%      152MB 97.88%  encoding/json.(*encodeState).reflectValue
         0     0% 98.99%      152MB 97.88%  encoding/json.Marshal
         0     0% 98.99%      152MB 97.88%  encoding/json.arrayEncoder.encode
         0     0% 98.99%      152MB 97.88%  encoding/json.sliceEncoder.encode
         0     0% 98.99%      152MB 97.88%  encoding/json.stringEncoder
```
