/**
 * Sandboxed expression evaluator — no eval/Function, allowlisted AST only.
 * Shared by Parameter Explorer (CT.16) and authoring validation.
 */

const MAX_EXPR_LEN = 500
const MAX_AST_DEPTH = 32
const MAX_EVAL_STEPS = 2000

export type EvalErrorCode =
  | 'empty'
  | 'too_long'
  | 'too_deep'
  | 'char'
  | 'number'
  | 'trailing'
  | 'paren'
  | 'syntax'
  | 'unknown_fn'
  | 'unknown_var'
  | 'steps'
  | 'exponent'
  | 'overflow'
  | 'arity'
  | 'internal'

export class SafeExprError extends Error {
  code: EvalErrorCode
  constructor(code: EvalErrorCode, message: string) {
    super(message)
    this.code = code
    this.name = 'SafeExprError'
  }
}

type TokenKind = 'num' | 'ident' | 'op' | 'lparen' | 'rparen' | 'comma' | 'eof'

type Token = { kind: TokenKind; text: string; num?: number }

type AstNode =
  | { kind: 'num'; value: number }
  | { kind: 'ident'; name: string }
  | { kind: 'unary'; op: string; arg: AstNode }
  | { kind: 'binary'; op: string; left: AstNode; right: AstNode }
  | { kind: 'call'; name: string; args: AstNode[] }

const ALLOWED_FNS = new Set([
  'abs',
  'sqrt',
  'sin',
  'cos',
  'tan',
  'asin',
  'acos',
  'atan',
  'ln',
  'log',
  'log10',
  'exp',
  'floor',
  'ceil',
  'round',
  'min',
  'max',
  'pow',
  'hypot',
  'sign',
  'clamp',
])

function tokenize(s: string): Token[] {
  const out: Token[] = []
  let i = 0
  while (i < s.length) {
    const ch = s[i]!
    if (/\s/.test(ch)) {
      i++
      continue
    }
    if (ch === '(') {
      out.push({ kind: 'lparen', text: '(' })
      i++
      continue
    }
    if (ch === ')') {
      out.push({ kind: 'rparen', text: ')' })
      i++
      continue
    }
    if (ch === ',') {
      out.push({ kind: 'comma', text: ',' })
      i++
      continue
    }
    const two = s.slice(i, i + 2)
    if (['>=', '<=', '==', '!=', '&&', '||'].includes(two)) {
      out.push({ kind: 'op', text: two })
      i += 2
      continue
    }
    if ('+-*/%^><!'.includes(ch)) {
      out.push({ kind: 'op', text: ch })
      i++
      continue
    }
    if (/[0-9.]/.test(ch)) {
      let j = i
      let sawDigit = false
      let sawDot = false
      while (j < s.length) {
        const c = s[j]!
        if (/[0-9]/.test(c)) {
          sawDigit = true
          j++
          continue
        }
        if (c === '.' && !sawDot) {
          sawDot = true
          j++
          continue
        }
        if ((c === 'e' || c === 'E') && sawDigit) {
          j++
          if (j < s.length && (s[j] === '+' || s[j] === '-')) j++
          while (j < s.length && /[0-9]/.test(s[j]!)) j++
          break
        }
        break
      }
      const raw = s.slice(i, j)
      const num = Number(raw)
      if (!Number.isFinite(num)) {
        throw new SafeExprError('number', 'invalid number')
      }
      out.push({ kind: 'num', text: raw, num })
      i = j
      continue
    }
    if (/[a-zA-Z_]/.test(ch)) {
      let j = i + 1
      while (j < s.length && /[a-zA-Z0-9_]/.test(s[j]!)) j++
      out.push({ kind: 'ident', text: s.slice(i, j) })
      i = j
      continue
    }
    throw new SafeExprError('char', `disallowed character ${JSON.stringify(ch)}`)
  }
  return out
}

class Parser {
  toks: Token[]
  pos = 0
  constructor(toks: Token[]) {
    this.toks = toks
  }
  peek(): Token {
    return this.toks[this.pos] ?? { kind: 'eof', text: '' }
  }
  next(): Token {
    const t = this.peek()
    if (t.kind !== 'eof') this.pos++
    return t
  }
  parse(): AstNode {
    const n = this.parseOr()
    if (this.peek().kind !== 'eof') {
      throw new SafeExprError('trailing', 'unexpected trailing tokens')
    }
    return n
  }
  parseOr(): AstNode {
    let left = this.parseAnd()
    while (this.peek().kind === 'op' && this.peek().text === '||') {
      this.next()
      const right = this.parseAnd()
      left = { kind: 'binary', op: '||', left, right }
    }
    return left
  }
  parseAnd(): AstNode {
    let left = this.parseCmp()
    while (this.peek().kind === 'op' && this.peek().text === '&&') {
      this.next()
      const right = this.parseCmp()
      left = { kind: 'binary', op: '&&', left, right }
    }
    return left
  }
  parseCmp(): AstNode {
    const left = this.parseSum()
    const t = this.peek()
    if (t.kind === 'op' && ['>', '<', '>=', '<=', '==', '!='].includes(t.text)) {
      this.next()
      const right = this.parseSum()
      return { kind: 'binary', op: t.text, left, right }
    }
    return left
  }
  parseSum(): AstNode {
    let left = this.parseProduct()
    while (this.peek().kind === 'op' && (this.peek().text === '+' || this.peek().text === '-')) {
      const op = this.next().text
      const right = this.parseProduct()
      left = { kind: 'binary', op, left, right }
    }
    return left
  }
  parseProduct(): AstNode {
    let left = this.parsePower()
    while (
      this.peek().kind === 'op' &&
      (this.peek().text === '*' || this.peek().text === '/' || this.peek().text === '%')
    ) {
      const op = this.next().text
      const right = this.parsePower()
      left = { kind: 'binary', op, left, right }
    }
    return left
  }
  parsePower(): AstNode {
    const left = this.parseUnary()
    if (this.peek().kind === 'op' && this.peek().text === '^') {
      this.next()
      const right = this.parsePower()
      return { kind: 'binary', op: '^', left, right }
    }
    return left
  }
  parseUnary(): AstNode {
    if (this.peek().kind === 'op' && ['+', '-', '!'].includes(this.peek().text)) {
      const op = this.next().text
      return { kind: 'unary', op, arg: this.parseUnary() }
    }
    return this.parseCall()
  }
  parseCall(): AstNode {
    if (this.peek().kind === 'ident') {
      const name = this.next().text
      if (this.peek().kind === 'lparen') {
        this.next()
        const args: AstNode[] = []
        if (this.peek().kind !== 'rparen') {
          for (;;) {
            args.push(this.parseOr())
            if (this.peek().kind === 'comma') {
              this.next()
              continue
            }
            break
          }
        }
        if (this.peek().kind !== 'rparen') {
          throw new SafeExprError('paren', 'expected closing parenthesis')
        }
        this.next()
        if (!ALLOWED_FNS.has(name.toLowerCase())) {
          throw new SafeExprError('unknown_fn', `unknown function "${name}"`)
        }
        return { kind: 'call', name, args }
      }
      return { kind: 'ident', name }
    }
    return this.parsePrimary()
  }
  parsePrimary(): AstNode {
    const t = this.peek()
    if (t.kind === 'num') {
      this.next()
      return { kind: 'num', value: t.num ?? 0 }
    }
    if (t.kind === 'lparen') {
      this.next()
      const n = this.parseOr()
      if (this.peek().kind !== 'rparen') {
        throw new SafeExprError('paren', 'expected closing parenthesis')
      }
      this.next()
      return n
    }
    throw new SafeExprError('syntax', "expected number, identifier, or '('")
  }
}

function depth(n: AstNode): number {
  switch (n.kind) {
    case 'num':
    case 'ident':
      return 1
    case 'unary':
      return 1 + depth(n.arg)
    case 'binary':
      return 1 + Math.max(depth(n.left), depth(n.right))
    case 'call':
      return 1 + (n.args.length ? Math.max(...n.args.map(depth)) : 0)
    default: {
      const _exhaustive: never = n
      return _exhaustive
    }
  }
}

function boolNum(b: boolean): number {
  return b ? 1 : 0
}

function callFn(name: string, args: number[]): number {
  const n = name.toLowerCase()
  const need = (k: number) => {
    if (args.length !== k) throw new SafeExprError('arity', `${n} expects ${k} args`)
  }
  switch (n) {
    case 'abs':
      need(1)
      return Math.abs(args[0]!)
    case 'sqrt':
      need(1)
      return args[0]! < 0 ? NaN : Math.sqrt(args[0]!)
    case 'sin':
      need(1)
      return Math.sin(args[0]!)
    case 'cos':
      need(1)
      return Math.cos(args[0]!)
    case 'tan':
      need(1)
      return Math.tan(args[0]!)
    case 'asin':
      need(1)
      return Math.asin(args[0]!)
    case 'acos':
      need(1)
      return Math.acos(args[0]!)
    case 'atan':
      need(1)
      return Math.atan(args[0]!)
    case 'ln':
    case 'log':
      need(1)
      return args[0]! <= 0 ? NaN : Math.log(args[0]!)
    case 'log10':
      need(1)
      return args[0]! <= 0 ? NaN : Math.log10(args[0]!)
    case 'exp':
      need(1)
      if (args[0]! > 700) throw new SafeExprError('overflow', 'exp overflow')
      return Math.exp(args[0]!)
    case 'floor':
      need(1)
      return Math.floor(args[0]!)
    case 'ceil':
      need(1)
      return Math.ceil(args[0]!)
    case 'round':
      need(1)
      return Math.round(args[0]!)
    case 'sign':
      need(1)
      return Math.sign(args[0]!)
    case 'min':
      if (args.length < 1) throw new SafeExprError('arity', 'min expects at least 1 arg')
      return Math.min(...args)
    case 'max':
      if (args.length < 1) throw new SafeExprError('arity', 'max expects at least 1 arg')
      return Math.max(...args)
    case 'pow':
      need(2)
      if (Math.abs(args[1]!) > 1000) throw new SafeExprError('exponent', 'exponent too large')
      return Math.pow(args[0]!, args[1]!)
    case 'hypot':
      need(2)
      return Math.hypot(args[0]!, args[1]!)
    case 'clamp':
      need(3)
      return Math.min(args[2]!, Math.max(args[0]!, args[1]!))
    default:
      throw new SafeExprError('unknown_fn', `unknown function "${name}"`)
  }
}

function evalNode(n: AstNode, env: Record<string, number>, steps: { n: number }): number {
  steps.n++
  if (steps.n > MAX_EVAL_STEPS) {
    throw new SafeExprError('steps', 'expression exceeded evaluation step limit')
  }
  switch (n.kind) {
    case 'num':
      return n.value
    case 'ident': {
      const lower = n.name.toLowerCase()
      if (lower === 'true') return 1
      if (lower === 'false') return 0
      if (Object.prototype.hasOwnProperty.call(env, n.name)) return env[n.name]!
      if (Object.prototype.hasOwnProperty.call(env, lower)) return env[lower]!
      throw new SafeExprError('unknown_var', `unknown variable "${n.name}"`)
    }
    case 'unary': {
      const v = evalNode(n.arg, env, steps)
      if (n.op === '+') return v
      if (n.op === '-') return -v
      if (n.op === '!') return v === 0 ? 1 : 0
      throw new SafeExprError('internal', 'bad unary')
    }
    case 'binary': {
      if (n.op === '&&' || n.op === '||') {
        const l = evalNode(n.left, env, steps)
        if (n.op === '&&' && l === 0) return 0
        if (n.op === '||' && l !== 0) return 1
        const r = evalNode(n.right, env, steps)
        return r !== 0 ? 1 : 0
      }
      const l = evalNode(n.left, env, steps)
      const r = evalNode(n.right, env, steps)
      switch (n.op) {
        case '+':
          return l + r
        case '-':
          return l - r
        case '*':
          return l * r
        case '/':
          return r === 0 ? NaN : l / r
        case '%':
          return r === 0 ? NaN : l % r
        case '^':
          if (Math.abs(r) > 1000) throw new SafeExprError('exponent', 'exponent too large')
          return Math.pow(l, r)
        case '>':
          return boolNum(l > r)
        case '<':
          return boolNum(l < r)
        case '>=':
          return boolNum(l >= r)
        case '<=':
          return boolNum(l <= r)
        case '==':
          return boolNum(l === r)
        case '!=':
          return boolNum(l !== r)
        default:
          throw new SafeExprError('internal', 'bad binary')
      }
    }
    case 'call': {
      const args = n.args.map((a) => evalNode(a, env, steps))
      return callFn(n.name, args)
    }
    default: {
      const _exhaustive: never = n
      return _exhaustive
    }
  }
}

function parseAst(expr: string): AstNode {
  const trimmed = expr.trim()
  if (!trimmed) throw new SafeExprError('empty', 'expression is empty')
  if (trimmed.length > MAX_EXPR_LEN) throw new SafeExprError('too_long', 'expression exceeds length limit')
  const toks = tokenize(trimmed)
  const ast = new Parser(toks).parse()
  if (depth(ast) > MAX_AST_DEPTH) throw new SafeExprError('too_deep', 'expression nesting too deep')
  return ast
}

/** Validate without evaluating. */
export function validateExpression(expr: string): { ok: true } | { ok: false; code: EvalErrorCode; message: string } {
  try {
    parseAst(expr)
    return { ok: true }
  } catch (e) {
    if (e instanceof SafeExprError) return { ok: false, code: e.code, message: e.message }
    return { ok: false, code: 'internal', message: 'invalid expression' }
  }
}

/** Evaluate arithmetic / predicate expression. Returns 0/1 for comparisons. */
export function evalExpression(expr: string, vars: Record<string, number> = {}): number {
  const ast = parseAst(expr)
  const env: Record<string, number> = { pi: Math.PI, e: Math.E, ...vars }
  return evalNode(ast, env, { n: 0 })
}

/** Predicate helper. */
export function evalPredicate(expr: string, vars: Record<string, number> = {}): boolean {
  const v = evalExpression(expr, vars)
  return v !== 0 && !Number.isNaN(v)
}
