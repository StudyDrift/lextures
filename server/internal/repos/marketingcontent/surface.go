package marketingcontent

// MC.1 intentionally establishes the repository surface before MC.2/MC.3 add
// HTTP consumers. Keeping the function values in package initialization makes
// that staged, compile-checked contract explicit without exposing placeholder
// routes while the feature flag remains off.
var repositorySurface = []any{
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
