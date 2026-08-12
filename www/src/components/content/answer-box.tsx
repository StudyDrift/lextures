import type { ReactNode } from 'react'
export function AnswerBox({ children }: { children: ReactNode }) { return <section className="content-card content-answer" aria-label="Direct answer">{children}</section> }
