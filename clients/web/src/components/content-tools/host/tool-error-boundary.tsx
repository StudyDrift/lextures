import { Component, type ErrorInfo, type ReactNode } from 'react'

type ToolErrorBoundaryProps = {
  children: ReactNode
  title: string
  retryLabel: string
  onRetry?: () => void
}

type ToolErrorBoundaryState = {
  hasError: boolean
}

export class ToolErrorBoundary extends Component<ToolErrorBoundaryProps, ToolErrorBoundaryState> {
  state: ToolErrorBoundaryState = { hasError: false }

  static getDerivedStateFromError(): ToolErrorBoundaryState {
    return { hasError: true }
  }

  componentDidCatch(error: Error, info: ErrorInfo): void {
    console.error('Content tool renderer error', error, info.componentStack)
  }

  private handleRetry = () => {
    this.setState({ hasError: false })
    this.props.onRetry?.()
  }

  render() {
    if (this.state.hasError) {
      return (
        <div
          role="alert"
          className="rounded-md border border-rose-200 bg-rose-50 px-3 py-3 text-sm text-rose-900 dark:border-rose-900/50 dark:bg-rose-950/40 dark:text-rose-100"
        >
          <p className="font-medium">{this.props.title}</p>
          <button
            type="button"
            onClick={this.handleRetry}
            className="mt-2 rounded-md bg-rose-800 px-3 py-1.5 text-xs font-medium text-white hover:bg-rose-700 dark:bg-rose-200 dark:text-rose-950 dark:hover:bg-rose-100"
          >
            {this.props.retryLabel}
          </button>
        </div>
      )
    }
    return this.props.children
  }
}
