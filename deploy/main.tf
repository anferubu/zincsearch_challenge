provider "aws" {
  region = "us-east-1"
}

# ----------------------------------------------------------------------------
# Frontend Vue3: S3

# S3
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


# ----------------------------------------------------------------------------
# ZincSearch: EC2

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


# ----------------------------------------------------------------------------
# API Golang: Lambda + API Gateway

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


# Outputs
output "frontend_url" {
  value = "http://${aws_s3_bucket_website_configuration.frontend.website_endpoint}"
}

output "zincsearch_url" {
  value = aws_instance.zincsearch.public_dns
}

output "api_url" {
  value = "${aws_api_gateway_stage.api.invoke_url}"
}
