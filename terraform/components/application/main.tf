resource "aws_lightsail_instance" "app" {
  name              = "recipe-app-${var.environment}"
  availability_zone = "${var.region}a"
  blueprint_id      = "amazon_linux_2023"
  bundle_id         = var.bundle_id
  user_data         = templatefile("${path.module}/userdata.sh", {
    domain = var.domain
  })

  tags = {
    Name        = "recipe-app"
    Environment = var.environment
  }
}

resource "aws_lightsail_static_ip" "app" {
  name = "recipe-app-${var.environment}-ip"
}

resource "aws_lightsail_static_ip_attachment" "app" {
  static_ip_name = aws_lightsail_static_ip.app.name
  instance_name  = aws_lightsail_instance.app.name
}

resource "aws_lightsail_instance_public_ports" "app" {
  instance_name = aws_lightsail_instance.app.name

  port_info {
    protocol  = "tcp"
    from_port = 22
    to_port   = 22
  }

  port_info {
    protocol  = "tcp"
    from_port = 80
    to_port   = 80
  }

  port_info {
    protocol  = "tcp"
    from_port = 443
    to_port   = 443
  }
}

data "aws_route53_zone" "domain" {
  name = var.domain
}

resource "aws_route53_record" "apex" {
  zone_id = data.aws_route53_zone.domain.zone_id
  name    = var.domain
  type    = "A"
  ttl     = 300
  records = [aws_lightsail_static_ip.app.ip_address]
}

resource "aws_route53_record" "www" {
  zone_id = data.aws_route53_zone.domain.zone_id
  name    = "www.${var.domain}"
  type    = "A"
  ttl     = 300
  records = [aws_lightsail_static_ip.app.ip_address]
}
