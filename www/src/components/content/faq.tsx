import type { ReactNode } from 'react'
export type FAQItem = { question: string; answer: ReactNode }
export function FAQ({ items }: { items: FAQItem[] }) { return <section className="content-faq" data-faq><h2>Frequently asked questions</h2>{items.map(item => <details className="content-faq-item" open key={item.question}><summary>{item.question}</summary><div>{item.answer}</div></details>)}</section> }
