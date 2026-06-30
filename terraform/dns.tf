resource "aws_route53_record" "haircuts" {
  zone_id = data.aws_route53_zone.main.zone_id
  name    = "haircuts.1136mpco.com"
  type    = "A"

  alias {
    name                   = aws_cloudfront_distribution.frontend.domain_name
    zone_id                = aws_cloudfront_distribution.frontend.hosted_zone_id
    evaluate_target_health = false
  }
}
