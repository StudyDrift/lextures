package discussions

import "database/sql"

// authorColumns are display name and avatar from the "user".users join aliased as au.
const authorColumns = `NULLIF(TRIM(au.display_name), ''), NULLIF(TRIM(au.avatar_url), '')`

const threadAuthorJoin = `LEFT JOIN "user".users au ON au.id = t.author_id`

const postAuthorJoin = `LEFT JOIN "user".users au ON au.id = p.author_id`

func authorFromNulls(name, avatar sql.NullString) (displayName, avatarURL *string) {
	if name.Valid {
		s := name.String
		displayName = &s
	}
	if avatar.Valid {
		s := avatar.String
		avatarURL = &s
	}
	return displayName, avatarURL
}
