package repository

import "github.com/Grivvus/ym/internal/db"

type PasswordHashParams struct {
	Memory      int32
	Iterations  int32
	Parallelism int32
	KeyLength   int32
}

func passwordHashParamsFromDBUser(user db.User) PasswordHashParams {
	return PasswordHashParams{
		Memory:      user.PasswordMemory,
		Iterations:  user.PasswordIterations,
		Parallelism: user.PasswordParallelism,
		KeyLength:   user.PasswordKeyLength,
	}
}

func passwordHashParamsFromUserByUsernameRow(user db.GetUserByUsernameRow) PasswordHashParams {
	return PasswordHashParams{
		Memory:      user.PasswordMemory,
		Iterations:  user.PasswordIterations,
		Parallelism: user.PasswordParallelism,
		KeyLength:   user.PasswordKeyLength,
	}
}

func passwordHashParamsFromUserByIDRow(user db.GetUserByIDRow) PasswordHashParams {
	return PasswordHashParams{
		Memory:      user.PasswordMemory,
		Iterations:  user.PasswordIterations,
		Parallelism: user.PasswordParallelism,
		KeyLength:   user.PasswordKeyLength,
	}
}
