output "public_ip" {
  description = "Static IP address of the Lightsail instance"
  value       = aws_lightsail_static_ip.app.ip_address
}

output "ssh_command" {
  description = "SSH command for the instance"
  value       = "ssh ec2-user@${aws_lightsail_static_ip.app.ip_address}"
}

output "deploy_command" {
  description = "Makefile command to deploy the binary"
  value       = "make deploy HOST=${aws_lightsail_static_ip.app.ip_address}"
}
