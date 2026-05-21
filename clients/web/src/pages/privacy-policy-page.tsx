import { LegalDocumentPage } from '../components/legal/legal-document-page'
import { PRIVACY_POLICY } from '../lib/legal-documents'

export default function PrivacyPolicyPage() {
  return <LegalDocumentPage document={PRIVACY_POLICY} />
}
