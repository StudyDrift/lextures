/**
 * W03 — forbid window.alert/confirm/prompt (and globalThis/bare variants) in app UI code.
 */

const NATIVE_DIALOG_CALLEES = new Set(['alert', 'confirm', 'prompt'])

function isGlobalBuiltin(context, node, name) {
  const { scopeManager } = context.sourceCode
  if (!scopeManager) return true
  let scope = scopeManager.acquire(node)
  if (!scope) return true
  while (scope) {
    const variable = scope.variables.find((v) => v.name === name)
    if (variable) {
      return variable.defs.some((def) => def.type === 'ImportBinding' || def.type === 'Variable')
        ? false
        : variable.defs.some((def) => def.type === 'FunctionName' || def.type === 'Parameter')
    }
    scope = scope.upper
  }
  return true
}

/** Hook replacements pass an options object; native dialogs take a string message. */
function hasNativeDialogCallShape(node) {
  const firstArg = node.arguments[0]
  if (!firstArg) return true
  if (firstArg.type === 'Literal' && typeof firstArg.value === 'string') return true
  if (firstArg.type === 'TemplateLiteral') return true
  return false
}

function isNativeDialogCall(context, node) {
  if (node.type !== 'CallExpression') return false
  const callee = node.callee
  if (callee.type === 'Identifier' && NATIVE_DIALOG_CALLEES.has(callee.name)) {
    if (!hasNativeDialogCallShape(node)) return false
    return isGlobalBuiltin(context, node, callee.name)
  }
  if (
    callee.type === 'MemberExpression' &&
    !callee.computed &&
    callee.property.type === 'Identifier' &&
    NATIVE_DIALOG_CALLEES.has(callee.property.name)
  ) {
    const obj = callee.object
    if (obj.type === 'Identifier' && (obj.name === 'window' || obj.name === 'globalThis')) {
      return true
    }
  }
  return false
}

export default {
  meta: {
    type: 'problem',
    docs: {
      description: 'Disallow native window.alert/confirm/prompt in pages and components (W03)',
    },
    schema: [],
    messages: {
      nativeDialog:
        'Use toast (sonner) or ConfirmDialog/InputDialog instead of native {{name}}().',
    },
  },
  create(context) {
    return {
      CallExpression(node) {
        if (!isNativeDialogCall(context, node)) return
        const callee = node.callee
        const name =
          callee.type === 'Identifier'
            ? callee.name
            : callee.type === 'MemberExpression' && callee.property.type === 'Identifier'
              ? callee.property.name
              : 'dialog'
        context.report({ node, messageId: 'nativeDialog', data: { name } })
      },
    }
  },
}
