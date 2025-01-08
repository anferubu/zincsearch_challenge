# Despliegue de la aplicación en AWS
La aplicación se desplegó bajo una arquitectura que prioriza la simplicidad y los servicios incluidos en el Free Tier de AWS. La arquitectura propuesta es:

- S3: para almacenar los archivos del frontend.
- Lambda + API Gateway: para desplegar la API hecha en Go.
- EC2: para desplegar ZincSearch.

## Infraestructura como código para S3
Primero, se crearon los archivos para producción del proyecto hecho en Vue3 con el comando:

```bash
$ npm run build
```

Dicho comando crea un directorio /dist/ que contiene los archivos optimizados para producción. Luego, con Terraform se indica la creación de un bucket "zincsearch-challenge-frontend" y se configura como hosting estático para que apunte al archivo s3://zincsearch-challenge-frontend/index.html.

Así mismo, se suben los archivos del directorio local /dist/ al bucket S3.

```txt
resource "aws_s3_bucket" "frontend" {
  bucket = "zincsearch-challenge-frontend"
  force_destroy = true
}

resource "aws_s3_bucket_website_configuration" "frontend" {
  bucket = aws_s3_bucket.frontend.id

  index_document {
    suffix = "index.html"
  }
}

resource "aws_s3_bucket_public_access_block" "frontend" {
  bucket = aws_s3_bucket.frontend.id

  block_public_acls       = false
  block_public_policy     = false
  ignore_public_acls      = false
  restrict_public_buckets = false
}

resource "aws_s3_bucket_policy" "frontend" {
  bucket = aws_s3_bucket.frontend.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "PublicReadGetObject"
        Effect    = "Allow"
        Principal = "*"
        Action    = "s3:GetObject"
        Resource  = "${aws_s3_bucket.frontend.arn}/*",
      }
    ]
  })
  depends_on = [aws_s3_bucket_public_access_block.frontend]
}

resource "aws_s3_object" "frontend_files" {
  for_each = fileset("frontend/dist", "**/*")

  bucket = aws_s3_bucket.frontend.id
  key    = each.value
  source = "frontend/dist/${each.value}"

  content_type = lookup({
    "html" = "text/html",
    "js"   = "application/javascript",
    "css"  = "text/css",
    "png"  = "image/png",
    "jpg"  = "image/jpeg",
    "svg"  = "image/svg+xml",
    "ico"  = "image/x-icon"
  }, split(".", each.value)[length(split(".", each.value)) - 1], "application/octet-stream")
}
```

> Se tuvo que modificar en index.html y el JS para apuntar a los archivos de S3 y al endpoint de API Gateway, respectivamente.

## Infraestructura como código para ZincSearch
Se utilizó una instancia EC2 Ubuntu para desplegar ZincSearch. Además, para que la conexión funcionara a través de los puertos 22 (SSH) y 4080 (ZincSearch) se tuvo que configurar una VPC, subnet, tabla de enrutamiento y grupo de seguridad.

También se creó un key pair para poder conectarnos posteriormente vía SSH.

```bash
$ aws ec2 create-key-pair --key-name my-key --query 'KeyMaterial' --output text | tee key.pem
$ ssh-keygen -y -f key.pem > key.pub
```

```txt
# VPC Configuration
resource "aws_vpc" "main" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name = "main-vpc"
  }
}
resource "aws_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id

  tags = {
    Name = "main-igw"
  }
}
resource "aws_subnet" "main" {
  vpc_id                  = aws_vpc.main.id
  cidr_block              = "10.0.1.0/24"
  map_public_ip_on_launch = true

  tags = {
    Name = "main-subnet"
  }
}
resource "aws_route_table" "main" {
  vpc_id = aws_vpc.main.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.main.id
  }

  tags = {
    Name = "main-rt"
  }
}
resource "aws_route_table_association" "main" {
  subnet_id      = aws_subnet.main.id
  route_table_id = aws_route_table.main.id
}

resource "aws_security_group" "zincsearch" {
  name_prefix = "zincsearch-sg"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 4080
    to_port     = 4080
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_key_pair" "zinc_key" {
  key_name   = "key"
  public_key = file("key.pub")
}

resource "aws_instance" "zincsearch" {
  ami                    = "ami-0c7217cdde317cfec"  # Amazon Linux 2023
  instance_type          = "t2.micro"
  key_name               = aws_key_pair.zinc_key.key_name
  subnet_id              = aws_subnet.main.id
  vpc_security_group_ids = [aws_security_group.zincsearch.id]

  user_data = <<-EOF
    #!/bin/bash
    # Actualizar el sistema
    yum update -y

    # Descargar ZincSearch
    wget https://github.com/zinclabs/zincsearch/releases/download/v0.4.10/zincsearch_0.4.10_linux_x86_64.tar.gz
    tar -xzf zincsearch_0.4.10_linux_x86_64.tar.gz
    mv zincsearch /usr/local/bin/zincsearch

    # Crear el directorio de datos y configurar permisos
    mkdir -p /var/lib/zincsearch/data
    chmod -R 777 /var/lib/zincsearch/data

    # Crear un archivo de servicio para ZincSearch
    cat <<EOT > /etc/systemd/system/zincsearch.service
    [Unit]
    Description=ZincSearch Service
    After=network.target

    [Service]
    Environment="ZINC_FIRST_ADMIN_USER=admin"
    Environment="ZINC_FIRST_ADMIN_PASSWORD=admin123"
    Environment="ZINC_DATA_PATH=/var/lib/zincsearch/data"
    ExecStart=/usr/local/bin/zincsearch
    Restart=always
    User=root
    WorkingDirectory=/var/lib/zincsearch

    [Install]
    WantedBy=multi-user.target
    EOT

    # Recargar systemd, habilitar y arrancar el servicio
    systemctl daemon-reload
    systemctl enable zincsearch.service
    systemctl start zincsearch.service
  EOF

  tags = {
    Name = "zincsearch"
  }
}
```

Luego, se ejecutó el indexador apuntando a la dirección de la instancia EC2 para crear el índice y subir los emails. Hay que tener en cuenta que la instancia es t2.micro, por lo que su capacidad es bastante reducida. Por ello, no se indexaron todos los emails, solo algunos para probar en producción.

A veces, ya sea la instancia como tal o el servicio ZincSearch se quedaban congelados, por lo que tocaba reiniciar la instancia o conectarse por SSH para revisar el servicio:

```bash
$ ssh -i "key.pem" ubuntu@ec2-xx-xxx-xxx-xx.compute-1.amazonaws.com
```

## Infraestructura como código para la API de Golang
Para el caso de la API, se transformó el código al formato de las funciones Lambda de AWS, se convirtió en un binario y se empaquetó como ZIP para subirlo al servicio AWS Lambda.

```bash
$ GOOS=linux GOARCH=amd64 go build -o bootstrap main.go services.go
$ zip function.zip bootstrap
```

También se configuró una API Gateway conectada a la función Lambda para poder hacer peticiones HTTP:

```txt
# Lambda
resource "aws_lambda_function" "email_search" {
  filename         = "backend/function.zip"
  function_name    = "email-search"
  role            = aws_iam_role.lambda_role.arn
  handler         = "bootstrap"
  runtime         = "provided.al2"

  environment {
    variables = {
      ZINC_HOST     = var.zinc_host
      ZINC_INDEX    = var.zinc_index
      ZINC_USER     = var.zinc_user
      ZINC_PASSWORD = var.zinc_password
    }
  }
}

resource "aws_iam_role" "lambda_role" {
  name = "email_search_lambda_role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "lambda.amazonaws.com"
        }
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "lambda_basic" {
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
  role       = aws_iam_role.lambda_role.name
}

# API Gateway configuration
resource "aws_apigatewayv2_api" "email_search" {
  name          = "email-search-api"
  protocol_type = "HTTP"
}

resource "aws_apigatewayv2_stage" "email_search" {
  api_id = aws_apigatewayv2_api.email_search.id
  name   = "prod"
  auto_deploy = true
}

resource "aws_apigatewayv2_integration" "email_search" {
  api_id           = aws_apigatewayv2_api.email_search.id
  integration_type = "AWS_PROXY"

  integration_uri    = aws_lambda_function.email_search.invoke_arn
  integration_method = "POST"
}

resource "aws_apigatewayv2_route" "email_search" {
  api_id    = aws_apigatewayv2_api.email_search.id
  route_key = "GET /api/emails"
  target    = "integrations/${aws_apigatewayv2_integration.email_search.id}"
}

resource "aws_lambda_permission" "api_gw" {
  statement_id  = "AllowExecutionFromAPIGateway"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.email_search.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.email_search.execution_arn}/*/*"
}

variable "zinc_host" {
  type = string
}

variable "zinc_index" {
  type = string
}

variable "zinc_user" {
  type = string
}

variable "zinc_password" {
  type = string
  sensitive = true
}
```
