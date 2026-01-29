package service

import "fmt"

type ErrNotFound struct {
	identifier any
	entityName string
}

func NewErrNotFound(entity string, identifier any) ErrNotFound {
	return ErrNotFound{
		identifier: identifier,
		entityName: entity,
	}
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("No such %v with identifier: %v", e.entityName, e.identifier)
}
