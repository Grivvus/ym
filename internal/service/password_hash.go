package service

import (
	"github.com/Grivvus/ym/internal/repository"
	"github.com/Grivvus/ym/internal/utils"
)

func repositoryPasswordHashParams(params utils.PasswordHashParams) repository.PasswordHashParams {
	return repository.PasswordHashParams{
		Memory:      int32(params.Memory),
		Iterations:  int32(params.Iterations),
		Parallelism: int32(params.Parallelism),
		KeyLength:   int32(params.KeyLength),
	}
}

func utilsPasswordHashParams(params repository.PasswordHashParams) utils.PasswordHashParams {
	return utils.PasswordHashParams{
		Memory:      uint32(params.Memory),
		Iterations:  uint32(params.Iterations),
		Parallelism: uint8(params.Parallelism),
		KeyLength:   uint32(params.KeyLength),
	}
}
