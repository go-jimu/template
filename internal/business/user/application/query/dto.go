package query

// User is the read-side representation returned by User queries.
type User struct {
	ID    string
	Name  string
	Email string
}

// UserPage is one page of Users, newest first.
type UserPage struct {
	Users []User
	// NextCursor continues after the last User of this page; empty when no
	// more Users match.
	NextCursor string
}
