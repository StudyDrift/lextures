resource "aws_lb" "main" {
  count = var.enable_ecs ? 1 : 0

  name               = "${local.name_prefix}-alb"
  internal           = false
  load_balancer_type = "application"
  security_groups    = [aws_security_group.alb.id]
  subnets            = module.vpc.public_subnets

  enable_deletion_protection = var.environment == "production"

  tags = {
    Name = "${local.name_prefix}-alb"
  }
}

resource "aws_lb_target_group" "api" {
  count = local.use_api_container ? 1 : 0

  name        = "${local.name_prefix}-api"
  port        = 8080
  protocol    = "HTTP"
  vpc_id      = module.vpc.vpc_id
  target_type = "ip"

  health_check {
    enabled             = true
    path                = "/health"
    healthy_threshold   = 2
    unhealthy_threshold = 3
    timeout             = 5
    interval            = 30
    matcher             = "200"
  }

  tags = {
    Name = "${local.name_prefix}-api"
  }
}

resource "aws_lb_target_group" "web" {
  count = local.use_web_container ? 1 : 0

  name        = "${local.name_prefix}-web"
  port        = 80
  protocol    = "HTTP"
  vpc_id      = module.vpc.vpc_id
  target_type = "ip"

  # nginx serves index.html for SPA routes; do not use /health (image proxies that to Docker hostname "server").
  health_check {
    enabled             = true
    path                = "/"
    healthy_threshold   = 2
    unhealthy_threshold = 3
    timeout             = 5
    interval            = 30
    matcher             = "200"
  }

  tags = {
    Name = "${local.name_prefix}-web"
  }
}

# Default: web container when present, otherwise API-only.
resource "aws_lb_listener" "http" {
  count = local.use_web_container || local.use_api_container ? 1 : 0

  load_balancer_arn = aws_lb.main[0].arn
  port              = 80
  protocol          = "HTTP"

  default_action {
    type = "forward"
    target_group_arn = (
      local.use_web_container
      ? aws_lb_target_group.web[0].arn
      : aws_lb_target_group.api[0].arn
    )
  }
}

# When the SPA is on Fargate, route API/health/tus to the Go service; everything else stays on web.
resource "aws_lb_listener_rule" "api_paths" {
  for_each = local.use_web_container && local.use_api_container ? toset(local.api_path_patterns) : toset([])

  listener_arn = aws_lb_listener.http[0].arn
  priority     = 10 + index(local.api_path_patterns, each.value)

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.api[0].arn
  }

  condition {
    path_pattern {
      values = [each.value]
    }
  }
}
