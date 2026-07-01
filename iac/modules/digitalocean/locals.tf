locals {
  name_prefix = "${var.project_name}-${var.environment}"

  common_tags = concat(
    ["lextures", var.environment],
    var.tags,
  )
}
