# Optional jump host for emergency Postgres access (plan 17.1 FR-9).
# Prefer AWS Systems Manager Session Manager (no inbound SSH required).

locals {
  enable_bastion = coalesce(var.enable_bastion, local.is_production)
}

data "aws_ami" "amazon_linux_2023" {
  count = local.enable_bastion ? 1 : 0

  most_recent = true
  owners      = ["amazon"]

  filter {
    name   = "name"
    values = ["al2023-ami-*-x86_64"]
  }
}

resource "aws_security_group" "bastion" {
  count = local.enable_bastion ? 1 : 0

  name_prefix = "${local.name_prefix}-bastion-"
  description = "Emergency DB access bastion (SSM Session Manager)"
  vpc_id      = module.vpc.vpc_id

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-bastion"
  })

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_iam_role" "bastion_ssm" {
  count = local.enable_bastion ? 1 : 0

  name_prefix = "${local.name_prefix}-bastion-ssm-"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = {
        Service = "ec2.amazonaws.com"
      }
      Action = "sts:AssumeRole"
    }]
  })

  tags = local.common_tags
}

resource "aws_iam_role_policy_attachment" "bastion_ssm" {
  count = local.enable_bastion ? 1 : 0

  role       = aws_iam_role.bastion_ssm[0].name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

resource "aws_iam_instance_profile" "bastion" {
  count = local.enable_bastion ? 1 : 0

  name_prefix = "${local.name_prefix}-bastion-"
  role        = aws_iam_role.bastion_ssm[0].name
}

resource "aws_instance" "bastion" {
  count = local.enable_bastion ? 1 : 0

  ami                    = data.aws_ami.amazon_linux_2023[0].id
  instance_type          = "t3.micro"
  subnet_id              = module.vpc.public_subnets[0]
  vpc_security_group_ids = [aws_security_group.bastion[0].id]
  iam_instance_profile   = aws_iam_instance_profile.bastion[0].name

  metadata_options {
    http_endpoint = "enabled"
    http_tokens   = "required"
  }

  root_block_device {
    encrypted = true
  }

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-bastion"
  })
}

resource "aws_security_group_rule" "rds_from_bastion" {
  count = local.enable_bastion ? 1 : 0

  type                     = "ingress"
  description              = "PostgreSQL from bastion (emergency access)"
  from_port                = 5432
  to_port                  = 5432
  protocol                 = "tcp"
  security_group_id        = aws_security_group.rds.id
  source_security_group_id = aws_security_group.bastion[0].id
}
