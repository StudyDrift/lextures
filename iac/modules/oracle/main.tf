resource "oci_core_vcn" "app" {
  compartment_id = var.compartment_id
  cidr_blocks    = ["10.0.0.0/16"]
  display_name   = "${local.name_prefix}-vcn"
  dns_label      = "lexvcn"

  freeform_tags = local.common_freeform_tags
}

resource "oci_core_internet_gateway" "app" {
  compartment_id = var.compartment_id
  vcn_id         = oci_core_vcn.app.id
  display_name   = "${local.name_prefix}-igw"
  enabled        = true

  freeform_tags = local.common_freeform_tags
}

resource "oci_core_route_table" "public" {
  compartment_id = var.compartment_id
  vcn_id         = oci_core_vcn.app.id
  display_name   = "${local.name_prefix}-public-rt"

  route_rules {
    network_entity_id = oci_core_internet_gateway.app.id
    destination       = "0.0.0.0/0"
    destination_type  = "CIDR_BLOCK"
  }

  freeform_tags = local.common_freeform_tags
}

resource "oci_core_security_list" "public" {
  compartment_id = var.compartment_id
  vcn_id         = oci_core_vcn.app.id
  display_name   = "${local.name_prefix}-public-sl"

  ingress_security_rules {
    protocol = "6"
    source   = "0.0.0.0/0"

    tcp_options {
      min = 22
      max = 22
    }
  }

  ingress_security_rules {
    protocol = "6"
    source   = "0.0.0.0/0"

    tcp_options {
      min = 80
      max = 80
    }
  }

  ingress_security_rules {
    protocol = "6"
    source   = "0.0.0.0/0"

    tcp_options {
      min = 443
      max = 443
    }
  }

  egress_security_rules {
    protocol    = "all"
    destination = "0.0.0.0/0"
  }

  freeform_tags = local.common_freeform_tags
}

resource "oci_core_subnet" "public" {
  compartment_id    = var.compartment_id
  vcn_id            = oci_core_vcn.app.id
  cidr_block        = "10.0.1.0/24"
  display_name      = "${local.name_prefix}-public"
  dns_label         = "public"
  route_table_id    = oci_core_route_table.public.id
  security_list_ids = [oci_core_security_list.public.id]
  # Reserved public IPs (oci_core_public_ip) require public IPs allowed on the subnet VNIC.
  prohibit_public_ip_on_vnic = false

  freeform_tags = local.common_freeform_tags
}

resource "tls_private_key" "app" {
  algorithm = "ED25519"
}

resource "oci_core_volume" "data" {
  availability_domain = local.availability_domain
  compartment_id      = var.compartment_id
  display_name        = "${local.name_prefix}-data"
  size_in_gbs         = var.data_volume_size_gb

  freeform_tags = local.common_freeform_tags
}

resource "oci_core_instance" "app" {
  availability_domain = local.availability_domain
  compartment_id      = var.compartment_id
  display_name        = local.name_prefix
  shape               = var.instance_shape

  shape_config {
    ocpus         = var.instance_ocpus
    memory_in_gbs = var.instance_memory_gbs
  }

  source_details {
    source_type             = "image"
    source_id               = local.image_id
    boot_volume_size_in_gbs = var.boot_volume_size_gb
  }

  create_vnic_details {
    subnet_id        = oci_core_subnet.public.id
    assign_public_ip = false
    display_name     = "${local.name_prefix}-vnic"
  }

  metadata = {
    ssh_authorized_keys = tls_private_key.app.public_key_openssh
    user_data           = base64encode(templatefile("${path.module}/cloud-init.yaml.tftpl", local.cloud_init_vars))
  }

  freeform_tags = local.common_freeform_tags

  lifecycle {
    replace_triggered_by = [
      terraform_data.cloud_init_revision,
    ]
  }
}

resource "oci_core_volume_attachment" "data" {
  attachment_type = "paravirtualized"
  compartment_id  = var.compartment_id
  instance_id     = oci_core_instance.app.id
  volume_id       = oci_core_volume.data.id
}

data "oci_core_vnic_attachments" "app" {
  compartment_id = var.compartment_id
  instance_id    = oci_core_instance.app.id
}

data "oci_core_vnic" "app" {
  vnic_id = data.oci_core_vnic_attachments.app.vnic_attachments[0].vnic_id
}

data "oci_core_private_ips" "app" {
  vnic_id = data.oci_core_vnic.app.id
}

resource "oci_core_public_ip" "app" {
  compartment_id = var.compartment_id
  display_name   = "${local.name_prefix}-ip"
  lifetime       = "RESERVED"
  private_ip_id  = data.oci_core_private_ips.app.private_ips[0].id

  freeform_tags = local.common_freeform_tags
}
