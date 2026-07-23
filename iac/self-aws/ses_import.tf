# One-shot: adopt existing SES domain created in the console.
# Remove this file after a successful apply that imports the identity.
import {
  to = aws_sesv2_email_identity.domain[0]
  id = "lextures.com"
}
