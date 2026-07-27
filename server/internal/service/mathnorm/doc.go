// Package mathnorm provides a bounded algebraic expression normaliser for
// Content Tools (CT.18) and any future math-checking surface.
//
// Supported subset (polynomial / rational over declared variables):
//   - Numbers (integers and decimals)
//   - Declared variable identifiers
//   - Operators: +, -, *, /, ^ (integer exponents 0..8)
//   - Parentheses and implicit multiplication (e.g. 3x, 3(x+2), (x+1)(x-1))
//
// Comparison expands products, collects like terms, and compares coefficient
// maps. When the input cannot be decided safely (unsupported constructs,
// oversized AST, division by zero polynomial, non-integer exponent),
// Compare returns OutcomeUndecidable so callers can fall back to an
// author-supplied accepted-answer list or mark needs_review.
//
// Security: no eval; depth/size/token limits; pure AST rewrite.
package mathnorm
