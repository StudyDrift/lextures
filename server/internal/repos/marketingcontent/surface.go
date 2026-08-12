package marketingcontent

// MC.1 establishes the repository surface before later plans add every HTTP
// consumer. Referencing the functions in a blank package-level value keeps the
// staged contract compile-checked and reachable for deadcode without an unused
// named variable (golangci unused).
var _ = []any{
	ListArticles,
	GetArticleByID,
	GetArticleByPath,
	InsertArticle,
	UpdateArticle,
	SoftDeleteArticle,
	ListRevisions,
	GetRevision,
	UpsertCategory,
	ListCategories,
	UpsertAuthor,
	ListAuthors,
	InsertRedirect,
	ListRedirects,
	DeleteRedirect,
	UpsertTag,
	ListTags,
	DeleteTag,
	DeleteCategory,
}
