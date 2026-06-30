locals {
  # S3 website endpoint format (not the REST endpoint — website config lives here)
  s3_website_origin = "${aws_s3_bucket.frontend.bucket}.s3-website-${var.region}.amazonaws.com"
}

resource "aws_cloudfront_distribution" "frontend" {
  origin {
    origin_id   = "s3-website"
    domain_name = local.s3_website_origin

    # S3 website endpoints are HTTP-only; CloudFront still serves HTTPS to users
    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "http-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  enabled             = true
  default_root_object = "index.html"
  aliases             = ["haircuts.1136mpco.com"]

  default_cache_behavior {
    target_origin_id       = "s3-website"
    allowed_methods        = ["GET", "HEAD", "OPTIONS"]
    cached_methods         = ["GET", "HEAD"]
    viewer_protocol_policy = "redirect-to-https"
    compress               = true

    forwarded_values {
      query_string = false
      cookies {
        forward = "none"
      }
    }

    min_ttl     = 0
    default_ttl = 300
    max_ttl     = 3600
  }

  # Return index.html for unknown paths (supports single-page app navigation)
  custom_error_response {
    error_code         = 404
    response_code      = 200
    response_page_path = "/index.html"
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    acm_certificate_arn      = aws_acm_certificate_validation.haircuts.certificate_arn
    ssl_support_method       = "sni-only"
    minimum_protocol_version = "TLSv1.2_2021"
  }
}
