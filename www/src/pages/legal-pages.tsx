import { LegalDocumentPage } from './legal-document-page'
import { PRIVACY_POLICY, TERMS_OF_SERVICE } from '../lib/legal-documents'

export function PrivacyPolicyPage() {
  return <LegalDocumentPage document={PRIVACY_POLICY} />
}

export function PrivacyPolicyHistoryPage() {
  return <LegalDocumentPage document={PRIVACY_POLICY} showHistory />
}

export function TermsOfServicePage() {
  return <LegalDocumentPage document={TERMS_OF_SERVICE} />
}

export function TermsOfServiceHistoryPage() {
  return <LegalDocumentPage document={TERMS_OF_SERVICE} showHistory />
}
