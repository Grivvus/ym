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

type ErrAlreadyExists struct {
	identifier any
	entityName string
}

func NewErrAlreadyExists(entity string, identifier any) ErrAlreadyExists {
	return ErrAlreadyExists{
		identifier: identifier,
		entityName: entity,
	}
}

func (e ErrAlreadyExists) Error() string {
	return fmt.Sprintf("Already exists %v with identifier: %v", e.entityName, e.identifier)
}

var ErrUnknownDBError = fmt.Errorf("unknown database error")

var ErrBadParams = fmt.Errorf("bad params")
