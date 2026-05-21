import { LegalDocumentPage } from '../components/legal/legal-document-page'
import { TERMS_OF_SERVICE } from '../lib/legal-documents'

export default function TermsOfUsePage() {
  return <LegalDocumentPage document={TERMS_OF_SERVICE} />
}
