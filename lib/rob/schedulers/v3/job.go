package schedulers

import "github.com/google/uuid"

type Job struct {
	User  *User
	Ready chan<- uuid.UUID
}
