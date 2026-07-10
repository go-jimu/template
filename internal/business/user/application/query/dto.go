package query

// User is the read-side representation returned by User queries.
type User struct {
	ID    string
	Name  string
	Email string
}

// UserPage is a page of User read models.
type UserPage struct {
	Users    []User
	Total    int
	Page     int
	PageSize int
}
