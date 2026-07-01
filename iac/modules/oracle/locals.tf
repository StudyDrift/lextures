locals {
  name_prefix = "${var.project_name}-${var.environment}"

  common_freeform_tags = merge(
    {
      Project     = var.project_name
      Environment = var.environment
      ManagedBy   = "terraform"
    },
    var.tags,
  )

  availability_domain = coalesce(
    var.availability_domain,
    data.oci_identity_availability_domains.home.availability_domains[0].name,
  )

  image_id = coalesce(
    var.image_id,
    data.oci_core_images.ubuntu.images[0].id,
  )
}
