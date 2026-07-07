import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'

export default function CheckoutCancelPage() {
  const { t } = useTranslation('billing')

  return (
    <main className="mx-auto flex min-h-screen max-w-lg flex-col items-center justify-center px-4 text-center">
      <h1 className="text-2xl font-semibold">{t('billing.checkout.cancel.title')}</h1>
      <p className="mt-2 text-slate-600 dark:text-neutral-400">{t('billing.checkout.cancel.description')}</p>
      <Link
        to="/courses"
        className="mt-6 inline-flex rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium hover:bg-slate-50 dark:border-neutral-700 dark:hover:bg-neutral-900"
      >
        {t('billing.checkout.cancel.backToCourses')}
      </Link>
    </main>
  )
}
