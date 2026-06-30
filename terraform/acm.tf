data "aws_route53_zone" "main" {
  name = "1136mpco.com"
}

resource "aws_acm_certificate" "haircuts" {
  provider          = aws.us_east_1
  domain_name       = "haircuts.1136mpco.com"
  validation_method = "DNS"

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_route53_record" "cert_validation" {
  for_each = {
    for dvo in aws_acm_certificate.haircuts.domain_validation_options : dvo.domain_name => {
      name   = dvo.resource_record_name
      type   = dvo.resource_record_type
      record = dvo.resource_record_value
    }
  }

  zone_id = data.aws_route53_zone.main.zone_id
  name    = each.value.name
  type    = each.value.type
  ttl     = 60
  records = [each.value.record]
}

resource "aws_acm_certificate_validation" "haircuts" {
  provider                = aws.us_east_1
  certificate_arn         = aws_acm_certificate.haircuts.arn
  validation_record_fqdns = [for r in aws_route53_record.cert_validation : r.fqdn]
}
