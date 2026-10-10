# No NAT gateway and no internet gateway on purpose: the Lambda functions
# only talk to RDS Proxy/Aurora inside this VPC, never to the public
# internet, and Lambda's own delivery of logs/metrics to CloudWatch does not
# route through the function's VPC ENIs — so there is nothing here that
# needs internet egress. This keeps the VPC free of NAT Gateway cost.

data "aws_availability_zones" "available" {
  state = "available"
}

resource "aws_vpc" "this" {
  cidr_block           = var.vpc_cidr
  enable_dns_support   = true
  enable_dns_hostnames = true

  tags = {
    Name = "${var.project_name}-${var.environment}"
  }
}

resource "aws_subnet" "private" {
  count             = length(var.private_subnet_cidrs)
  vpc_id            = aws_vpc.this.id
  cidr_block        = var.private_subnet_cidrs[count.index]
  availability_zone = data.aws_availability_zones.available.names[count.index]

  tags = {
    Name = "${var.project_name}-${var.environment}-private-${count.index}"
  }
}

resource "aws_db_subnet_group" "this" {
  name       = "${var.project_name}-${var.environment}"
  subnet_ids = aws_subnet.private[*].id

  tags = {
    Name = "${var.project_name}-${var.environment}"
  }
}

# Each security group is self-contained with CIDR-based rules (rather than
# SG-to-SG references) so there's no circular dependency between them to
# untangle: Lambda and RDS Proxy ENIs both live in the known private
# subnets, so scoping by those CIDRs is equivalent to scoping by SG, without
# the ordering problem of two groups each referencing the other.

resource "aws_security_group" "lambda" {
  name        = "${var.project_name}-${var.environment}-lambda"
  description = "API and migrate Lambda functions"
  vpc_id      = aws_vpc.this.id

  egress {
    description = "MySQL to RDS Proxy"
    from_port   = 3306
    to_port     = 3306
    protocol    = "tcp"
    cidr_blocks = var.private_subnet_cidrs
  }

  tags = {
    Name = "${var.project_name}-${var.environment}-lambda"
  }
}

resource "aws_security_group" "rds_proxy" {
  name        = "${var.project_name}-${var.environment}-rds-proxy"
  description = "RDS Proxy in front of Aurora"
  vpc_id      = aws_vpc.this.id

  ingress {
    description = "MySQL from Lambda"
    from_port   = 3306
    to_port     = 3306
    protocol    = "tcp"
    cidr_blocks = var.private_subnet_cidrs
  }

  egress {
    description = "MySQL to Aurora"
    from_port   = 3306
    to_port     = 3306
    protocol    = "tcp"
    cidr_blocks = var.private_subnet_cidrs
  }

  tags = {
    Name = "${var.project_name}-${var.environment}-rds-proxy"
  }
}

resource "aws_security_group" "aurora" {
  name        = "${var.project_name}-${var.environment}-aurora"
  description = "Aurora Serverless v2 cluster"
  vpc_id      = aws_vpc.this.id

  ingress {
    description = "MySQL from RDS Proxy"
    from_port   = 3306
    to_port     = 3306
    protocol    = "tcp"
    cidr_blocks = var.private_subnet_cidrs
  }

  tags = {
    Name = "${var.project_name}-${var.environment}-aurora"
  }
}
