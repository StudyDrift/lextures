package kernel

// Incremental adoption leaves some toolkit helpers unused until later
// endpoints convert. Referencing them here keeps the public contract
// compile-checked and reachable for the TD.4 deadcode ratchet.
var _ = []any{
	PUT[None, struct{}],
	PATCH[None, struct{}],
	DELETE[struct{}],
	InputType[None, struct{}],
	OutputType[None, struct{}],
	Unauthorized,
	Conflict,
	Public,
	RequireCourseAccess,
	RegisteredRoutes,
	ResetRegistryForTest,
	UnguardedCount,
	EncodeCreated,
	EncodeOK,
	CollectValidation,
	FieldError,
	Validation,
}
