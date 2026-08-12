export function ContextualLinks({ kind }: { kind: 'blog' | 'docs' }) {
  return <p data-contextual-links className="mt-10 leading-relaxed">
    Continue with the <a href={kind === 'docs' ? '/docs' : '/resources'}>{kind === 'docs' ? 'Lextures documentation' : 'learning resources hub'}</a>,
    see how these ideas connect to <a href="/platform/adaptive-learning">adaptive learning on the Lextures platform</a>,
    and review <a href="/pricing">Lextures pricing and deployment options</a>.
  </p>
}
